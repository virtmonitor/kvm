package main

import (
	plugin "github.com/hashicorp/go-plugin"
	kvm "github.com/virtmonitor/kvm/driver"
	"github.com/virtmonitor/plugins"
)

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugins.Handshake,
		Plugins: map[string]plugin.Plugin{
			"driver_grpc": &plugins.DriverGrpcPlugin{Impl: &kvm.KVM{}},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
