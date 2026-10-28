package discovery

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/net/ipv4"
)

const (
	DefaultGroup  = "239.254.0.2"
	DefaultPort   = 9910
	MessageHeader = "$VC-DISCOVER-1.0;json;"
)

type Peer struct {
	InstanceID     string    `json:"instanceId"`
	InstanceName   string    `json:"instanceName"`
	InstanceType   string    `json:"instanceType"`
	DeploymentMode string    `json:"deploymentMode"`
	HTTPPort       int       `json:"httpPort"`
	APIPort        int       `json:"apiPort"`
	PortalPort     int       `json:"portalPort"`
	Version        string    `json:"version"`
	Timestamp      int64     `json:"timestamp"`
	NodeAddresses  []string  `json:"nodeAddresses,omitempty"`
	SourceIP       string    `json:"sourceIp"`
	SeenAt         time.Time `json:"seenAt"`
}

func (p Peer) BaseURL() string {
	host := p.SourceIP
	if host == "" && len(p.NodeAddresses) > 0 {
		host = p.NodeAddresses[0]
	}
	port := p.HTTPPort
	if port == 0 {
		port = 9909
	}
	return fmt.Sprintf("http://%s:%d", host, port)
}

// Listen joins the Vector Charts discovery multicast group and collects unique
// peers until timeout elapses.
func Listen(group string, port int, timeout time.Duration) ([]Peer, error) {
	if group == "" {
		group = DefaultGroup
	}
	if port == 0 {
		port = DefaultPort
	}

	addr := &net.UDPAddr{IP: net.IPv4zero, Port: port}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return nil, fmt.Errorf("listen udp :%d: %w", port, err)
	}
	defer conn.Close()

	pconn := ipv4.NewPacketConn(conn)
	groupAddr := &net.UDPAddr{IP: net.ParseIP(group)}
	ifaces, _ := net.Interfaces()
	joined := false
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagMulticast == 0 {
			continue
		}
		if err := pconn.JoinGroup(&iface, groupAddr); err == nil {
			joined = true
		}
	}
	if !joined {
		if err := pconn.JoinGroup(nil, groupAddr); err != nil {
			return nil, fmt.Errorf("join multicast %s: %w", group, err)
		}
	}
	_ = pconn.SetMulticastLoopback(true)

	deadline := time.Now().Add(timeout)
	_ = conn.SetReadDeadline(deadline)

	byID := map[string]Peer{}
	buf := make([]byte, 65535)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			return nil, err
		}
		msg := string(buf[:n])
		if !strings.HasPrefix(msg, MessageHeader) {
			continue
		}
		raw := strings.TrimPrefix(msg, MessageHeader)
		var peer Peer
		if err := json.Unmarshal([]byte(raw), &peer); err != nil {
			continue
		}
		if peer.InstanceType != "" && peer.InstanceType != "vector-charts" {
			continue
		}
		peer.SourceIP = src.IP.String()
		peer.SeenAt = time.Now()
		key := peer.InstanceID
		if key == "" {
			key = peer.SourceIP
		}
		byID[key] = peer
	}

	out := make([]Peer, 0, len(byID))
	for _, p := range byID {
		out = append(out, p)
	}
	return out, nil
}
