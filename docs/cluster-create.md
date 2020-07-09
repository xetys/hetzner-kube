# Creating a Cluster

Hetzner-kube allows you to easily create a [kubernetes](https://kubernetes.io/) cluster on [Hetzner Cloud](https://hetzner.com/cloud).

## Pre-requisites

### API token
You will need to generate an API token in your [Hetzner Console](https://console.hetzner.cloud/)

Configure hetzner-kube with the project and token by running the following command:

    $ hetzner-kube context add my-project
    Token: <PASTE-TOKEN-HERE>

### Configure SSH Key
You will need to add an SSH key by running the following command:

     $ hetzner-kube ssh-key add -n my-key
     
     // This assumes, you already have a SSH keypair ~/.ssh/id_rsa and ~/.ssh/id_rsa.pub
     
## Create Cluster
You can create a cluster by running the following command:

    $ hetzner-kube cluster create --name my-cluster --ssh-key my-key
    
### Options
The following custom options are available for the cluster create command:

- `--name`, `-n`: Name of the cluster
- `--ssh-key`, `-k`: Name of the SSH key used for provisioning
- `--master-server-type`: Server type used for masters , *options: cx11*
- `--worker-server-type`: Server type used for workers , *options: cx11*
- `--ha-enabled`: Install high-available control plane , *default: false*
- `--isolated-etcd`: Isolates etcd cluster from master nodes , *default: false*
- `--master-count`, `-m`: Number of master nodes, works only if `--ha-enabled` is passed, *default: 3*
- `--etcd-count`, `-e`: Number of etcd nodes, works only if `--ha-enabled` and `--isolated-etcd` are passed, *default: 3*
- `--worker-count`,`-w`: Number of worker nodes for the cluster , *default: 1*
- `--cloud-init`: Cloud-init file for server preconfiguration
- `--datacenters`: Can be used to filter datacenters by their name, *options: fsn-dc8, nbg1-dc3, hel1-dc2, fsn1-dc14*
- `--cni string`: The CNI you want to use, *default: canal*, *options: canal, calico*
