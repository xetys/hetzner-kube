
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

#### BASH

To load completion run

```bash
source <(hetzner-kube completion bash)
```

To configure your bash shell to load completions for each session add to your "~/.bashrc" file

```bash
# ~/.bashrc or ~/.profile
echo 'source <(hetzner-kube completion bash)\n' >> ~/.bashrc
```

Or you can add it to your `bash_completition.d` folder:

```bash
# On linux
hetzner-kube completion bash > /etc/bash_completion.d/hetzner-kube
# On OSX with completion installed via brew (`brew install bash-completion`)
hetzner-kube completion bash > /usr/local/etc/bash_completion.d/hetzner-kube
```

#### ZSH

To configure your zsh shell to load completions run following commands:

```bash
# On linux
hetzner-kube completion zsh | sudo tee /usr/share/zsh/vendor-completions/_hetzner-kube
# On OSX
hetzner-kube completion zsh | sudo tee /usr/share/zsh/site-functions/_hetzner-kube
```

Than rebuild autocomplete function with:

```
compinits
```

### Docker image

You can use our [Docker image](https://hub.docker.com/r/xetys/hetzner-kube/) to run `hetzner-kube` without any need to install Go, e.g.

```bash
$ docker run --rm xetys/hetzner-kube
```
```bash
$ docker run --rm xetys/hetzner-kube ./hetzner-kube cluster list
```

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

To access the cluster via kubectl, create a config file:

```bash
# Write ~/.kube/config (you'll be asked to overwrite if file already exists)
hetzner-kube cluster kubeconfig my-cluster
# Alternatively, create a separate file and point kubectl to it:
#hetzner-kube cluster kubeconfig --print my-cluster > ~/.kube/config-my-cluster
#export KUBECONFIG=~/.kube/config-my-cluster
```

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

### contributing new addons

You want to add some cool stuff to hetzner-kube? It's quite easy! Learn how to add new addons in the [Developing Addons](docs/cluster-addons.md) documentation.

## backup & restore (for HA-clusters)

If you are running an external etcd cluster, you can use the etcd management to backup the
etcd cluster using

```bash
$ hetzner-kube cluster etcd backup my-cluster --snapshot-name my-snapshot
```

(`--snapshot-name` is optional)

and restore it using

```bash
$ hetzner-kube cluster etcd restore my-cluster --snapshot-name my-snapshot
```

If you place a different snapshot (with `.db` file extension) in /root/etcd-snapshots of the
first etcd node, you can use the restore command for migration of kubernetes clusters.


## phases

After a cluster is once created (nodes creation is enough), you can perform the steps from
`hetzner-kube cluster create` separately. 

You can use
``` bash
$ hetzner-kube cluster phase -h
```

to only list all available phases.

Cluster creation is a chain of these phases:

- `provision`
- `network-setup`
- `etcd` (if HA)
- `install-masters`
- `setup-ha` (if HA)
- `install-workers`

Some of theses phase have additional options, to run the actual phase differently from the
usual cluster creation.

### examples for phases

To simply run a phase from cluster creation, you cam run a phase, eg. the `etcd` phase using:

```bash
$ hetzner-kube cluster phase etcd my-cluster
```

In order to upgrade or migrate a cluster, you might want to keep the etcd data, and run

```bash
$ hetzner-kube cluster phase etcd my-cluster --keep-data
```

and preserve the existing certificates using

```bash
$ hetzner-kube cluster phase install-master my-cluster --keep-all-certs
```

The latter command can be also useful during cluster migration, if you place the existing certs
in /etc/kubernetes/pki before running the `install-masters` phase.

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
