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

func Node2IP(node Node) string {
	return node.IPAddress
}

func Nodes2IPs(nodes []Node) []string {
	ips := []string{}
	for _, node := range nodes {
		ips = append(ips, Node2IP(node))
	}

	return ips
}

func FatalOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
