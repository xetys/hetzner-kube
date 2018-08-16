HETZNER_KUBE=${TRAVIS_BUILD_DIR}/hetzner-kube

SSH_KEY_NAME=testing-key
CONTEXT_NAME=testing-${TRAVIS_JOB_NUMBER}
DATACENTER=fsn1-dc8
CLUSTER_NAME=testing-cluster-${TRAVIS_JOB_NUMBER}
SERVER_MASTER_COUNT=1
SERVER_WORKER_COUNT=1

info:
	${HETZNER_KUBE} help
	${HETZNER_KUBE} version

context:
	${HETZNER_KUBE} context add ${CONTEXT_NAME} -t ${HETZNER_API_KEY}
	${HETZNER_KUBE} context list
	${HETZNER_KUBE} context use ${CONTEXT_NAME}
	${HETZNER_KUBE} context current
	${HETZNER_KUBE} context delete ${CONTEXT_NAME}