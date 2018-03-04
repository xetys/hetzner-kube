package cmd

import "testing"

func TestGenerateWireguardConf(t *testing.T) {
	nodes := []Node{
		{Name: "node1", IPAddress: "1.1.1.1", PrivateIPAddress: "10.0.0.1", WireGuardKeyPair: WgKeyPair{Private: "node1priv", Public: "node1pub"}},
		{Name: "node2", IPAddress: "1.1.1.2", PrivateIPAddress: "10.0.0.2", WireGuardKeyPair: WgKeyPair{Private: "node2priv", Public: "node2pub"}},
	}

	expectedConf := `[Interface]
Address = 10.0.0.2
PrivateKey = node2priv
ListenPort = 51820

# node1
[Peer]
PublicKey = node1pub
AllowedIps = 10.0.0.1/32
Endpoint = 1.1.1.1:51820
`

	generatedConf := GenerateWireguardConf(nodes[1], nodes)

	if generatedConf != expectedConf {
		t.Errorf("The file was not rendered as expected\n%s\n\n", generatedConf)
	}

}
