package config

import (
	"fmt"
	"net"

	"github.com/pelletier/go-toml"
	"go.uber.org/zap"
)

type configToml struct {
	VPNRoutes struct {
		Domains []string
		IPs     []string
	}
}

// Config holds config fields for vpnroutesd.
type Config struct {
	VPNDomains []string
	VPNIPs     []net.IP
}

// Load reads and parses a vpnroutesd config file from p, which can be one of
// the following:
//   1. filesystem path, e.g.
//        /home/user/.vpnroutesd.toml
//   2. https URL, e.g.
//        https://internal.4seasontotallandscaping.com/.vpnroutesd.toml
//   3. keybase filesystem path, e.g.
//        keybase@alice://team/4seasontotallandscaping/vpn/.vpnroutesd.toml
//        (alice is the system username, not keybase username)
func Load(logger *zap.Logger, p string) (cfg Config, err error) {
	data, err := readConfig(logger, p)
	if err != nil {
		return Config{}, fmt.Errorf("reading config file error: %v", err)
	}

	var cfgToml configToml
	if err = toml.Unmarshal(data, &cfgToml); err != nil {
		return Config{}, fmt.Errorf("parsing config file error: %v", err)
	}

	cfg.VPNDomains = cfgToml.VPNRoutes.Domains

	for _, ipStr := range cfgToml.VPNRoutes.IPs {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			logger.Sugar().Warnf("ignoring invalid IP: %s", ipStr)
			continue
		}
		if ip.To4() == nil {
			logger.Sugar().Warnf("ignoring non-IPv4 IP: %v", ip)
			continue
		}
		cfg.VPNIPs = append(cfg.VPNIPs, ip)
	}

	return cfg, err
}
