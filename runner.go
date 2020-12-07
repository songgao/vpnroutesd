package main

import (
	"net"

	"github.com/songgao/vpnroutesd/config"
	"github.com/songgao/vpnroutesd/dns"
	"github.com/songgao/vpnroutesd/sys"
	"go.uber.org/zap"
)

type runResult struct {
	config string
	dns    string
	routes string
}

func dedupIPs(ips ...[]net.IP) []net.IP {
	m := make(map[string]net.IP)
	for _, l := range ips {
		for _, ip := range l {
			m[ip.String()] = ip
		}
	}
	ret := make([]net.IP, 0, len(m))
	for _, ip := range m {
		ret = append(ret, ip)
	}
	return ret
}

func run(logger *zap.Logger) (result runResult) {
	logger.Debug("+ run")
	defer logger.Debug("- run")

	cfg, cfgChanged, err := config.Load(logger, *fConfig)
	if err != nil {
		logger.Sugar().Errorf("loading config error: %v", err.Error())
		result.config = "ERR"
		return
	}
	if cfgChanged {
		result.config = "CHANGE DETECTED"
	} else {
		result.config = "UNCHANGED"
	}
	logger.Sugar().Debugf("using config: %s", cfg)

	domainIPs, dnsChanged, err := dns.GetIPs(logger, cfg.DNSServer, cfg.VPNDomains)
	if err != nil {
		logger.Sugar().Errorf("dns.GetIPs error: %v", err)
		result.dns = "ERR"
		return result
	}
	if dnsChanged {
		result.dns = "CHANGED"
	} else {
		result.dns = "UNCHANGED"
	}
	logger.Sugar().Debugf("IPs from DNS: %s", domainIPs)

	args := sys.ApplyRoutesArgs{
		VPNIPs: dedupIPs(cfg.VPNIPs, domainIPs),
	}

	if len(*fPrimaryIfce) > 0 && len(*fVPNIfce) > 0 {
		args.Interfaces = &sys.InterfaceNames{
			Primary: *fPrimaryIfce,
			VPN:     *fVPNIfce,
		}
	}

	routesChanged, err := sys.ApplyRoutes(logger, args)
	if err != nil {
		logger.Sugar().Errorf("ApplyRoutes error: %v", err)
		result.routes = "ERR"
		return result
	}
	if routesChanged {
		result.routes = "CHANGED"
	} else {
		result.routes = "UNCHANGED"
	}

	return result
}
