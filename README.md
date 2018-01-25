# hetzner-kube: fast and easy setup of kubernetes clusters on Hetzner Cloud

This project contains a CLI tool to easily provision [kubernetes](https://kubernetes.io) clusters 
on [Hetzner Cloud](https://hetzner.com/cloud).

This is my very first tool written in Go. 

## How to install

Currently, the only way is

```
$ go get -u github.com/xetys/hetzner-kube
```

## Usage

In your [Hetzner Console](https://console.hetzner.cloud) generate an API token and

```
$ hetzner-kube context add my-project
Token: <PASTE-TOKEN-HERE>
```

Then you need to add an SSH key:

```
$ hetzner-kube ssh-key add -n my-key
```

This assumes, you already have a SSH keypair `~/.ssh/id_rsa` and `~/.ssh/id_rsa.pub`

And finally you can create a cluster by running:

```
$ hetzner-kube cluster create --name my-cluster --nodes 3 --ssh-key my-key

```

This will provision a brand new kubernetes cluster in latest version!


More info will come as more development happens here...
