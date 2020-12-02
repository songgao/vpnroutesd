package sys

import (
	"errors"
	"os/exec"
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

func autoDetectIfces(args *ApplyRoutesArgs) error {
	_, err := exec.Command("/usr/sbin/scutil", "--nwi").Output()
	if err != nil {
		return err
	}
	return errors.New("wip")
}
