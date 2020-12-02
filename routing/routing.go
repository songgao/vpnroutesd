package routing

import "net"

// ApplyRoutesArgs includes args needed to call ApplyRoutes. These arges
// specifies the desired final state that ApplyRoutes should achieve.
type ApplyRoutesArgs struct {
	IfceNamePrimary string
	IfceNameVPN     string
	VPNIPs          []net.IP
}

// ApplyRoutes takes a declarative speficiation of what the routes should be
// like, and interact with the system routing table to achieve that state.
func ApplyRoutes(args ApplyRoutesArgs) error {
	return applyRoutes(args)
}
