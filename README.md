
[![Build Status](https://travis-ci.org/xetys/hetzner-kube.svg?branch=master)](https://travis-ci.org/xetys/hetzner-kube)


# hetzner-kube: fast and easy setup of kubernetes clusters on Hetzner Cloud

This project contains a CLI tool to easily provision [kubernetes](https://kubernetes.io) clusters 
on [Hetzner Cloud](https://hetzner.com/cloud).

This is my very first tool written in Go. 

## How to install

Get the linux binary from releases page.

[Download version 0.2.0 - linux-amd64](https://github.com/xetys/hetzner-kube/releases/download/0.2.0/hetzner-kube)


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
$ hetzner-kube cluster create --name my-cluster --ssh-key my-key

```

This will provision a brand new kubernetes cluster in latest version!
## HA-clusters

You can built high available clusters with hetzner-kube. Read the [High availability Guide](docs/high-availability.md) for
further information.

# Custom Options 

## addons

You can install some addons to your cluster using the `cluster addon` sub-command. Currently these are the supported first addons:

* helm
* [Rook](https://rook.io)
* OpenEBS
* NGinx ingress controller (requires helm)

### contributing new addons

Feel free to contribute cluster addons. You can simply create one by implementing the `ClusterAddon` interface and 
adding it to the addons.

## cloud-init

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
