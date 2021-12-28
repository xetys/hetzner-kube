package clustermanager

import "fmt"

func GenerateHaProxyConfiguration(masterNodes []Node) string {
	haProxyCfgTpl := `
global
        log /dev/log    local0
        log /dev/log    local1 notice
        chroot /var/lib/haproxy
        stats socket /run/haproxy/admin.sock mode 660 level admin expose-fd listeners
        stats timeout 30s
        user haproxy
        group haproxy
        daemon

defaults
        log     global
        mode    tcp
        option  httplog
        option  dontlognull
        timeout connect 5s
        timeout client  10s
        timeout server  10s
		balance source

frontend rkeserver
  bind :19345
  default_backend rkeservers

frontend apiserver
  bind :16443
  default_backend apiservers

backend rkeservers
%s

backend apiservers
%s
`
	rkeServers := ""
	apiServers := ""

	for i, node := range masterNodes {
		rkeServers = fmt.Sprintf("%s  server s%d %s:9345 check\n", rkeServers, i, node.IPAddress)
		apiServers = fmt.Sprintf("%s  server s%d %s:6443 check\n", apiServers, i, node.IPAddress)
	}

	return fmt.Sprintf(haProxyCfgTpl, rkeServers, apiServers)
}
