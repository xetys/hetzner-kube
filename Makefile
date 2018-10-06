HETZNER_KUBE=${TRAVIS_BUILD_DIR}/hetzner-kube

SSH_KEY_PATH_PRIVATE=${TRAVIS_BUILD_DIR}/tests/id_rsa
SSH_KEY_PATH_PUBLIC=${SSH_KEY_PATH_PRIVATE}.pub
DATACENTER=fsn1-dc8
SERVER_MASTER_COUNT=1
SERVER_WORKER_COUNT=1

SSH_KEY_NAME=testing-ssh-key-${TRAVIS_JOB_NUMBER}
CONTEXT_NAME=testing-context-${TRAVIS_JOB_NUMBER}
CLUSTER_NAME=testing-cluster-${TRAVIS_JOB_NUMBER}

info:
	${HETZNER_KUBE} help
	${HETZNER_KUBE} version

context:
	${HETZNER_KUBE} context add ${CONTEXT_NAME} --token ${HETZNER_API_KEY}
	${HETZNER_KUBE} context list
	${HETZNER_KUBE} context use ${CONTEXT_NAME}
	${HETZNER_KUBE} context current
	# ${HETZNER_KUBE} context delete ${CONTEXT_NAME}

ssh-key:
	mkdir ${TRAVIS_BUILD_DIR}/tests
	ssh-keygen -b 2048 -t rsa -f ${SSH_KEY_PATH_PRIVATE} -q -N ""
	${HETZNER_KUBE} ssh-key add --name ${SSH_KEY_NAME} --private-key-path ${SSH_KEY_PATH_PRIVATE} --public-key-path ${SSH_KEY_PATH_PUBLIC}
	${HETZNER_KUBE} ssh-key list
	# ${HETZNER_KUBE} ssh-key delete --name ${SSH_KEY_NAME}

cluster:
	${HETZNER_KUBE} cluster create --worker-count ${SERVER_WORKER_COUNT} --datacenters ${DATACENTER} --ssh-key ${SSH_KEY_NAME} --name ${CLUSTER_NAME}
	${HETZNER_KUBE} cluster list
	${HETZNER_KUBE} cluster master-ip ${CLUSTER_NAME}
	${HETZNER_KUBE} cluster kubeconfig ${CLUSTER_NAME}
	${HETZNER_KUBE} cluster add-worker --nodes 1 --datacenters ${DATACENTER} --name ${CLUSTER_NAME}
	# ${HETZNER_KUBE} cluster delete ${CLUSTER_NAME}

cleanup:
	${HETZNER_KUBE} cluster delete ${CLUSTER_NAME}
	${HETZNER_KUBE} ssh-key delete --name ${SSH_KEY_NAME}
	${HETZNER_KUBE} context delete ${CONTEXT_NAME}
