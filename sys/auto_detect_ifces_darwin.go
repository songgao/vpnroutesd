package sys

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"regexp"

	"go.uber.org/zap"
)

/* Example output:

Network information

IPv4 network interface information
   utun6 : flags      : 0x5 (IPv4,DNS)
           address    : 10.100.0.2
           VPN server : 127.0.0.1
           reach      : 0x00000003 (Reachable,Transient Connection)
     en0 : flags      : 0x5 (IPv4,DNS)
           address    : 10.0.1.7
           reach      : 0x00000002 (Reachable)

   REACH : flags 0x00000003 (Reachable,Transient Connection)

IPv6 network interface information
   No IPv6 states found


   REACH : flags 0x00000000 (Not Reachable)

Network interfaces: utun6 en0

*/

var reScutilStart = regexp.MustCompile("IPv4 network interface information")
var reScutilEnd = regexp.MustCompile("IPv6 network interface information")
var reScutilInterfaceStart = regexp.MustCompile(`([a-z0-9]+) : flags\s+: 0x(\S+)`)
var reScutilInterfaceVPNServer = regexp.MustCompile(`\s+VPN server\s+: (\S+)`)
var reUTUN = regexp.MustCompile(`^utun\d+$`)

// TODO: we can be smarter here and check the Reach flags if we want
// var reScutilInterfaceAddress = regexp.MustCompile(`\s+address\s+: (\S+)`)
// var reScutilInterfaceReach = regexp.MustCompile(`\s+reach\s+: 0x(\S+)`)

type ifceForAutoDetect struct {
	ifceName string
	isVPN    bool
}

func findIfces(output []byte) (ifces []ifceForAutoDetect, err error) {
	buf := bytes.NewBuffer(output)

	// Find and consume the start line
	for {
		line, err := buf.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		if reScutilStart.Match(line) {
			break
		}
	}

	currentIfceName := ""
	currentIsVPN := false
	for {
		line, err := buf.ReadBytes('\n')
		switch err {
		case nil:
		case io.EOF:
			break
		default:
			return nil, err
		}

		if matches := reScutilInterfaceStart.FindSubmatch(line); len(matches) > 0 {
			if len(currentIfceName) > 0 {
				ifces = append(ifces, ifceForAutoDetect{currentIfceName, currentIsVPN})
				currentIfceName = ""
				currentIsVPN = false
			}
			currentIfceName = string(matches[1])
		} else if reScutilInterfaceVPNServer.Match(line) {
			currentIsVPN = true
		} else if reScutilEnd.Match(line) {
			break
		}
	}
	if len(currentIfceName) > 0 {
		ifces = append(ifces, ifceForAutoDetect{currentIfceName, currentIsVPN})
		currentIfceName = ""
		currentIsVPN = false
	}
	return ifces, nil
}

func autoDetectIfces(logger *zap.Logger, args *ApplyRoutesArgs) error {
	output, err := exec.Command("/usr/sbin/scutil", "--nwi").Output()
	if err != nil {
		return err
	}
	ifces, err := findIfces(output)
	if err != nil {
		return fmt.Errorf("failed to auto detect: %v", err)
	}
	if len(ifces) != 2 {
		return fmt.Errorf("failed to auto detect: expected two interfaces but found: %#+v", ifces)
	}
	if ifces[0].isVPN && ifces[1].isVPN {
		return fmt.Errorf("failed to auto detect: both interfaces have isVPN=true")
	}

	if !ifces[0].isVPN && !ifces[1].isVPN {
		logger.Sugar().Debugf("both interfaces have isVPN=false. Will try using just the interface names")
		isUTUN0 := reUTUN.MatchString(ifces[0].ifceName)
		isUTUN1 := reUTUN.MatchString(ifces[1].ifceName)
		if isUTUN0 == isUTUN1 {
			return fmt.Errorf("failed to auto detect: both interfaces have isVPN=false and both have isUTUN=%v", isUTUN0)
		}

		if isUTUN0 {
			args.Interfaces = &InterfaceNames{
				VPN:     ifces[0].ifceName,
				Primary: ifces[1].ifceName,
			}
		} else {
			args.Interfaces = &InterfaceNames{
				Primary: ifces[0].ifceName,
				VPN:     ifces[1].ifceName,
			}
		}

		return nil
	}

	if ifces[0].isVPN {
		args.Interfaces = &InterfaceNames{
			VPN:     ifces[0].ifceName,
			Primary: ifces[1].ifceName,
		}
	} else {
		args.Interfaces = &InterfaceNames{
			Primary: ifces[0].ifceName,
			VPN:     ifces[1].ifceName,
		}
	}
	return nil
}
