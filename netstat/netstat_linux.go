// Package netstat provides primitives for getting socket information on a
// Linux based operating system.
package netstat

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
)

const (
	pathTCPTab  = "/proc/net/tcp"
	pathTCP6Tab = "/proc/net/tcp6"
	pathUDPTab  = "/proc/net/udp"
	pathUDP6Tab = "/proc/net/udp6"

	ipv4StrLen = 8
	ipv6StrLen = 32
)

// Socket states
const (
	Established SkState = 0x01
	SynSent             = 0x02
	SynRecv             = 0x03
	FinWait1            = 0x04
	FinWait2            = 0x05
	TimeWait            = 0x06
	Close               = 0x07
	CloseWait           = 0x08
	LastAck             = 0x09
	Listen              = 0x0a
	Closing             = 0x0b
)

var skStates = [...]string{
	"UNKNOWN",
	"ESTABLISHED",
	"SYN_SENT",
	"SYN_RECV",
	"FIN_WAIT1",
	"FIN_WAIT2",
	"TIME_WAIT",
	"", // CLOSE
	"CLOSE_WAIT",
	"LAST_ACK",
	"LISTEN",
	"CLOSING",
}

// Errors returned by gonetstat
var (
	ErrNotEnoughFields = errors.New("gonetstat: not enough fields in the line")
)

func parseIPv4(s string) (net.IP, error) {
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return nil, err
	}
	ip := make(net.IP, net.IPv4len)
	binary.LittleEndian.PutUint32(ip, uint32(v))
	return ip, nil
}

func parseIPv6(s string) (net.IP, error) {
	ip := make(net.IP, net.IPv6len)
	const grpLen = 4
	i, j := 0, 4
	for len(s) != 0 {
		grp := s[0:8]
		u, err := strconv.ParseUint(grp, 16, 32)
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint32(ip[i:j], uint32(u))
		i, j = i+grpLen, j+grpLen
		s = s[8:]
	}
	return ip, nil
}

func parseAddr(s string) (*SockAddr, error) {
	fields := strings.Split(s, ":")
	if len(fields) < 2 {
		return nil, fmt.Errorf("netstat: not enough fields: %v", s)
	}
	var ip net.IP
	var err error
	switch len(fields[0]) {
	case ipv4StrLen:
		ip, err = parseIPv4(fields[0])
	case ipv6StrLen:
		ip, err = parseIPv6(fields[0])
	default:
		err = fmt.Errorf("netstat: bad formatted string: %v", fields[0])
	}
	if err != nil {
		return nil, err
	}
	v, err := strconv.ParseUint(fields[1], 16, 16)
	if err != nil {
		return nil, err
	}
	return &SockAddr{IP: ip, Port: uint16(v)}, nil
}

func parsePort(s string) (*uint16, error) {
	fields := strings.Split(s, ":")
	if len(fields) < 2 {
		return nil, fmt.Errorf("netstat: not enough fields: %v", s)
	}
	v, err := strconv.ParseUint(fields[1], 16, 16)
	if err != nil {
		return nil, err
	}
	u := uint16(v)
	return &u, nil
}

func ParseSocktab(r io.Reader, accept AcceptFn) ([]SockTabEntry, error) {
	br := bufio.NewScanner(r)
	tab := make([]SockTabEntry, 0, 4)

	// Discard title
	br.Scan()

	for br.Scan() {
		var e SockTabEntry
		line := br.Text()
		// Skip comments
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		fields := strings.Fields(line)
		if len(fields) < 12 {
			return nil, fmt.Errorf("netstat: not enough fields: %v, %v", len(fields), fields)
		}
		addr, err := parseAddr(fields[1])
		if err != nil {
			return nil, err
		}
		e.LocalAddr = addr
		addr, err = parseAddr(fields[2])
		if err != nil {
			return nil, err
		}
		e.RemoteAddr = addr
		u, err := strconv.ParseUint(fields[3], 16, 8)
		if err != nil {
			return nil, err
		}
		e.State = SkState(u)
		u, err = strconv.ParseUint(fields[7], 10, 32)
		if err != nil {
			return nil, err
		}
		e.UID = uint32(u)
		e.ino = fields[9]
		//extractProcInfo(&e)
		if accept(&e) {
			tab = append(tab, e)
		}
	}
	return tab, br.Err()
}

func ParseSockPort(r io.Reader, accept AcceptFn) ([]uint16, error) {
	br := bufio.NewScanner(r)
	ports := make([]uint16, 0, 4)

	// Discard title
	br.Scan()

	for br.Scan() {
		line := br.Text()
		// Skip comments
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		fields := strings.Fields(line)
		if len(fields) < 12 {
			return nil, fmt.Errorf("netstat: not enough fields: %v, %v", len(fields), fields)
		}
		port, err := parsePort(fields[1])
		if err != nil {
			return nil, err
		}
		ports = append(ports, *port)
	}
	return ports, br.Err()
}

type procFd struct {
	base  string
	pid   int
	sktab *SockTabEntry
	p     *Process
}

const sockPrefix = "socket:["

func getProcName(s []byte) string {
	i := bytes.Index(s, []byte("("))
	if i < 0 {
		return ""
	}
	j := bytes.LastIndex(s, []byte(")"))
	if i < 0 {
		return ""
	}
	if i > j {
		return ""
	}
	return string(s[i+1 : j])
}

func (p *procFd) iterFdDir() {
	// link Name is of the form socket:[5860846]
	fddir := path.Join(p.base, "/fd")
	fi, err := ioutil.ReadDir(fddir)
	if err != nil {
		return
	}
	var buf [128]byte

	for _, file := range fi {
		fd := path.Join(fddir, file.Name())
		lname, err := os.Readlink(fd)
		if err != nil || !strings.HasPrefix(lname, sockPrefix) {
			continue
		}

		sk := p.sktab
		ss := sockPrefix + sk.ino + "]"
		if ss != lname {
			continue
		}
		if p.p == nil {
			stat, err := os.Open(path.Join(p.base, "stat"))
			if err != nil {
				return
			}
			if stat != nil {
				defer stat.Close()
			}
			n, err := stat.Read(buf[:])
			if err != nil {
				return
			}
			z := bytes.SplitN(buf[:n], []byte(" "), 3)
			name := getProcName(z[1])
			p.p = &Process{p.pid, name}
			stat.Close()
		}
		sk.Process = p.p
	}
}

func extractProcInfo(sktab *SockTabEntry) {
	const basedir = "/proc"
	fi, err := ioutil.ReadDir(basedir)
	if err != nil {
		return
	}

	for _, file := range fi {
		if !file.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(file.Name())
		if err != nil {
			continue
		}
		base := path.Join(basedir, file.Name())
		proc := procFd{base: base, pid: pid, sktab: sktab}
		proc.iterFdDir()
	}
}

// DoNetstat - collect information about network port status
func DoNetstat(path string, fn AcceptFn) ([]SockTabEntry, error) {
	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	if f != nil {
		defer f.Close()
	}
	tabs, err := ParseSocktab(f, fn)
	if err != nil {
		return nil, err
	}

	return tabs, nil
}

func ListPorts(path string, fn AcceptFn) ([]uint16, error) {
	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	if f != nil {
		defer f.Close()
	}
	ports, err := ParseSockPort(f, fn)
	if err != nil {
		return nil, err
	}

	return ports, nil
}

func osPorts(mode string, accept AcceptFn) ([]uint16, error) {
	switch mode {
	case "tcp":
		return ListPorts(pathTCPTab, accept)
	case "tcp6":
		return ListPorts(pathTCP6Tab, accept)
	case "udp":
		return ListPorts(pathUDPTab, accept)
	case "udp6":
		return ListPorts(pathUDP6Tab, accept)
	default:
		return nil, fmt.Errorf("netstat: unknown mode: %v", mode)
	}
}

func osSocks(mode string, accept AcceptFn) ([]SockTabEntry, error) {
	switch mode {
	case "tcp":
		return DoNetstat(pathTCPTab, accept)
	case "tcp6":
		return DoNetstat(pathTCP6Tab, accept)
	case "udp":
		return DoNetstat(pathUDPTab, accept)
	case "udp6":
		return DoNetstat(pathUDP6Tab, accept)
	default:
		return nil, fmt.Errorf("netstat: unknown mode: %v", mode)
	}
}

// TCPSocks returns a slice of active TCP sockets containing only those
// elements that satisfy the accept function
func osTCPSocks(accept AcceptFn) ([]SockTabEntry, error) {
	return DoNetstat(pathTCPTab, accept)
}

// TCP6Socks returns a slice of active TCP IPv4 sockets containing only those
// elements that satisfy the accept function
func osTCP6Socks(accept AcceptFn) ([]SockTabEntry, error) {
	return DoNetstat(pathTCP6Tab, accept)
}

// UDPSocks returns a slice of active UDP sockets containing only those
// elements that satisfy the accept function
func osUDPSocks(accept AcceptFn) ([]SockTabEntry, error) {
	return DoNetstat(pathUDPTab, accept)
}

// UDP6Socks returns a slice of active UDP IPv6 sockets containing only those
// elements that satisfy the accept function
func osUDP6Socks(accept AcceptFn) ([]SockTabEntry, error) {
	return DoNetstat(pathUDP6Tab, accept)
}
