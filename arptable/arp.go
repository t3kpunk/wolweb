//go:build linux
// +build linux

package arptable

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/vishvananda/netlink"
)

const (
	f_IPAddr int = iota
	f_HWType
	f_Flags
	f_HWAddr
	f_Mask
	f_Device
)

type ArpTable map[string]string

type cache struct {
	sync.RWMutex
	table ArpTable

	Updated      time.Time
	UpdatedCount int
}

func (c *cache) Refresh() {
	c.Lock()
	defer c.Unlock()

	c.table = Table()
	c.Updated = time.Now()
	c.UpdatedCount += 1
}

func (c *cache) Search(ip string) string {
	c.RLock()
	defer c.RUnlock()

	mac, ok := c.table[ip]

	if !ok {
		c.RUnlock()
		c.Refresh()
		c.RLock()
		mac = c.table[ip]
	}

	return mac
}

var (
	stop     = make(chan struct{})
	arpCache = &cache{table: make(ArpTable)}
)

func init() {
	arpCache.Refresh()
	AutoRefresh(10 * time.Second)
}

func AutoRefresh(t time.Duration) {
	go func() {
		for {
			select {
			case <-time.After(t):
				arpCache.Refresh()
			case <-stop:
				return
			}
		}
	}()
}

func StopAutoRefresh() {
	stop <- struct{}{}
}

func CacheUpdate() {
	arpCache.Refresh()
}

func CacheLastUpdate() time.Time {
	return arpCache.Updated
}

func CacheUpdateCount() int {
	return arpCache.UpdatedCount
}

// Search looks up the MAC address for an IP address
// in the arpx table
func Search(ip string) string {
	return arpCache.Search(ip)
}

func SearchHardware(ip string) (net.HardwareAddr, error) {
	result := Search(ip)
	if result != "" {
		return net.ParseMAC(result)
	}
	return nil, fmt.Errorf("arpx search table failed: %s", ip)
}

func Table() ArpTable {
	f, err := os.Open("/proc/net/arp")

	if err != nil {
		return nil
	}

	defer f.Close()

	s := bufio.NewScanner(f)
	s.Scan() // skip the field descriptions

	var table = make(ArpTable)

	for s.Scan() {
		line := s.Text()
		fields := strings.Fields(line)

		// Flags: 0x0 (incomplete), 0x2 (complete/valid), 0x4 (static).
		if fields[f_Flags] != "0x0" {
			table[fields[f_IPAddr]] = fields[f_HWAddr]
		}
	}

	return table
}

// FlushARP deletes every neighbour entry the kernel knows about.
// It works for both IPv4 ARP and IPv6 NDP tables.
func FlushARP() {
	// Get all interfaces – we’ll iterate over them
	ifaces, err := netlink.LinkList()
	if err != nil {
		log.Printf("list links: %w", err)
	}

	for _, iface := range ifaces {
		neighs, err := netlink.NeighList(iface.Attrs().Index,
			netlink.FAMILY_ALL) // FAMILY_V4 | FAMILY_V6
		if err != nil {
			log.Printf("list neigh on %q: %w", iface.Attrs().Name, err)
		}

		for _, n := range neighs {
			err = netlink.NeighDel(&n)
			if err != nil {
				log.Printf("failed to delete neighbour %#v: %v", n.IP.String(), err)
			}
		}
	}
}
