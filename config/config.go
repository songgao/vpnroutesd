package config

import (
	"bytes"
	"fmt"
	"net"

	"github.com/pelletier/go-toml"
	"go.uber.org/zap"
)

type configToml struct {
	DNSServer string
	VPNRoutes struct {
		Domains []string
		IPs     []string
	}
}

// Config holds config fields for vpnroutesd.
type Config struct {
	DNSServer  net.IP
	VPNDomains []string
	VPNIPs     []net.IP
}

var lastConfigData []byte

// Load reads and parses a vpnroutesd config file from p, which can be one of
// the following:
//   1. filesystem path, e.g.
//        /home/user/.vpnroutesd.toml
//   2. https URL, e.g.
//        https://internal.4seasontotallandscaping.com/.vpnroutesd.toml
//   3. keybase filesystem path, e.g.
//        keybase@alice://team/4seasontotallandscaping/vpn/.vpnroutesd.toml
//        (alice is the system username, not keybase username)
func Load(logger *zap.Logger, p string) (cfg Config, changed bool, err error) {
	data, err := readConfig(logger, p)
	if err != nil {
		return Config{}, false, fmt.Errorf("reading config file error: %v", err)
	}
	changed = !bytes.Equal(lastConfigData, data)
	lastConfigData = data

	var cfgToml configToml
	if err = toml.Unmarshal(data, &cfgToml); err != nil {
		return Config{}, false, fmt.Errorf("parsing config file error: %v", err)
	}

	if len(cfgToml.DNSServer) == 0 {
		logger.Sugar().Debugf("DNSServer missing; using 8.8.8.8")
		cfg.DNSServer = net.ParseIP("8.8.8.8")
	} else {
		cfg.DNSServer = net.ParseIP(cfgToml.DNSServer)
		if cfg.DNSServer == nil {
			return Config{}, false, fmt.Errorf("%s is not a valid IP address", cfgToml.DNSServer)
		}
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

	return cfg, changed, err
}
