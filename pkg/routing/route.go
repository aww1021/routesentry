package routing

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

func AddHostRoute(iface string, ip net.IP) error {
	dst := &net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(32, 32),
	}

	link, err := netlink.LinkByName(iface)
	if err != nil {
		return fmt.Errorf("could not find link %s: %w", iface, err)
	}

	route := &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Dst:       dst,
	}
	err = netlink.RouteAdd(route)
	if err != nil {
		return fmt.Errorf("could not add route to %s: %w", iface, err)
	}
	return nil
}
