package clustermanager

import "log"

func waitOrError(tc chan bool, ec chan error, numProcPtr *int) error {
	numProcs := *numProcPtr
	for numProcs > 0 {
		select {
		case err := <-ec:
			return err
		case <-tc:
			numProcs--
		}
	}

	return nil
}

//Node2IP get IP address for a node
func Node2IP(node Node) string {
	return node.IPAddress
}

//Nodes2IPs get the collection of IP addresses for a node
func Nodes2IPs(nodes []Node) []string {
	ips := []string{}
	for _, node := range nodes {
		ips = append(ips, Node2IP(node))
	}

	return ips
}

//FatalOnError is an helper function used to transfor error to fatal
func FatalOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
