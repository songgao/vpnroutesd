# vpnroutesd

`vpnroutesd` makes your corporate VPN less miserable by having your Internet
traffic go through the primary interface (e.g. WiFi, Ethernet) by default, and
allows you to explicitly specify IPs and domain names to which the traffic
should go through the VPN.

This is less trivial than it seems. Not all services/websites use fixed IPs, so
the routes need to follow DNS record changes. But the routing layer is below
the DNS, and the routing table doesn't understand DNS. As a result, having a
pair of `/etc/ppp/ip.{up,down}` won't work. We need something in application
layer that constantly resolves DNS names into IP addresses, and feeds them into
the routing table. That's what `vpnroutesd` does.

`vpnroutesd` is currently supported on macOS only. I'll try to find time to add
Linux support. Maybe Windows too if I ever figure out how Windows routing
works. Contributions are of course welcome but there's no test or CI right now
so it might be a bit tough.

## Installation

```
go get -u github.com/songgao/vpnroutesd
```

## Usage

Create a config file `config.toml`:

```toml
# Optional. This is the DNS server that vpnroutesd uses to look up domain
# names. If omitted, "8.8.8.8" is used.
DNSServer = "1.1.1.1"

[vpnroutes]

IPs = [
  # Be sure to include your DNS servers. Often with VPN connected, DNS lookups
  # want to go through the VPN interface. So if you have it routed through
  # the primary lookups will fail.
  "8.8.8.8",
  "8.8.4.4",

  # Add other IPs for accessing corporate internal resources.
  "17.253.144.10",
]

Domains = [
  # domains that don't use a fixed IP (e.g. behind an AWS ELB)
  "internal.4seasontotallandscaping.com",
  "kibana.4seasontotallandscaping.com",
]

```

Store this file somewhere. There are three ways `vpnroutesd` can read a config
file: the good old filesystem, a `https://` URL, or a Keybase Filesystem path:

```bash
sudo ./vpnroutesd -c ~/.vpnroutesd.toml
```
```bash
sudo ./vpnroutesd -c "https://internal.4seasontotallandscaping.com/.vpnroutesd.toml"
```
```bash
# Note the "@$USER" part -- this is because only current user can access KBFS
# paths. Since vpnroutesd is run as root, it needs to know which user to run
# keybase commands as.
sudo ./vpnroutesd -c "keybase@$USER://team/4seasontotallandscaping/vpn/.vpnroutesd.toml"
```

You may have noticed we didn't tell `vpnroutesd` which network interface was
the primary and which was the VPN interface. This is because it has built-in
auto detection for network interfaces. If it fails, you'll know from the logs
and can manually specify inteface names. For example:

```bash
sudo ./vpnroutesd --primary-interface en0 --vpn-interface utun6 -c ~/.vpnroutesd.toml
```

`vpnroutesd` is designed to be a long term running process. It executes tasks
on an interval (default to 1min). On each iteration it reloads the config
file, looks up DNS names, and apply routing changes if needed. As a result, any
configuration changes will be dynamically picked up.

## TODOs

* tests
* Linux
* Windows
