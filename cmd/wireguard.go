package cmd

import (
	"fmt"
	"encoding/json"
)

type WgKeyPair struct {
	Private string `json:"private"`
	Public string `json:"public"`
}

func GenerateKeyPairs(node Node, count int) []WgKeyPair {
	genKeyPairs := fmt.Sprintf(`echo "[" ;for i in {1..%d}; do pk=$(wg genkey); pubk=$(echo $pk | wg pubkey);echo "{\"private\":\"$pk\",\"public\":\"$pubk\"},"; done; echo "]";`, count)
	// gives an invalid JSON back
	o, err := runCmd(node, genKeyPairs)
	FatalOnError(err)
	o = o[0:len(o)-4] + "]"
	// now it's a valid json

	var keyPairs []WgKeyPair
	err = json.Unmarshal([]byte(o), &keyPairs)
	FatalOnError(err)

	return keyPairs
}

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
