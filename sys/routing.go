package sys

import (
	"net"

	"go.uber.org/zap"
)

// InterfaceNames contains two interface names.
type InterfaceNames struct {
	Primary string
	VPN     string
}

// ApplyRoutesArgs includes args needed to call ApplyRoutes. These arges
// specifies the desired final state that ApplyRoutes should achieve.
type ApplyRoutesArgs struct {
	// Interface specifies the primary and VPN interfaces. Set to nil to auto
	// detect.
	Interfaces *InterfaceNames
	// VPNIPs is a list of IPs that should go through the VPN interface.
	VPNIPs []net.IP
}

// ApplyRoutes takes a declarative speficiation of what the routes should be
// like, and interact with the system routing table to achieve that state.
func ApplyRoutes(logger *zap.Logger, args ApplyRoutesArgs) (changed bool, err error) {
	logger.Sugar().Debugf("+ ApplyRoutes")
	defer logger.Sugar().Debugf("- ApplyRoutes")
	return applyRoutes(logger, args)
}
