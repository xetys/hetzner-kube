# hetzner-kube: fast and easy setup of kubernetes clusters on Hetzner Cloud

This project contains a CLI tool to easily provision [kubernetes](https://kubernetes.io) clusters 
on [Hetzner Cloud](https://hetzner.com/cloud).

This is my very first tool written in Go. 

## How to install

Get the linux binary from releases page.

[Download version 0.0.3 - linux-amd64](https://github.com/xetys/hetzner-kube/releases/download/0.0.3/hetzner-kube)


or get it from source:

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

If you like to run some scripts or install some additional packages while provisioning new servers you can use cloud-init
```
$ hetzner-kube cluster create --name my-cluster --nodes 3 --ssh-key my-key --cloud-init <PATH-TO-FILE>
```
An example file to make all nodes ansible ready. The comment on the first line is important:
```yaml
#cloud-config
package_update: true
packages:
 - python
```

## Full tutorial

[This article](http://stytex.de/blog/2018/01/29/deploy-kubernetes-hetzner-cloud-openebs/) guides through a full
cluster setup.
