package eemqtt

import (
	"crypto/tls"
	"github.com/eclipse/paho.golang/paho"
)

type Option func(c *Container)

//WithWillMessage  使用遗嘱消息
func WithWillMessage(topic string, payload []byte, qos byte, retain bool) Option {
	return func(c *Container) {
		c.config.enableWillMessage = true
		c.config.willTopic = topic
		c.config.willPayload = payload
		c.config.willQos = qos
		c.config.willRetain = retain
	}
}

func WithConnectPacketConfigurator(fn func(*paho.Connect) *paho.Connect) Option {
	return func(c *Container) {
		c.config.enableConnectPacket = true
		c.config.connectPacketBuilder = fn
	}
}

//WithTLSConfig  自定义tls 验证配置
func WithTLSConfig(tsc *tls.Config) Option {
	return func(c *Container) {
		c.config.EnableTLS = true
		c.config.customizeTlsConfig = tsc
	}
}

func WithTLSSessionCache(tsc tls.ClientSessionCache) Option {
	return func(c *Container) {
		c.config.TLSSessionCache = tsc
	}
}

func WithClientID(clientID string) Option {
	return func(c *Container) {
		c.config.ClientID = clientID
	}
}

func WithUsername(username string) Option {
	return func(c *Container) {
		c.config.Username = username
	}
}

func WithPassword(password string) Option {
	return func(c *Container) {
		c.config.Password = password
	}
}
