package eemqtt

import "crypto/tls"

type Option func(c *Container)

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
