package dns

import (
	"net"

	"go.uber.org/zap"
)

var lastIPs []net.IP

func sameIPs(a, b []net.IP) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[ipv4Addr]bool)
	for _, ip := range a {
		set[ipToArray(ip)] = true
	}
	for _, ip := range b {
		if !set[ipToArray(ip)] {
			return false
		}
	}
	return true
}

// GetIPs returns IP address for domains. The IP address include both currently
// resolved addresses from the DNS, and any addresses from previously seen
// records that haven't expired.
func GetIPs(logger *zap.Logger, dnsServer net.IP, domains []string) (ips []net.IP, changed bool, err error) {
	logger.Debug("+ GetIPs")
	defer logger.Debug("- GetIPs")
	for _, domain := range domains {
		domainIPs := theResolver.get(logger, dnsServer, domain)
		logger.Sugar().Debugf("resolved IPs for %s: %s", domain, domainIPs)
		ips = append(ips, domainIPs...)
	}
	changed = !sameIPs(lastIPs, ips)
	lastIPs = ips
	return ips, changed, nil
}
