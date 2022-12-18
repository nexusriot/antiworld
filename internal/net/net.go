package net

import (
	"context"
	"net"
	"net/http"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"

	cf "github.com/nexusriot/antiworld/internal/config"
)

type Net struct {
	Client      *http.Client
	ProxyConfig *cf.Proxy
}

func NewNet(proxyConfig *cf.Proxy) *Net {
	var client *http.Client

	if proxyConfig != nil {
		proxyUrl := proxyConfig.Address
		auth := proxy.Auth{
			User:     proxyConfig.Username,
			Password: proxyConfig.Password,
		}
		dialer, err := proxy.SOCKS5("tcp", proxyUrl, &auth, proxy.Direct)
		if err != nil {
			log.Fatalf("Failed to create dialer %s", err.Error())
		}
		dialContext := func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialer.Dial(network, address)
		}
		transport := &http.Transport{DialContext: dialContext,
			DisableKeepAlives: true}
		client = &http.Client{Transport: transport}
	} else {
		client = &http.Client{}
	}

	return &Net{
		Client:      client,
		ProxyConfig: proxyConfig,
	}
}
