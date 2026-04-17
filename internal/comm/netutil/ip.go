// Package netutil provides network utility functions for Shepherd.
package netutil

import (
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/utils"
	"net"
)

// GetBestLocalIP tries to find the best local IP address.
// It first attempts to determine the outbound IP by making a UDP connection
// to a public DNS server (8.8.8.8). If that fails, it falls back to
// enumerating network interfaces to find a non-loopback IPv4 address.
//
// Returns "127.0.0.1" if no suitable IP address is found.
func GetBestLocalIP() string {
	// 首先尝试连接到一个外部地址来获取出口IP
	// 这可以确保我们获取的是可以访问Master的IP
	if conn, err := net.Dial("udp", "8.8.8.8:80"); err == nil {
		defer utils.CloseQuietly(conn)
		if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
			return addr.IP.String()
		}
	}

	// 回退到接口枚举
	interfaces, _ := net.Interfaces()
	for _, iface := range interfaces {
		// 跳过回环和down的接口
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.To4() != nil && !ipnet.IP.IsLoopback() {
					return ipnet.IP.String()
				}
			}
		}
	}

	return "127.0.0.1"
}
