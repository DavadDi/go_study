package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

// From https://gist.github.com/creack/43ee6542ddc6fe0da8c02bd723d5cc53#file-dial_from_iface-go-L85
// Dialer .

type Dialer struct {
	laddrIP string
	err     error
	dialer  *net.Dialer
}

// DialFromInterface .
func DialFromInterface(ifaceName string) *Dialer {
	d := &Dialer{}

	// Lookup rquested interface.
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		d.err = err
		return d
	}

	// Pull the addresses.
	addres, err := iface.Addrs()
	if err != nil {
		d.err = err
		return d
	}

	// Look for the first usable address.
	var targetIP string
	for _, addr := range addres {
		ip, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			d.err = err
			return d
		}
		if ip.IsUnspecified() {
			continue
		}
		if ip.To4().Equal(ip) {
			targetIP = ip.String()
		} else {
			targetIP = "[" + ip.String() + "]"
		}
	}
	if targetIP == "" {
		d.err = fmt.Errorf("no ipv4 found for interface")
		return d
	}
	d.laddrIP = targetIP
	return d
}

func DialFromIP(ipaddr string) *Dialer {
	d := &Dialer{laddrIP: ipaddr}
	return d
}

func (d *Dialer) lookupAddr(network, addr string) (net.Addr, error) {
	if d.err != nil {
		return nil, d.err
	}
	// If no custom dialer specified, use default one.
	if d.dialer == nil {
		d.dialer = &net.Dialer{}
	}

	// Resolve the address.
	switch network {
	case "tcp", "tcp4", "tcp6":
		addr, err := net.ResolveTCPAddr(network, d.laddrIP+":0")
		return addr, err
	case "udp", "udp4", "udp6":
		addr, err := net.ResolveUDPAddr(network, d.laddrIP+":0")
		return addr, err
	default:
		return nil, fmt.Errorf("unkown network")
	}
}

// Dial .
func (d *Dialer) Dial(network, addr string) (net.Conn, error) {
	laddr, err := d.lookupAddr(network, addr)
	if err != nil {
		return nil, err
	}
	d.dialer.LocalAddr = laddr
	return d.dialer.Dial(network, addr)
}

// DialTimeout .
func (d *Dialer) DialTimeout(network, addr string, timeout time.Duration) (net.Conn, error) {
	laddr, err := d.lookupAddr(network, addr)
	if err != nil {
		return nil, err
	}
	d.dialer.Timeout = timeout
	d.dialer.LocalAddr = laddr
	return d.dialer.Dial(network, addr)
}

// WithDialer .
func (d *Dialer) WithDialer(dialer net.Dialer) *Dialer {
	d.dialer = &dialer
	return d
}

var (
	count     int
	server    string
	bind      string
	frequency int
	help      bool
)

func init() {
	flag.IntVar(&count, "c", 10000, "count, default 10000")
	flag.StringVar(&server, "s", "192.168.0.21:30096", "server default 192.168.0.21:30096")
	flag.StringVar(&bind, "b", "127.0.0.1", "bind local ip addr, default 127.0.0.1")
	flag.IntVar(&frequency, "f", 2, "start one connect interval (ms) defalut 2ms")
	flag.BoolVar(&help, "h", false, "show help")
}

func showUsage(args []string) {
	fmt.Printf(" %s help", args[0])
	fmt.Println("\t-c total count, default 10000")
	fmt.Println("\t-s server default 192.168.0.21:30096")
	fmt.Println("\t-b bind local ip addr, default 127.0.0.1")
	fmt.Println("\t-f start one connect interval (ms) defalut 2ms")
	fmt.Println("\t-h show help")

}

func main() {
	flag.Parse()

	if help {
		showUsage(os.Args)
		os.Exit(1)
	}

	conns := make([]net.Conn, 0, count)

	for i := 1; i <= count; i++ {
		conn, err := DialFromIP(bind).WithDialer(net.Dialer{KeepAlive: 2 * time.Second}).Dial("tcp", server)
		if err != nil {
			log.Fatal(err)
		}

		if i%1000 == 0 {
			fmt.Println("Conns: ", i)
		}

		time.Sleep(time.Microsecond * time.Duration(frequency))

		conns = append(conns, conn)
	}

	fmt.Println("We are going to sleep 2 hour")
	time.Sleep(time.Hour * 2)

	for _, conn := range conns {
		conn.Close()
	}
}
