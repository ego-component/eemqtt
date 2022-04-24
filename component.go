package eemqtt

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/gotomicro/ego/core/elog"
	"github.com/gotomicro/ego/core/emetric"
	"io/ioutil"
	"net/url"
	"sync"
	"time"
)

const PackageName = "component.eemqtt"

// Component ...
type Component struct {
	ServerCtx        context.Context
	stopServer       context.CancelFunc
	name             string
	mod              int //0-初始化  1 运行中
	config           *config
	Brokers          string
	rmu              *sync.RWMutex
	logger           *elog.Component
	ec               *autopaho.ConnectionManager
	onPublishHandler OnPublishHandler
}

func newComponent(name string, config *config, logger *elog.Component) *Component {
	serverCtx, stopServer := context.WithCancel(context.Background())
	cc := &Component{
		ServerCtx:  serverCtx,
		stopServer: stopServer,
		mod:        0,
		name:       name,
		rmu:        &sync.RWMutex{},
		logger:     logger,
		config:     config,
	}
	logger.Info("dial emqtt server")
	return cc
}

//Start 开始启用
func (c *Component) Start(handler OnPublishHandler) {
	c.rmu.RLock()
	if c.mod == 0 {
		c.onPublishHandler = handler
		c.rmu.RUnlock()
		c.connServer()
	} else {
		c.rmu.RUnlock()
		c.logger.Error("client has started")
	}
}

func (c *Component) connServer() {
	urls, brokers := parseUrl(c.config.Brokers)
	if len(urls) == 0 {
		c.logger.Panic("client emqtt brokers empty / error", elog.FieldValueAny(c.config))
	}
	c.Brokers = brokers
	cliCfg := autopaho.ClientConfig{
		BrokerUrls:        urls,
		KeepAlive:         c.config.KeepAlive,
		ConnectRetryDelay: c.config.ConnectRetryDelay,
		ConnectTimeout:    c.config.ConnectTimeout,
		Debug:             paho.NOOPLogger{},
		OnConnectionUp: func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
			c.logger.Info("mqtt connection up")
			sOs := make(map[string]paho.SubscribeOptions)
			for st := range c.config.SubscribeTopics {
				sOs[c.config.SubscribeTopics[st].Topic] = paho.SubscribeOptions{QoS: c.config.SubscribeTopics[st].Qos}
			}
			var err error
			if len(sOs) > 0 {
				if _, err = cm.Subscribe(context.Background(), &paho.Subscribe{
					Subscriptions: sOs,
				}); err != nil {
					c.logger.Error(fmt.Sprintf("failed to subscribe (%v). This is likely to mean no messages will be received.", sOs), elog.FieldErr(err))
				}
			}
			if c.config.EnableMetricInterceptor {
				emetric.ClientHandleCounter.Inc("emqtt", c.name, "Connect", c.Brokers, "OK")
				if len(sOs) > 0 {
					for so := range sOs {
						if err != nil {
							emetric.ClientHandleCounter.Inc("emqtt", c.name, "subscribe_"+so, c.Brokers, "Error")
						} else {
							emetric.ClientHandleCounter.Inc("emqtt", c.name, "subscribe_"+so, c.Brokers, "OK")
						}

					}
				}
			}
		},
		OnConnectError: func(err error) {
			c.logger.Error("error whilst attempting connection", elog.FieldErr(err))
			if c.config.EnableMetricInterceptor {
				emetric.ClientHandleCounter.Inc("emqtt", c.name, "Connect", c.Brokers, "Error")
			}
		},
		ClientConfig: paho.ClientConfig{
			ClientID: c.config.ClientID,
			Router: paho.NewSingleHandlerRouter(func(pp *paho.Publish) {
				if c.onPublishHandler != nil {
					c.onPublishHandler(c.ServerCtx, pp)
				} else {
					c.logger.Warn("Received message, but no handler is defined")
				}
			}),
			OnClientError: func(err error) {
				c.logger.Error("server requested disconnect", elog.FieldErr(err))
				if c.config.EnableMetricInterceptor {
					emetric.ClientHandleCounter.Inc("emqtt", c.name, "Connect", c.Brokers, "Error")
				}
			},
			OnServerDisconnect: func(d *paho.Disconnect) {
				if d.Properties != nil {
					c.logger.Warn(fmt.Sprintf("server requested disconnect: %s\n", d.Properties.ReasonString))
				} else {
					c.logger.Warn(fmt.Sprintf("server requested disconnect; reason code: %d\n", d.ReasonCode))
				}
				if c.config.EnableMetricInterceptor {
					emetric.ClientHandleCounter.Inc("emqtt", c.name, "Connect", c.Brokers, "Error")
				}
			},
		},
	}

	if c.config.Debug {
		cliCfg.Debug = debugLogger{prefix: "emqtt-autoPaho"}
		cliCfg.PahoDebug = debugLogger{prefix: "emqtt-paho"}
	}

	if c.config.Username != "" && c.config.Password != "" {
		cliCfg.SetUsernamePassword(c.config.Username, ([]byte)(c.config.Password))
	}

	if c.config.EnableTLS {
		if c.config.customizeTlsConfig != nil {
			cliCfg.TlsCfg = c.config.customizeTlsConfig
		} else {
			tslConfig, errTLS := c.buildTLSConfig()
			if errTLS != nil {
				c.logger.Panic("build TLSConfig fialed", elog.FieldValueAny(c.config))
			}
			cliCfg.TlsCfg = tslConfig
		}
	}

	if c.config.enableWillMessage {
		cliCfg.SetWillMessage(c.config.willTopic, c.config.willPayload, c.config.willQos, c.config.willRetain)
	}

	if c.config.enableConnectPacket {
		cliCfg.SetConnectPacketConfigurator(c.config.connectPacketBuilder)
	}

	cm, err := autopaho.NewConnection(c.ServerCtx, cliCfg)
	if err != nil {
		c.logger.Panic("emqtt connect fialed", elog.FieldValueAny(c.config))
	} else {
		c.rmu.Lock()
		c.ec = cm
		c.mod = 1
		c.rmu.Unlock()
	}
}

func (c *Component) buildTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{}
	tlsConfig.RootCAs = x509.NewCertPool()
	ca, err := ioutil.ReadFile(c.config.TLSClientCA)
	if err != nil {
		return nil, fmt.Errorf("read client ca fail:%+v", err)
	}
	if !tlsConfig.RootCAs.AppendCertsFromPEM(ca) {
		return nil, fmt.Errorf("append client ca fail:%+v", err)
	}
	//设置了客户端证书
	if c.config.TLSClientCertFile != "" && c.config.TLSClientKeyFile != "" {
		clientCert, err := tls.LoadX509KeyPair(c.config.TLSClientCertFile, c.config.TLSClientKeyFile)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}
	tlsConfig.ClientAuth = c.config.ClientAuthType()
	tlsConfig.ClientSessionCache = c.config.TLSSessionCache
	return tlsConfig, nil
}

func parseUrl(brokers []string) ([]*url.URL, string) {
	var urls []*url.URL
	resBrokers := ""
	for _, val := range brokers {
		if url, err := url.Parse(val); err == nil {
			urls = append(urls, url)
			if resBrokers != "" {
				resBrokers = resBrokers + ","
			}
			resBrokers = resBrokers + val
		} else {
			fmt.Printf("url %s parse error %s", val, err.Error())
		}
	}
	return urls, resBrokers
}

func (c *Component) Client() *autopaho.ConnectionManager {
	return c.ec
}

func (c *Component) PublishMsg(topic string, qos byte, payload interface{}) {
	c.rmu.RLock()
	if c.mod == 0 {
		c.rmu.RUnlock()
		c.logger.Error("client not start")
		return
	}

	err := c.ec.AwaitConnection(c.ServerCtx)
	if err != nil { // Should only happen when context is cancelled
		c.logger.Error(fmt.Sprintf("publisher done (AwaitConnection: %s)", err))
		return
	}

	var msgByte []byte
	switch payload.(type) {
	case string:
		msgByte = []byte(payload.(string))
	case []byte:
		msgByte = payload.([]byte)
	default:
		c.logger.Error("Unknown payload type")
		return
	}

	go func(msg []byte) {
		pr, err := c.ec.Publish(c.ServerCtx, &paho.Publish{
			QoS:     qos,
			Topic:   topic,
			Payload: msgByte,
		})
		if err != nil {
			c.logger.Error(fmt.Sprintf("error publishing: %s\n", err))
		}

		if qos > 0 {
			if pr.ReasonCode != 0 && pr.ReasonCode != 16 { // 16 = Server received message but there are no subscribers
				c.logger.Info(fmt.Sprintf("reason code %d received\n", pr.ReasonCode))
			} else {
				c.logger.Info(fmt.Sprintf("reason code %d publish success ", pr.ReasonCode))
			}
		}
	}(msgByte)
}

func (c *Component) Stop() {
	c.rmu.Lock()
	if c.mod == 1 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = c.ec.Disconnect(ctx)
		c.mod = 0
	}
	c.stopServer()
	c.rmu.Unlock()
}
