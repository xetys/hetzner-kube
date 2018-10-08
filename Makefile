HETZNER_KUBE=${TRAVIS_BUILD_DIR}/hetzner-kube

SSH_KEY_FOLDER=${TRAVIS_BUILD_DIR}/tests/keys
DATACENTER=fsn1-dc8

SSH_KEY_NAME=testing-ssh-key-${TRAVIS_JOB_NUMBER}
CONTEXT_NAME=testing-context-${TRAVIS_JOB_NUMBER}
CLUSTER_NAME=testing-cluster-${TRAVIS_JOB_NUMBER}

VERSION=${TRAVIS_TAG}

build-cleanup:
	@rm -rf dist/*

build-prepare:
	@dep ensure

build: build-cleanup build-prepare
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -ldflags "-X cmd.version=${VERSION}" -o dist/hetzner-kube-${VERSION}-linux-amd64
	CGO_ENABLED=0 GOOS=linux   GOARCH=386   go build -ldflags "-X cmd.version=${VERSION}" -o dist/hetzner-kube-${VERSION}-linux-386
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm   go build -ldflags "-X cmd.version=${VERSION}" -o dist/hetzner-kube-${VERSION}-linux-arm
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -ldflags "-X cmd.version=${VERSION}" -o dist/hetzner-kube-${VERSION}-linux-arm64
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -ldflags "-X cmd.version=${VERSION}" -o dist/hetzner-kube-${VERSION}-darwin-amd64
	CGO_ENABLED=0 GOOS=darwin  GOARCH=386   go build -ldflags "-X cmd.version=${VERSION}" -o dist/hetzner-kube-${VERSION}-darwin-386
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-X cmd.version=${VERSION}" -o dist/hetzner-kube-${VERSION}-windows-amd64.exe
	CGO_ENABLED=0 GOOS=windows GOARCH=386   go build -ldflags "-X cmd.version=${VERSION}" -o dist/hetzner-kube-${VERSION}-windows-386.exe

preparare:
	mkdir -p ${SSH_KEY_FOLDER}
	ssh-keygen -t rsa -b 4096 -P "" -f ${SSH_KEY_FOLDER}/id_rsa

test-all: test-info test-context test-ssh-key test-clusters

test-info:
	${HETZNER_KUBE} help
	${HETZNER_KUBE} version

test-context:
	${HETZNER_KUBE} context add ${CONTEXT_NAME} --token ${HETZNER_API_KEY}
	${HETZNER_KUBE} context list
	${HETZNER_KUBE} context use ${CONTEXT_NAME}
	${HETZNER_KUBE} context current
	# We move it on clanup since they are used to test cluster operations
	# ${HETZNER_KUBE} context delete ${CONTEXT_NAME}

test-ssh-key:
	${HETZNER_KUBE} ssh-key add --name ${SSH_KEY_NAME} --private-key-path ${SSH_KEY_FOLDER}/id_rsa --public-key-path ${SSH_KEY_FOLDER}/id_rsa.pub
	${HETZNER_KUBE} ssh-key list
	# We move it on clanup since they are used to test cluster operations
	# ${HETZNER_KUBE} ssh-key delete --name ${SSH_KEY_NAME}

test-clusters: test-cluster-without-ha test-cluster-with-ha-level3 test-cluster-with-ha-level4

test-cluster-without-ha:
	${HETZNER_KUBE} cluster create --worker-count 1 --datacenters ${DATACENTER} --ssh-key ${SSH_KEY_NAME} --name ${CLUSTER_NAME}-without-ha
	${HETZNER_KUBE} cluster list
	${HETZNER_KUBE} cluster master-ip ${CLUSTER_NAME}-without-ha
	${HETZNER_KUBE} cluster kubeconfig -f ${CLUSTER_NAME}-without-ha
	${HETZNER_KUBE} cluster add-worker --nodes 1 --datacenters ${DATACENTER} --name ${CLUSTER_NAME}-without-ha
	${HETZNER_KUBE} cluster delete ${CLUSTER_NAME}-without-ha

test-cluster-with-ha-level3:
	${HETZNER_KUBE} cluster create --worker-count 1 --master-count 3 --ha-enabled --datacenters ${DATACENTER} --ssh-key ${SSH_KEY_NAME} --name ${CLUSTER_NAME}-with-ha-level3
	${HETZNER_KUBE} cluster list
	${HETZNER_KUBE} cluster master-ip ${CLUSTER_NAME}-with-ha-level3
	${HETZNER_KUBE} cluster kubeconfig -f ${CLUSTER_NAME}-with-ha-level3
	${HETZNER_KUBE} cluster add-worker --nodes 1 --datacenters ${DATACENTER} --name ${CLUSTER_NAME}-with-ha-level3
	${HETZNER_KUBE} cluster delete ${CLUSTER_NAME}-with-ha-level3

test-cluster-with-ha-level4:
	${HETZNER_KUBE} cluster create --worker-count 1 --master-count 3 --etcd-count 3 --ha-enabled --isolated-etcd --datacenters ${DATACENTER} --ssh-key ${SSH_KEY_NAME} --name ${CLUSTER_NAME}-with-ha-level4
	${HETZNER_KUBE} cluster list
	${HETZNER_KUBE} cluster master-ip ${CLUSTER_NAME}-with-ha-level4
	${HETZNER_KUBE} cluster kubeconfig -f ${CLUSTER_NAME}-with-ha-level4
	${HETZNER_KUBE} cluster add-worker --nodes 1 --datacenters ${DATACENTER} --name ${CLUSTER_NAME}-with-ha-level4
	${HETZNER_KUBE} cluster delete ${CLUSTER_NAME}-with-ha-level4

cleanup:
	${HETZNER_KUBE} ssh-key delete --name ${SSH_KEY_NAME}
	${HETZNER_KUBE} context delete ${CONTEXT_NAME}

hard-cleanup:
	curl -s --request GET --url https://api.hetzner.cloud/v1/servers --header "Authorization: Bearer ${HETZNER_API_KEY}" | jq '.servers[] | "\(.name) \(.id)"' | grep ${TRAVIS_JOB_NUMBER} | awk 'BEGIN { FS = "\"" } ; { print $$2 }' | awk '{ print $$2 }' | xargs -I '{}' bash -c "curl -s --request DELETE https://api.hetzner.cloud/v1/servers/{} --header \"Authorization: Bearer ${HETZNER_API_KEY}\""
	curl -s --request GET --url https://api.hetzner.cloud/v1/ssh_keys --header "Authorization: Bearer ${HETZNER_API_KEY}" | jq '.ssh_keys[] | "\(.name) \(.id)"' | grep ${TRAVIS_JOB_NUMBER} | awk 'BEGIN { FS = "\"" } ; { print $$2 }' | awk '{ print $$2 }' | xargs -I '{}' bash -c "curl -s --request DELETE https://api.hetzner.cloud/v1/ssh_keys/{} --header \"Authorization: Bearer ${HETZNER_API_KEY}\""
