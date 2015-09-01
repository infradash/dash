
all: test-circleci

test-circleci:
	${GODEP} go test ./... -check.vv -v -logtostderr ${TEST_ARGS}
