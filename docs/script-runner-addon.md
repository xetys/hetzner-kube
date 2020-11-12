# Script Runner Addon Guide

Script Runner lets you run a bash script on all your cluster nodes with one command. It uploads the bash script to each node then execute it. The script will receive two arguments upon execution:
1. Node group, allowed values are `master`, `worker` and `etcd`
2. Cluster info JSON config, which is similar to the hetzner-kube config found in `~/.hetzner-kube/config.json`

### Example

hetzner-kube Command:
```bash
hetzner-kube cluster addon install --name demo script-runner /path/to/script.sh  
```
Example script.sh:
```bash
#!/bin/bash

#node group
echo $1
#cluster info 
echo $2

if [ "$1" == "worker" ]; then
    apt-get install ceph-fs-common ceph-common -y
fi


echo $2 | python -c "import sys, json; print json.load(sys.stdin)[0]['name']"

```

