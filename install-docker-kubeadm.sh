#!/bin/bash

installPackages() {
echo "
Package: docker-ce
Pin: version 17.03.*
Pin-Priority: 1000
" > /etc/apt/preferences.d/docker-ce

apt-get update
# transport stuff
apt-get install -y \
    apt-transport-https \
    ca-certificates \
    curl \
    software-properties-common

# docker-ce
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -
add-apt-repository \
   "deb https://download.docker.com/linux/$(. /etc/os-release; echo "$ID") \
   $(lsb_release -cs) \
   stable"
# kubernetes

curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
cat <<EOF >/etc/apt/sources.list.d/kubernetes.list
deb http://apt.kubernetes.io/ kubernetes-xenial main
EOF

# prepare wireguard
add-apt-repository ppa:wireguard/wireguard -y

apt-get update && apt-get install -y docker-ce
apt-get install -y docker-ce kubelet kubeadm kubectl wireguard linux-headers-$(uname -r)

# prepare for hetzners cloud controller manager
mkdir -p /etc/systemd/system/kubelet.service.d
cat > /etc/systemd/system/kubelet.service.d/20-hcloud.conf << EOM
[Service]
Environment="KUBELET_EXTRA_ARGS=--cloud-provider=external"
EOM

systemctl daemon-reload
}

S=$(type -p kubeadm > /dev/null &> /dev/null; echo $?)
while [ ${S} = 1 ]; do
    echo "installing packages..."
    installPackages
    S=$(type -p kubeadm > /dev/null &> /dev/null; echo $?)
done;
