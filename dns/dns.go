package dns

import (
	"net"

	"go.uber.org/zap"
)

// GetIPs returns IP address for domains. The IP address include both currently
// resolved addresses from the DNS, and any addresses from previously seen
// records that haven't expired.
func GetIPs(logger *zap.Logger, domains []string) (ips []net.IP, changed bool, err error) {
	// TODO wip
	return nil, false, nil
}
