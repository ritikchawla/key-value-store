package network

import (
    "net"
    "time"
    "context"
)

type NetworkConfig struct {
    DialTimeout    time.Duration
    MaxRetries     int
    RetryInterval  time.Duration
    KeepAlive      bool
    BufferSize     int
}

func DefaultNetworkConfig() *NetworkConfig {
    return &NetworkConfig{
        DialTimeout:    time.Second * 5,
        MaxRetries:     3,
        RetryInterval:  time.Second,
        KeepAlive:     true,
        BufferSize:    4096,
    }
}

type Transport struct {
    config *NetworkConfig
    dialer *net.Dialer
}

func NewTransport(config *NetworkConfig) *Transport {
    return &Transport{
        config: config,
        dialer: &net.Dialer{
            Timeout:   config.DialTimeout,
            KeepAlive: time.Second * 30,
        },
    }
}