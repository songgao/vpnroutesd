package sys

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"

	"go.uber.org/zap"
	"golang.org/x/net/route"
)

const (
	routeMessageVersion = 5
)

type ipv4Addr [4]byte

var ipv4Zeros = ipv4Addr{0, 0, 0, 0}

var rtAddrNames = []string{
	syscall.RTAX_AUTHOR:  "author",
	syscall.RTAX_BRD:     "brd",
	syscall.RTAX_DST:     "dst",
	syscall.RTAX_GATEWAY: "gateway",
	syscall.RTAX_GENMASK: "genmask",
	syscall.RTAX_IFA:     "ifa",
	syscall.RTAX_IFP:     "ifp",
	syscall.RTAX_MAX:     "max",
	syscall.RTAX_NETMASK: "netmask",
}

var rtFlagNames = []struct {
	mask int
	name string
}{
	{syscall.RTF_BLACKHOLE, "BLACKHOLE"},
	{syscall.RTF_BROADCAST, "BROADCAST"},
	{syscall.RTF_CLONING, "CLONING"},
	{syscall.RTF_CONDEMNED, "CONDEMNED"},
	{syscall.RTF_DELCLONE, "DELCLONE"},
	{syscall.RTF_DONE, "DONE"},
	{syscall.RTF_DYNAMIC, "DYNAMIC"},
	{syscall.RTF_GATEWAY, "GATEWAY"},
	{syscall.RTF_HOST, "HOST"},
	{syscall.RTF_IFREF, "IFREF"},
	{syscall.RTF_IFSCOPE, "IFSCOPE"},
	{syscall.RTF_LLINFO, "LLINFO"},
	{syscall.RTF_LOCAL, "LOCAL"},
	{syscall.RTF_MODIFIED, "MODIFIED"},
	{syscall.RTF_MULTICAST, "MULTICAST"},
	{syscall.RTF_PINNED, "PINNED"},
	{syscall.RTF_PRCLONING, "PRCLONING"},
	{syscall.RTF_PROTO1, "PROTO1"},
	{syscall.RTF_PROTO2, "PROTO2"},
	{syscall.RTF_PROTO3, "PROTO3"},
	{syscall.RTF_REJECT, "REJECT"},
	{syscall.RTF_STATIC, "STATIC"},
	{syscall.RTF_UP, "UP"},
	{syscall.RTF_WASCLONED, "WASCLONED"},
	{syscall.RTF_XRESOLVE, "XRESOLVE"},
}

func rtFlags(f int) (ret []string) {
	for _, fn := range rtFlagNames {
		if fn.mask&f != 0 {
			ret = append(ret, fn.name)
		}
	}
	return ret
}

type ifceInfo struct {
	name   string
	index  int
	selfIP ipv4Addr
}

func (ii ifceInfo) String() string {
	return fmt.Sprintf("[%s] index=%d ip=%s", ii.name, ii.index, net.IP(ii.selfIP[:]))
}

func getIfceInfo(logger *zap.Logger, name string) (info ifceInfo, err error) {
	b, err := route.FetchRIB(syscall.AF_INET, route.RIBTypeInterface, 0)
	if err != nil {
		return ifceInfo{}, err
	}
	msgs, err := route.ParseRIB(route.RIBTypeInterface, b)
	if err != nil {
		return ifceInfo{}, err
	}

loopLink:
	for _, msg := range msgs {
		switch m := msg.(type) {
		case (*route.InterfaceMessage):
			for _, addr := range m.Addrs {
				linkAddr, ok := addr.(*route.LinkAddr)
				if !ok {
					// logger.Sugar().Debugf("ignoring message that is not LinkAddr: %#+v\n", addr)
					continue
				}
				if linkAddr.Name == name {
					info.index = linkAddr.Index
					break loopLink
				}
			}
		default:
			// logger.Sugar().Debugf("ignoring message that is not InterfaceMessage")
			continue
		}
	}

	if info.index == 0 {
		return ifceInfo{}, errors.New("interface not found")
	}

loopAddr:
	for _, msg := range msgs {
		switch m := msg.(type) {
		case (*route.InterfaceAddrMessage):
			if m.Index != info.index {
				continue loopAddr
			}
			ipAddr, ok := m.Addrs[syscall.RTAX_IFA].(*route.Inet4Addr)
			if !ok {
				// logger.Sugar().Debugf("ignoring message that is not Inet4Addr: %#+v\n", addr)
				continue
			}
			info.selfIP = ipAddr.IP
			break loopAddr
		default:
			// logger.Sugar().Debugf("ignoring message that is not InterfaceMessage")
			continue
		}
	}

	info.name = name

	return info, nil
}

func fetchRoutes(logger *zap.Logger, ifceIndex int) (routes []*route.RouteMessage, err error) {
	b, err := route.FetchRIB(syscall.AF_INET, route.RIBTypeRoute, 0)
	if err != nil {
		return nil, err
	}
	msgs, err := route.ParseRIB(route.RIBTypeRoute, b)
	if err != nil {
		return nil, err
	}
	for _, msg := range msgs {
		rm, ok := msg.(*route.RouteMessage)
		if !ok {
			// logger.Sugar().Debugf("ignoring message that is not RouteMessage")
			continue
		}
		if rm.Index != ifceIndex {
			// logger.Sugar().Debugf("other interface:\n%s", pretty.Sprint(rm))
			continue
		}
		routes = append(routes, rm)
	}
	return routes, nil
}

func printRoutesForDebug(logger *zap.Logger, ifceIndex int, ignoreErrs bool) {
	routeMsgs, err := fetchRoutes(logger, ifceIndex)
	if err != nil {
		logger.Sugar().Fatal(err)
	}
	for _, rm := range routeMsgs {
		if ignoreErrs && rm.Err != nil {
			continue
		}
		// fmt.Println(pretty.Sprint(rm))
		fmt.Println("-- route --")
		if !ignoreErrs {
			fmt.Printf("Err: %v\n", rm.Err)
		}
		fmt.Printf("Version: %d\n", rm.Version)
		fmt.Printf("Type: %d\n", rm.Type)
		fmt.Printf("Flags: %s\n", strings.Join(rtFlags(rm.Flags), ","))
		fmt.Printf("Addrs:\n")
		for i, addr := range rm.Addrs {
			if addr == nil {
				continue
			}
			fmt.Printf("%s: ", rtAddrNames[i])
			switch a := addr.(type) {
			case *route.Inet4Addr:
				fmt.Println(net.IP(a.IP[:]).String())
			case *route.LinkAddr:
				fmt.Printf("index: %d; addr: %s\n", a.Index, net.HardwareAddr(a.Addr).String())
			default:
				fmt.Printf("%#+v\n", addr)
			}
		}
	}
}

type routeItem struct {
	dst         ipv4Addr
	gatewayLink *int
	gatewayIP   *ipv4Addr // only for LOCAL routes
	netmask     *ipv4Addr
	ifa         *ipv4Addr
}

func (ri *routeItem) String() (ret string) {
	if ri.netmask == nil {
		ret += net.IP(ri.dst[:]).String()
	} else {
		ret += (&net.IPNet{
			IP:   net.IP(ri.dst[:]),
			Mask: net.IPMask(ri.netmask[:]),
		}).String()
	}
	ret += " via"
	if ri.gatewayLink != nil {
		ret += fmt.Sprintf(" link#%d", *ri.gatewayLink)
	}
	if ri.gatewayIP != nil {
		ret += fmt.Sprintf(" %s", net.IP((*ri.gatewayIP)[:]).String())
	}
	if ri.gatewayIP == nil && ri.gatewayLink == nil {
		ret += " [empty]"
	}
	if ri.ifa != nil {
		ret += fmt.Sprintf(" (%s)", net.IP((*ri.ifa)[:]).String())
	}
	return ret
}

func matchIP(ip *ipv4Addr, addr route.Addr) bool {
	a, ok := addr.(*route.Inet4Addr)
	if ip == nil && (!ok || a == nil) {
		return true
	}
	return a.IP == *ip
}

func matchLink(linkIndex *int, addr route.Addr) bool {
	a, ok := addr.(*route.LinkAddr)
	if linkIndex == nil && (!ok || a == nil) {
		return true
	}
	return linkIndex != nil && a != nil && a.Index == *linkIndex
}

func (ri *routeItem) matches(logger *zap.Logger, routeMessage *route.RouteMessage) bool {
	// NOTE: It seems AF_ROUTE reports error even when it works. So ignore this
	// check for now.
	// if routeMessage.Err != nil {
	// 	logger.Sugar().Debugf("routeMessage not matched: error: %v", routeMessage.Err)
	// 	return false
	// }
	if !matchIP(&ri.dst, routeMessage.Addrs[syscall.RTAX_DST]) {
		logger.Sugar().Debugf("routeMessage not matched: dst")
		return false
	}
	if !matchLink(ri.gatewayLink, routeMessage.Addrs[syscall.RTAX_GATEWAY]) {
		logger.Sugar().Debugf("routeMessage not matched: gateway")
		return false
	}
	if !matchIP(ri.gatewayIP, routeMessage.Addrs[syscall.RTAX_GATEWAY]) {
		logger.Sugar().Debugf("routeMessage not matched: gateway")
		return false
	}
	if !matchIP(ri.netmask, routeMessage.Addrs[syscall.RTAX_NETMASK]) {
		logger.Sugar().Debugf("routeMessage not matched: netmask")
		return false
	}
	if !matchIP(ri.ifa, routeMessage.Addrs[syscall.RTAX_IFA]) {
		logger.Sugar().Debugf("routeMessage not matched: ifa")
		return false
	}
	return true
}

func toRouteAddr(ip *ipv4Addr, linkIndex *int) route.Addr {
	if ip != nil {
		return &route.Inet4Addr{
			IP: *ip,
		}
	}
	if linkIndex != nil {
		return &route.LinkAddr{
			Index: *linkIndex,
		}
	}
	return nil
}

func (ri *routeItem) toRouteMessage(seq int, ifceIndex int, msgType int) *route.RouteMessage {
	var flags int = syscall.RTF_UP
	if ri.gatewayIP != nil && ri.gatewayLink == nil {
		flags |= syscall.RTF_LOCAL
	}
	if ri.netmask == nil {
		flags |= syscall.RTF_HOST
	}
	return &route.RouteMessage{
		Version: routeMessageVersion,
		Type:    msgType,
		Flags:   flags,
		Index:   ifceIndex,
		ID:      uintptr(os.Getpid()),
		Seq:     seq,
		Addrs: []route.Addr{
			syscall.RTAX_DST:     toRouteAddr(&ri.dst, nil),
			syscall.RTAX_GATEWAY: toRouteAddr(ri.gatewayIP, ri.gatewayLink),
			syscall.RTAX_NETMASK: toRouteAddr(ri.netmask, nil),
			syscall.RTAX_IFA:     toRouteAddr(ri.ifa, nil),
		},
	}
}

type routesDescription struct {
	iiPrimary ifceInfo
	iiVPN     ifceInfo
	vpnIPs    []ipv4Addr
}

func (rd *routesDescription) apply(logger *zap.Logger) (changed bool, err error) {
	expectedItems := map[ipv4Addr]*routeItem{
		rd.iiVPN.selfIP: {
			dst:       rd.iiVPN.selfIP,
			gatewayIP: &rd.iiVPN.selfIP,
			ifa:       &rd.iiVPN.selfIP,
		},
		ipv4Zeros: {
			dst:         ipv4Zeros,
			netmask:     &ipv4Addr{0, 0, 0, 0},
			gatewayLink: &rd.iiPrimary.index,
			ifa:         &rd.iiPrimary.selfIP,
		},
	}
	for _, ip := range rd.vpnIPs {
		expectedItems[ip] = &routeItem{
			dst:         ip,
			gatewayLink: &rd.iiVPN.index,
			ifa:         &rd.iiVPN.selfIP,
		}
	}
	found := make(map[ipv4Addr]bool)

	// See if we can find the default route, and if so, mark it as found.
	routeMsgsPrimary, err := fetchRoutes(logger, rd.iiPrimary.index)
	if err != nil {
		return false, err
	}
	for _, rm := range routeMsgsPrimary {
		addr, ok := rm.Addrs[syscall.RTAX_DST].(*route.Inet4Addr)
		if !ok || addr.IP != ipv4Zeros {
			continue
		}
		expected := expectedItems[ipv4Zeros]
		if !expected.matches(logger, rm) {
			continue
		}
		logger.Sugar().Debugf("skipping for existing routeItem: %s", expected)
		found[ipv4Zeros] = true
		break
	}

	// Go through all routes on the VPN interface and make changes as needed.
	routeMsgsVPN, err := fetchRoutes(logger, rd.iiVPN.index)
	if err != nil {
		return false, err
	}
	nextSeq := 1
	var toWrite []*route.RouteMessage
	for _, rm := range routeMsgsVPN {
		if rm.Flags&syscall.RTF_WASCLONED != 0 {
			// ignore cloned routes
			continue
		}
		dstAddr, ok := rm.Addrs[syscall.RTAX_DST].(*route.Inet4Addr)
		if !ok || dstAddr == nil {
			// ???
			continue
		}

		expected := expectedItems[dstAddr.IP]
		if expected == nil || !expected.matches(logger, rm) {
			// Construct a DELETE message and append it to toWrite.
			td := *rm
			td.Seq = nextSeq
			nextSeq++
			td.Type = syscall.RTM_DELETE

			logger.Sugar().Infof("queueing DELETE (seq %d) because routeMessage doesn't match routeItem: %s", td.Seq, expected)
			toWrite = append(toWrite, &td)
		} else {
			// Mark it as found so we don't re-add it.
			found[dstAddr.IP] = true
		}
	}

	for dst, item := range expectedItems {
		if found[dst] {
			logger.Sugar().Debugf("skipping for existing routeItem: %s", item)
			continue
		}
		// Append a ADD message to toWrite.
		logger.Sugar().Infof("queueing ADD (seq %d) for routeItem: %s", nextSeq, item)
		toWrite = append(toWrite, item.toRouteMessage(nextSeq, rd.iiVPN.index, syscall.RTM_ADD))
		nextSeq++
	}

	fd, err := syscall.Socket(syscall.AF_ROUTE, syscall.SOCK_RAW, 0)
	if err != nil {
		return false, err
	}
	defer syscall.Close(fd)

	if len(toWrite) == 0 {
		logger.Sugar().Debugf("routes are correct; done!")
		return false, nil
	}

	logger.Sugar().Infof("writing %d routeMessage items to AF_ROUTE", len(toWrite))
	for _, msg := range toWrite {
		// logger.Sugar().Infof("writing message: %s", pretty.Sprint(msg))
		b, err := msg.Marshal()
		if err != nil {
			return false, err
		}
		_, err = syscall.Write(fd, b)
		if err != nil {
			logger.Sugar().Warnf("error writing message seq %d: %v", msg.Seq, err)
			continue
		}
	}
	logger.Sugar().Infof("done writing %d routeMessage items to AF_ROUTE", len(toWrite))

	return true, nil
}

func applyRoutes(logger *zap.Logger, args ApplyRoutesArgs) (changed bool, err error) {
	if args.Interfaces == nil {
		logger.Sugar().Debugf("using auto detect for interface names")
		if err := autoDetectIfces(logger, &args); err != nil {
			return false, err
		}
	}
	if args.Interfaces.Primary == args.Interfaces.VPN {
		return false, errors.New("primary and vpn interface can't be same")
	}
	ifceInfoPrimary, err := getIfceInfo(logger, args.Interfaces.Primary)
	if err != nil {
		return false, err
	}
	logger.Sugar().Debugf("Primary Interface: %s\n", ifceInfoPrimary)

	ifceInfoVPN, err := getIfceInfo(logger, args.Interfaces.VPN)
	if err != nil {
		return false, err
	}
	logger.Sugar().Debugf("VPN Interface: %s\n", ifceInfoVPN)

	vpnIPs := make([]ipv4Addr, 0, len(args.VPNIPs))
	for _, argIP := range args.VPNIPs {
		argIPv4 := argIP.To4()
		if argIP == nil {
			logger.Sugar().Infof("ignored non-IPv6 address: %s\n", argIP)
			continue
		}

		var ip ipv4Addr
		copy(ip[:], argIPv4[:4])

		vpnIPs = append(vpnIPs, ip)
	}

	return (&routesDescription{
		iiPrimary: ifceInfoPrimary,
		iiVPN:     ifceInfoVPN,
		vpnIPs:    vpnIPs,
	}).apply(logger)
}
