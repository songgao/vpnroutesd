package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

var fVerbose = pflag.BoolP("verbose", "v", false, "[optional] turn on debug logging")
var fInterval = pflag.Uint64("interval", 60, "[optional] interval in seconds to do stuff. default is 60")
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

	ticker := time.NewTicker(time.Duration(*fInterval) * time.Second)
	first := make(chan struct{}, 1)
	first <- struct{}{}
	for {
		select {
		case <-ticker.C:
		case <-first:
		}
		results := run(logger)
		logger.Sugar().Infof("Iteration: config [%s]; dns [%s]; routes [%s]", results.config, results.dns, results.routes)
	}
}
