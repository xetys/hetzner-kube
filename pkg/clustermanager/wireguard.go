package clustermanager

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
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

// PrivateIPPrefix extracts the first 3 digits of an IPv4 address
func PrivateIPPrefix(ip string) string {
	return strings.Join(strings.Split(ip, ".")[:3], ".")
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
