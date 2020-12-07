package main

import (
	"fmt"
	"log"
	"os"

	"github.com/songgao/vpnroutesd/config"
	"github.com/songgao/vpnroutesd/sys"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

var fVerbose = pflag.BoolP("verbose", "v", false, "[optional] turn on debug logging")
var fConfig = pflag.StringP("config", "c", "", "[required] path to config file")
var fPrimaryIfce = pflag.StringP("primary-interface", "i", "", "[optional] primary interface name (leave empty to use auto detection)")
var fVPNIfce = pflag.StringP("vpn-interface", "j", "", "[optional] VPN interface name (leave empty to use auto detection)")

func parseFlagsOrBust() {
	pflag.Parse()
	if !pflag.Parsed() {
		pflag.Usage()
		os.Exit(1)
	}
	if (len(*fPrimaryIfce) == 0) != (len(*fVPNIfce) == 0) {
		fmt.Fprintln(os.Stderr, "error: --primary-interface and --vpn-interface must be supplied or omitted together")
		pflag.Usage()
		os.Exit(1)
	}
	if len(*fConfig) == 0 {
		fmt.Fprintln(os.Stderr, "error: --config is required")
		pflag.Usage()
		os.Exit(1)
	}
}

func main() {
	parseFlagsOrBust()

	var options []zap.Option
	if !*fVerbose {
		options = append(options, zap.IncreaseLevel(zap.InfoLevel))
	}
	logger, err := zap.NewDevelopment(options...)
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	logger.Info("Init")
	defer logger.Info("Done")

	cfg, err := config.Load(logger, *fConfig)
	if err != nil {
		logger.Fatal(err.Error())
	}

	logger.Sugar().Debugf("using config: %s", cfg)

	args := sys.ApplyRoutesArgs{
		VPNIPs: cfg.VPNIPs,
	}

	if len(*fPrimaryIfce) > 0 && len(*fVPNIfce) > 0 {
		args.Interfaces = &sys.InterfaceNames{
			Primary: *fPrimaryIfce,
			VPN:     *fVPNIfce,
		}
	}

	if err := sys.ApplyRoutes(logger, args); err != nil {
		log.Fatalf("ApplyRoutes error: %v", err)
	}
}
