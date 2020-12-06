package main

import (
	"log"
	"net"
	"os"

	"github.com/songgao/vpnroutesd/sys"
	"go.uber.org/zap"
)

func main() {
	var options []zap.Option
	// options = append(options, zap.IncreaseLevel(zap.InfoLevel))
	logger, err := zap.NewDevelopment(options...)
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	logger.Info("Init")
	defer logger.Info("Done")

	args := sys.ApplyRoutesArgs{
		VPNIPs: []net.IP{{8, 8, 8, 8}, {8, 8, 4, 4}, {18, 214, 166, 21}},
	}

	if len(os.Args) == 3 {
		args.Interfaces = &sys.InterfaceNames{
			Primary: os.Args[1],
			VPN:     os.Args[2],
		}
	}

	if err := sys.ApplyRoutes(logger, args); err != nil {
		log.Fatalf("ApplyRoutes error: %v", err)
	}
}
