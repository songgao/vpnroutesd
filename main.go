package main

import (
	"log"
	"net"
	"os"

	"github.com/songgao/vpnroutesd/routing"
)

func main() {
	ifceNamePrimary := os.Args[1]
	ifceNameVPN := os.Args[2]

	if err := routing.ApplyRoutes(routing.ApplyRoutesArgs{
		IfceNamePrimary: ifceNamePrimary,
		IfceNameVPN:     ifceNameVPN,
		VPNIPs:          []net.IP{{8, 8, 8, 8}, {8, 8, 4, 4}, {18, 214, 166, 21}},
	}); err != nil {
		log.Fatalf("applyRoutes error: %v", err)
	}
}
