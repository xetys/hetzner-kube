package clustermanager

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"strings"

	"golang.org/x/crypto/curve25519"
)

//WgKeyPair containse key pairs
type WgKeyPair struct {
	Private string `json:"private"`
	Public  string `json:"public"`
}

//GenerateWireguardConf generate wireguard configuration file
func GenerateWireguardConf(node Node, nodes []Node) string {
	var output string
	// print header block
	headerTpl := `[Interface]
Address = %s
PrivateKey = %s
ListenPort = 51820
`
	peerTpl := `# %s
[Peer]
PublicKey = %s
AllowedIps = %s/32
Endpoint = %s:51820
`
	output = fmt.Sprintf(headerTpl, node.PrivateIPAddress, node.WireGuardKeyPair.Private)

	for _, peer := range nodes {
		if peer.Name == node.Name {
			continue
		}

		output = fmt.Sprintf("%s\n%s",
			output,
			fmt.Sprintf(peerTpl, peer.Name, peer.WireGuardKeyPair.Public, peer.PrivateIPAddress, peer.IPAddress),
		)
	}

	return output
}

// PrivateIPPrefix extracts the first 3 digits of an IPv4 address from CIDR block
func PrivateIPPrefix(cidr string) (string, error) {
	ipAddress, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("unable to parse cidr %q", cidr)
	}
	ipAddress = ipAddress.To4()
	if ipAddress == nil {
		return "", fmt.Errorf("unable to convert ip %q to IPv4s", ipAddress)
	}

	return strings.Join(strings.Split(ipAddress.String(), ".")[:3], "."), nil
}

// GenerateKeyPair create a key-pair used to instantiate a wireguard connection
// Code is redacted from https://github.com/WireGuard/wireguard-go/blob/1c025570139f614f2083b935e2c58d5dbf199c2f/noise-helpers.go
func GenerateKeyPair() (WgKeyPair, error) {
	var publicKey [32]byte
	var privateKey [32]byte
	_, err := rand.Reader.Read(privateKey[:])
	if err != nil {
		return WgKeyPair{}, fmt.Errorf("unable to generate a private key: %v", err)
	}

	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	curve25519.ScalarBaseMult(&publicKey, &privateKey)

	return WgKeyPair{
		Private: base64.StdEncoding.EncodeToString(privateKey[:]),
		Public:  base64.StdEncoding.EncodeToString(publicKey[:]),
	}, nil
}

//GenerateEtcdSystemdService generate configuration file used to manage etcd service on systemd
func GenerateOverlayRouteSystemdService(node Node) string {
	serviceTpls := `# /etc/systemd/system/overlay-route.service
[Unit]
Description=Overlay network route for Wireguard
After=wg-quick@wg0.service

[Service]
Type=oneshot
User=root
ExecStart=/sbin/ip route add 10.244.0.0/16 dev wg0 src %s

[Install]
WantedBy=multi-user.target
`
	service := fmt.Sprintf(serviceTpls, node.PrivateIPAddress)

	return service
}
