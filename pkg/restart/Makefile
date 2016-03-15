all: test-restart

# Local zookeeper running in a container using Docker Tools
ZK_HOSTS?=192.168.99.100:2181

test-restart:
	ZK_HOSTS=$(ZK_HOSTS) \
	${GODEP} go test ./... -check.vv -v -logtostderr ${TEST_ARGS}
