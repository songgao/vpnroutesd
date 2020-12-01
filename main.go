package main

import (
	"log"
	"net"
	"os"
)

func main() {
	ifceNamePrimary := os.Args[1]
	ifceNameVPN := os.Args[2]

	if err := applyRoutes(applyRoutesArgs{
		ifceNamePrimary: ifceNamePrimary,
		ifceNameVPN:     ifceNameVPN,
		vpnIPs:          []net.IP{{8, 8, 8, 8}, {8, 8, 4, 4}, {18, 214, 166, 21}},
	}); err != nil {
		log.Fatalf("applyRoutes error: %v", err)
	}
}
