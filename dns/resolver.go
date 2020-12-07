package dns

import (
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"go.uber.org/zap"
)

// resolver resolves a domain name to a list of IPs. It remembers previously
// answered records from the DNS server, but unlike a regular caching resolver,
// it's not done to minimize traffic, but an attempt to cover DNS changes. As a
// result, its logic for using remembered results is also different from
// caching resolvers.
//
// For example, consider following scenario: example.com has a CNAME record of
// elb.example.com. elb.example.com has A records of a.b.c.1, a.b.c.2.
// elb.example.com replaces IPs to a.b.d.1, a.b.d.2, either as part of a
// deployment, or because the load balancer needs to rotate IP addresses. Due
// to propagation delay, for a while clients may get either a.b.c.x or a.b.d.x
// records. Server side apparently needs to keep both IP usable, but our
// routing configuration also needs to make sure all four IPs are routed
// through the VPN if the domain is configured so.

type ipv4Addr [4]byte

func ipToArray(ip net.IP) ipv4Addr {
	ip4 := ip.To4()
	if ip4 == nil {
		return ipv4Addr{}
	}
	return ipv4Addr{ip4[0], ip4[1], ip4[2], ip4[3]}
}

type resolverDomain map[ipv4Addr]time.Time

type resolver struct {
	lock        sync.Mutex
	domainToIPs map[string]resolverDomain
}

var theResolver = resolver{
	domainToIPs: make(map[string]resolverDomain),
}

func (r *resolver) lookupLocked(logger *zap.Logger, dnsServer string, domain string) {
	if _, ok := r.domainToIPs[domain]; !ok {
		r.domainToIPs[domain] = make(resolverDomain)
	}
	m := &dns.Msg{}
	m.SetQuestion(domain, dns.TypeA)
	res, err := dns.Exchange(m, dnsServer)
	if err != nil {
		logger.Sugar().Warnf("dns look up for %s failed: %v", domain, err)
		return
	}
	for _, answer := range res.Answer {
		if a, ok := answer.(*dns.A); ok {
			ip := a.A.To4()
			if ip == nil {
				logger.Sugar().Warnf("unexpected non-IPv4 result returned as A record")
				continue
			}
			ipArray := ipToArray(ip)
			expiresAt := time.Now().Add(time.Duration(a.Hdr.Ttl) * time.Second)
			if existingExpireAt, ok := r.domainToIPs[domain][ipArray]; ok && existingExpireAt.After(expiresAt) {
				// don't shorten TTL
				continue
			}
			r.domainToIPs[domain][ipArray] = expiresAt
			logger.Sugar().Debugf("added resolver item: %s -> %s [expires at %s]", domain, ip, expiresAt.Format(time.RFC3339))
		}
	}
}

func (r *resolver) purgeExpiredLocked(logger *zap.Logger) {
	now := time.Now()
	for domain, rd := range r.domainToIPs {
		var purged []net.IP
		for ipArray, expiresAt := range rd {
			if now.After(expiresAt) {
				delete(rd, ipArray)
				ipCopy := make(net.IP, 4)
				copy(ipCopy, ipArray[:])
				purged = append(purged, ipCopy)
			}
		}
		if len(purged) > 0 {
			logger.Sugar().Debugf("purged %d IPs for %s: %s", len(purged), domain, purged)
		}
	}
}

func (r *resolver) get(logger *zap.Logger, dnsServer net.IP, domain string) []net.IP {
	logger.Sugar().Debugf("using %s for DNS lookups", dnsServer.String())
	if !strings.HasSuffix(domain, ".") {
		domain = domain + "."
	}
	r.lock.Lock()
	defer r.lock.Unlock()

	r.lookupLocked(logger, net.JoinHostPort(dnsServer.String(), "53"), domain)
	r.purgeExpiredLocked(logger)

	dr, ok := r.domainToIPs[domain]
	if !ok {
		return nil
	}
	ret := make([]net.IP, 0, len(dr))
	for ipArray := range dr {
		ipCopy := make(net.IP, 4)
		copy(ipCopy, ipArray[:])
		ret = append(ret, ipCopy)
	}
	return ret
}
