package netstat

import (
	"AeRO/proxy/util"
	"fmt"
	"net"
	"strconv"
)

// SockAddr represents an ip:port pair
type SockAddr struct {
	IP   net.IP
	Port uint16
}

func (s *SockAddr) String() string {
	return fmt.Sprintf("%v:%d", s.IP, s.Port)
}

// SockTabEntry type represents each line of the /proc/net/[tcp|udp]
type SockTabEntry struct {
	ino        string
	LocalAddr  *SockAddr
	RemoteAddr *SockAddr
	State      SkState
	UID        uint32
	Process    *Process
}

func (s *SockTabEntry) Port() uint16 {
	return s.LocalAddr.Port
}

// Process holds the PID and process Name to which each socket belongs
type Process struct {
	Pid  int
	Name string
}

func (p *Process) String() string {
	return fmt.Sprintf("%d/%s", p.Pid, p.Name)
}

// SkState type represents socket connection state
type SkState uint8

func (s SkState) String() string {
	return skStates[s]
}

// AcceptFn is used to filter socket entries. The value returned indicates
// whether the element is to be appended to the socket list.
type AcceptFn func(*SockTabEntry) bool

// NoopFilter - a test function returning true for all elements
func NoopFilter(*SockTabEntry) bool { return true }

// TCPSocks returns a slice of active TCP sockets containing only those
// elements that satisfy the accept function
func TCPSocks(accept AcceptFn) ([]SockTabEntry, error) {
	return osTCPSocks(accept)
}

// TCP6Socks returns a slice of active TCP IPv4 sockets containing only those
// elements that satisfy the accept function
func TCP6Socks(accept AcceptFn) ([]SockTabEntry, error) {
	return osTCP6Socks(accept)
}

// UDPSocks returns a slice of active UDP sockets containing only those
// elements that satisfy the accept function
func UDPSocks(accept AcceptFn) ([]SockTabEntry, error) {
	return osUDPSocks(accept)
}

// UDP6Socks returns a slice of active UDP IPv6 sockets containing only those
// elements that satisfy the accept function
func UDP6Socks(accept AcceptFn) ([]SockTabEntry, error) {
	return osUDP6Socks(accept)
}

func Socks(mode string, accept AcceptFn) ([]SockTabEntry, error) {
	return osSocks(mode, accept)
}

func Ports(mode string, accept AcceptFn) ([]uint16, error) {
	return osPorts(mode, accept)
}

func GetAllSocks(accept AcceptFn) ([]SockTabEntry, error) {
	var sktab []SockTabEntry
	tbl, err := osTCPSocks(accept)
	if err != nil {
		return nil, err
	}
	sktab = append(sktab, tbl...)
	tbl, err = osTCP6Socks(accept)
	if err != nil {
		return sktab, err
	}
	sktab = append(sktab, tbl...)
	tbl, err = osUDPSocks(accept)
	if err != nil {
		return sktab, err
	}
	sktab = append(sktab, tbl...)
	tbl, err = osUDP6Socks(accept)
	if err != nil {
		return sktab, err
	}
	sktab = append(sktab, tbl...)
	return sktab, err
}
func SockPort(e SockTabEntry) uint16 { return e.LocalAddr.Port }

type Set[T comparable] map[T]struct{}

func GetAllPorts(accept AcceptFn) []uint16 {
	ports := make([]uint16, 0, 65535)
	portSet, err := osPorts("tcp", accept)
	if err != nil {
		return nil
	}
	//portSet = util.DeDuplicate(portSet)
	ports = append(ports, portSet...)

	portSet, err = osPorts("tcp6", accept)
	if err != nil {
		return portSet
	}
	//portSet = util.DeDuplicate(portSet)
	ports = append(ports, portSet...)

	//portSet, err = osPorts("udp", accept)
	if err != nil {
		ports = util.DeDuplicate(ports)
		return ports
	}
	//portSet = util.DeDuplicate(portSet)
	ports = append(ports, portSet...)

	portSet, err = osPorts("udp6", accept)
	if err != nil {
		ports = util.DeDuplicate(ports)
		return ports
	}
	//portSet = util.DeDuplicate(portSet)
	ports = append(ports, portSet...)

	ports = util.DeDuplicate(ports)
	return portSet
}

func ToPorts(s []SockTabEntry) []uint16 {
	ports := make([]uint16, len(s))
	for i, e := range s {
		ports[i] = e.LocalAddr.Port
	}
	ports = util.DeDuplicate(ports)
	return ports
}

func FirstAvailPort() {

}

func FindOneAvailablePort(startPort, endPort int) (int, error) {
	for port := startPort; port <= endPort; port++ {
		address := ":" + strconv.Itoa(port)
		listener, err := net.Listen("tcp", address)
		if err == nil {
			listener.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port in the specified range")
}

var sPort, ePort int
var force bool = false

func SetPortRange(startPort, endPort int, forceRange bool) {
	sPort = startPort
	ePort = endPort
	force = forceRange
}

func FindAvailablePort(init int) int {
	if init > 0 {
		if force {
			if init < sPort || init > ePort {
				init = 0
			}
		} else {
			address := ":" + strconv.Itoa(init)
			listener, err := net.Listen("tcp", address)
			if err == nil {
				listener.Close()
				return init
			}
		}
	} else {
		init = sPort
	}
	for port := init; port <= ePort; port++ {
		address := ":" + strconv.Itoa(port)
		listener, err := net.Listen("tcp", address)
		if err == nil {
			listener.Close()
			return port
		}
	}
	return 0
}
