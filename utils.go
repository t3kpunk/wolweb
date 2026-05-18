package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/j-keck/arping"
)

// returns the number of consecutive 1‑bits in n,
// beginning at its LSB.
func CIDR(n uint32) int {
	count := 0
	for n&1 == 1 { // while LSB equals 1 …
		count++
		n >>= 1
	}
	return count
}

// get CIDR from Broadcast IP
func CIDRFromBroadcastIP(broadcastIP string) (string, error) {
	ip := net.ParseIP(broadcastIP)
	if ip == nil {
		return "", fmt.Errorf("invalid IP address: %s", broadcastIP)
	}
	ipInt := binary.BigEndian.Uint32(ip.To4())
	// mask := uint32(1<<32-1) << (32 - CIDR(ipInt))
	return strconv.Itoa(CIDR(ipInt)), nil
}

// true if the IP address in the network /cidr.
func ipInSubnet(ipStr, cidr string) (bool, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false, fmt.Errorf("invalid IP address: %s", ipStr)
	}
	ipStr = ipStr + "/" + cidr
	_, subnet, err := net.ParseCIDR(ipStr)
	if err != nil { // malformed CIDR
		return false, fmt.Errorf("invalid CIDR: %s", cidr)
	}
	return subnet.Contains(ip), nil
}

// func lookupHost(domain string) ([]string, error) {
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()
// 	ips, err := net.DefaultResolver.LookupHost(ctx, domain)
// 	if err != nil {
// 		return nil, fmt.Errorf("DNS query for %s failed: %v", domain, err)
// 	}
// 	return ips, nil
// }

// arpingHost - Function to arping a host by its name and return the status
func arpingHost(host string) (string, error) {

	var status string
	var errMsg error

	ips, err := net.LookupIP(host)
	if err != nil {
		status = fmt.Sprintf("Error: device '%s' could not be resolved", host)
		errMsg = err
	}

	bCastIP := strings.Split(appConfig.BCastIP, ":")[0]
	cidr, _ := CIDRFromBroadcastIP(bCastIP)

	for idx := range ips {
		// fmt.Printf("Checking if IP '%s' is in subnet '%s'\n", ips[idx], bCastIP)
		validIP, err := ipInSubnet(ips[idx].String(), cidr)
		if validIP == false {
			status = fmt.Sprintf("Error: device '%s' with IP '%s' is not in the same subnet as the Broadcast IP '%s'", host, ips[idx], appConfig.BCastIP)
			errMsg = err
			continue
		}

		_, time, err := arping.Ping(ips[idx])
		switch {
		case err == nil:
			return fmt.Sprintf("Device '%s' with IP '%s' is awake. Packet arp ping time '%s'", host, ips[idx], time), nil
		case err == arping.ErrTimeout:
			status = fmt.Sprintf("Device '%s' with IP '%s' is offline", host, ips[idx])
		case err.Error() == "interrupted system call":
			status = fmt.Sprintf("Device '%s' with IP '%s' is offline", host, ips[idx])
		default:
			status = fmt.Sprintf("Error: '%s' while sending arping to device '%s' with IP '%s'", err.Error(), host, ips[idx])
			errMsg = fmt.Errorf("%s", err.Error())
		}
	}
	return status, errMsg
}
