package eemqtt

import (
	"crypto/tls"
	"time"
)

type config struct {
	Brokers                 []string                  `json:"brokers" toml:"brokers"`                     //连接地址  / URL(s) for the broker (schemes supported include 'mqtt' and 'tls')
	Username                string                    `json:"username" toml:"username"`                   //用户名
	Password                string                    `json:"password" toml:"password"`                   //密码
	ClientID                string                    `json:"clientID" toml:"clientID"`                   //客户端标识
	KeepAlive               uint16                    `json:"keepAlive" toml:"keepAlive"`                 //默认值 30
	ConnectRetryDelay       time.Duration             `json:"connectRetryDelay" toml:"connectRetryDelay"` //重连时间间隔 default 10s
	ConnectTimeout          time.Duration             `json:"connectTimeout" toml:"connectTimeout"`       //连接超时时间 default 10s
	SubscribeTopics         map[string]subscribeTopic `json:"subscribeTopics" toml:"subscribeTopics"`     //连接后自动订阅主题
	Debug                   bool                      `json:"debug" toml:"debug"`                         // Debug 是否开启debug模式
	EnableTLS               bool                      //启用 tls 方式连接
	TLSClientCA             string                    //client 的 ca 证书
	TLSClientAuth           string                    //客户端认证方式默认为 NoClientCert(NoClientCert,RequestClientCert,RequireAnyClientCert,VerifyClientCertIfGiven,RequireAndVerifyClientCert)
	TLSClientCertFile       string                    //客户端证书
	TLSClientKeyFile        string                    //客户端证书Key
	TLSSessionCache         tls.ClientSessionCache
	EnableMetricInterceptor bool `json:"enableMetricInterceptor" toml:"enableMetricInterceptor"` // 是否开启监控，默认开启
}

//订阅主题
type subscribeTopic struct {
	Topic string `json:"topic" toml:"topic"`
	Qos   byte   `json:"qos" toml:"qos"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *config {
	return &config{
		Debug:             false,
		KeepAlive:         30,
		ConnectRetryDelay: time.Second * 10,
		ConnectTimeout:    time.Second * 10,
		ClientID:          "",
		SubscribeTopics:   make(map[string]subscribeTopic),
	}
}

// ClientAuthType 客户端auth类型
func (config *config) ClientAuthType() tls.ClientAuthType {
	switch config.TLSClientAuth {
	case "NoClientCert":
		return tls.NoClientCert
	case "RequestClientCert":
		return tls.RequestClientCert
	case "RequireAnyClientCert":
		return tls.RequireAnyClientCert
	case "VerifyClientCertIfGiven":
		return tls.VerifyClientCertIfGiven
	case "RequireAndVerifyClientCert":
		return tls.RequireAndVerifyClientCert
	default:
		return tls.NoClientCert
	}
}
