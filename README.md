
[![Build Status](https://travis-ci.org/xetys/hetzner-kube.svg?branch=master)](https://travis-ci.org/xetys/hetzner-kube)
[![Go Report Card](https://goreportcard.com/badge/github.com/xetys/hetzner-kube)](https://goreportcard.com/report/github.com/xetys/hetzner-kube)
[![Maintainability](https://api.codeclimate.com/v1/badges/3ef5b31a84811e3b8b02/maintainability)](https://codeclimate.com/github/xetys/hetzner-kube/maintainability)
[![Gitter chat](https://badges.gitter.im/hetzner-kube.png)](https://gitter.im/hetzner-kube)

# hetzner-kube: fast and easy setup of kubernetes clusters on Hetzner Cloud

This project contains a CLI tool to easily provision [kubernetes](https://kubernetes.io) clusters
on [Hetzner Cloud](https://hetzner.com/cloud).

This is my very first tool written in Go.

## How to install

### Binary releases

Get the binary from [releases page](https://github.com/xetys/hetzner-kube/releases)

### From source

`hetzner-kube` is written in [Go](https://golang.org/). To install Go please follow the instructions on its homepage.

To get and build `hetzner-kube` from source run this command:

```
$ go get -u github.com/xetys/hetzner-kube
```

The project source will now be in your `$GOPATH` directory (`$GOPATH/src/github.com/xetys/hetzner-kube`) and the binary will be in `$GOPATH/bin`.

If you want to build it yourself later, you can change into the source directory and run `go build` or `go install`.

### Code completion

To load completion for bash env run

    . <(hetzner-kube completion)

To configure your bash shell to load completions for each session add to your `.bashrc` file:

    # ~/.bashrc or ~/.profile
    . <(hetzner-kube completion)

Or you can add it to your `bash_completition.d` folder:

    # On linux
    hetzner-kube completion > /etc/bash_completion.d/hetzner-kube
    # On OSX with completion installed via brew (`brew intall bash-completion`)
    hetzner-kube completion > /usr/local/etc/bash_completion.d/hetzner-kube

## Usage

In your [Hetzner Console](https://console.hetzner.cloud) generate an API token and

```bash
$ hetzner-kube context add my-project
Token: <PASTE-TOKEN-HERE>
```

Then you need to add an SSH key:

```bash
$ hetzner-kube ssh-key add -n my-key
```

This assumes, you already have a SSH keypair `~/.ssh/id_rsa` and `~/.ssh/id_rsa.pub`

And finally, you can create a cluster by running:

```bash
$ hetzner-kube cluster create --name my-cluster --ssh-key my-key

```

This will provision a brand new kubernetes cluster in latest version!

For a full list of options that can be passed to the ```cluster create``` command, see the [Cluster Create Guide](docs/cluster-create.md) for more information.

## HA-clusters

You can build high available clusters with hetzner-kube. Read the [High availability Guide](docs/high-availability.md) for
further information.

# Custom Options

## addons

You can install some addons to your cluster using the `cluster addon` sub-command. Get a list of addons using:

```bash
$ hetzner-kube cluster addon list
```

## Script Runner Addon

The addon lets you run bash scripts on the running nodes, check [Script Runner Guide](docs/script-runner-addon.md) for more info. 

### contributing new addons

You want to add some cool stuff to hetzner-kube? It's quite easy! Learn how to add new addons in the [Developing Addons](docs/cluster-addons.md) documentation.

## cloud-init

If you like to run some scripts or install some additional packages while provisioning new servers, you can use cloud-init
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

[This article](http://stytex.de/blog/2018/01/29/deploy-kubernetes-hetzner-cloud-openebs/) guides through a full cluster setup.
