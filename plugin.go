package main

import (
	"os"

	hclog "github.com/hashicorp/go-hclog"
	plugin "github.com/hashicorp/go-plugin"
	kvm "github.com/virtmonitor/kvm/driver"
	"github.com/virtmonitor/plugins"
)

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "KVM",
		Output: os.Stderr,
	})

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugins.Handshake,
		Plugins: map[string]plugin.Plugin{
			"driver_grpc": &plugins.DriverGrpcPlugin{Impl: &kvm.KVM{}},
		},
		GRPCServer: plugin.DefaultGRPCServer,
		Logger:     logger,
	})
}
