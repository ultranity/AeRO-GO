package main

import (
	"AeRO/proxy/netstat"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

var (
	ocp       = flag.String("ocp", "", "Occupy ports in range")
	rand      = flag.String("range", "", "display avail ports in range")
	listPort  = flag.Bool("port", false, "display using ports")
	udp       = flag.Bool("udp", false, "display UDP sockets")
	tcp       = flag.Bool("tcp", false, "display TCP sockets")
	listening = flag.Bool("lis", false, "display only listening sockets")
	all       = flag.Bool("all", false, "display both listening and non-listening sockets")
	resolve   = flag.Bool("res", false, "lookup symbolic names for host addresses")
	ipv4      = flag.Bool("4", false, "display only IPv4 sockets")
	ipv6      = flag.Bool("6", false, "display only IPv6 sockets")
	help      = flag.Bool("help", false, "display this help screen")
)

const (
	protoIPv4 = 0x01
	protoIPv6 = 0x02
)

func main() {
	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}
	var occupy_range bool = *ocp != ""
	if occupy_range {
		ports := strings.Split(*ocp, ":")
		sPort, _ := strconv.Atoi(ports[0])
		ePort, _ := strconv.Atoi(ports[1])
		for i := sPort; i < ePort; i++ {
			go net.Listen("tcp", ":"+strconv.Itoa(i))
		}
		listener, _ := net.Listen("tcp", ":"+strconv.Itoa(ePort))

		for {
			conn, err := listener.Accept()
			if err != nil {
				fmt.Printf("%v listener closed", conn)
				return
			}
		}
	}
	var in_range bool = *rand != ""
	var sPort, ePort int
	if in_range {
		ports := strings.Split(*rand, ":")
		sPort, _ = strconv.Atoi(ports[0])
		ePort, _ = strconv.Atoi(ports[1])
		aPort, err := netstat.FindOneAvailablePort(sPort, ePort)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("Available port: %d\n", aPort)
		}
		return
	}

	var proto uint
	if *ipv4 {
		proto |= protoIPv4
	}
	if *ipv6 {
		proto |= protoIPv6
	}
	if proto == 0x00 {
		proto = protoIPv4 | protoIPv6
	}

	if os.Geteuid() != 0 {
		fmt.Println("Not all processes could be identified, you would have to be root to see it all.")
	}
	fmt.Printf("Proto %-23s %-23s %-12s %-16s\n", "Local Addr", "Foreign Addr", "State", "PID/Program name")

	if *udp {
		if proto&protoIPv4 == protoIPv4 {
			socks, err := netstat.UDPSocks(netstat.NoopFilter)
			if err == nil {
				displaySockInfo("udp", socks, *listPort)
			}
		}
		if proto&protoIPv6 == protoIPv6 {
			socks, err := netstat.UDP6Socks(netstat.NoopFilter)
			if err == nil {
				displaySockInfo("udp6", socks, *listPort)
			}
		}
	} else {
		*tcp = true
	}

	if *tcp {
		var fn netstat.AcceptFn

		switch {
		case *all:
			fn = func(*netstat.SockTabEntry) bool { return true }
		case *listening:
			fn = func(s *netstat.SockTabEntry) bool {
				return s.State == netstat.Listen
			}
		default:
			fn = func(s *netstat.SockTabEntry) bool {
				return s.State != netstat.Listen
			}
		}

		if proto&protoIPv4 == protoIPv4 {
			socks, err := netstat.TCPSocks(fn)
			if err == nil {
				displaySockInfo("tcp", socks, *listPort)
			}
		}
		if proto&protoIPv6 == protoIPv6 {
			socks, err := netstat.TCP6Socks(fn)
			if err == nil {
				displaySockInfo("tcp6", socks, *listPort)
			}
		}
	}
}

func lookup(skaddr *netstat.SockAddr) string {
	const IPv4Strlen = 17
	addr := skaddr.IP.String()
	if *resolve {
		names, err := net.LookupAddr(addr)
		if err == nil && len(names) > 0 {
			addr = names[0]
		}
	}
	if len(addr) > IPv4Strlen {
		addr = addr[:IPv4Strlen]
	}
	return fmt.Sprintf("%s:%d", addr, skaddr.Port)
}

func displaySockInfo(proto string, socks []netstat.SockTabEntry, listPort bool) {
	for _, e := range socks {
		p := ""
		if e.Process != nil {
			p = e.Process.String()
		}
		lAddr := lookup(e.LocalAddr)
		rAddr := lookup(e.RemoteAddr)
		fmt.Printf("%-5s %-23.23s %-23.23s %-12s %-16s\n", proto, lAddr, rAddr, e.State, p)
	}
	if listPort {
		p := netstat.ToPorts(socks)
		fmt.Printf("%v\n", p)
	}
}
