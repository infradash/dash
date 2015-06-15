.PHONY: setup

include hack/make/*.mk


all: dash

agent:
	cd pkg/agent && $(MAKE)

executor:
	cd pkg/executor && $(MAKE)

dash: agent executor build


setup:
	echo "Install godep, etc."
	./hack/env.sh
	echo "Clean up previous builds."
	rm -rf ./build/bin && mkdir -p ./build/bin

build: setup
	echo "Building dash"
	${GODEP} go build -o build/bin/dash -ldflags "$(LDFLAGS)" main/dash.go


# Simple local example -- assumes localhost zookeeper or SSH tunnel to zookeeper
# Local ssh tunnel:
# ssh -i decrypt/keys/bastion.cer -L 8080:zk1.prod.infradash.com:8080  -L 2181:zk1.prod.infradash.com:2181 ubuntu@bastion.infradash.com
run-local-agent:
	DASH_HOST=`hostname` \
	DASH_DOMAIN="accounts.qor.io" \
	DASH_TAGS="appserver,frontend" \
	DASH_NAME="dash" \
	DASH_ZK_HOSTS="localhost:2181" \
	DOCKER_PORT="tcp://192.168.59.103:2376" \
	${GODEP} go run main/dash.go --logtostderr --v=500 --self_register=false \
		--ui_docroot=$(HOME)/go/src/github.com/infradash/dash/www \
		--tlscert=$(HOME)/.boot2docker/certs/boot2docker-vm/cert.pem \
		--tlskey=$(HOME)/.boot2docker/certs/boot2docker-vm/key.pem \
		--tlsca=$(HOME)/.boot2docker/certs/boot2docker-vm/ca.pem \
		--config_url="file:///Users/david/go/src/github.com/infradash/dash/example/passport.json" \
	agent

run-exec-bash-export:
	DASH_DOMAIN="test.infradash.com" \
	DASH_ZK_HOSTS="localhost:2181" \
	${GODEP} go run main/dash.go \
		--service=infradash --version=develop \
		--custom_vars=EXEC_TS="{{.StartTimeUnix}},EXEC_DOMAIN={{.Domain}}" \
		--stdout --quote="'" --newline --bash_export \
	exec

run-exec-nginx:
	DASH_DOMAIN="test.infradash.com" \
	DASH_ZK_HOSTS="localhost:2181" \
	${GODEP} go run main/dash.go --logtostderr \
		--service=infradash --version=develop \
		--custom_vars=EXEC_TS="{{.StartTimeUnix}},EXEC_DOMAIN={{.Domain}}" \
		--daemon \
	    	--no_source_env=false \
		--config_url="http://infradash.github.io/ops-release/dash/profiles/test-nginx.json" \
	exec echo 'now={{.EXEC_TS}} and domain={{.EXEC_DOMAIN}} and db={{.DATABASE_HOST}}'

run-local-exec:
	DASH_DOMAIN="test.com" \
	DASH_ZK_HOSTS="localhost:2181" \
	${GODEP} go run main/dash.go --logtostderr \
		--service=infradash --version=develop \
		--custom_vars=EXEC_TS="{{.StartTimeUnix}},EXEC_DOMAIN={{.Domain}}" \
		--daemon=true \
	    	--no_source_env=false \
		--stdout --newline \
		--config_url="file:///Users/david/go/src/github.com/infradash/dash/example/executor-local.json" \
	exec echo {{.ENVIRONMENT_NAME}}

run-notty:
	DASH_DOMAIN="test.com" \
	DASH_ZK_HOSTS="localhost:2181" \
	DASH_NAME="test-notty" \
	${GODEP} go run main/dash.go --logtostderr \
		--service=infradash --version=develop \
		--daemon=false --ignore_child_process_fails=false \
		--custom_vars=EXEC_TS="{{.StartTimeUnix}},EXEC_DOMAIN={{.Domain}}" \
		--config_url="file:///Users/david/go/src/github.com/infradash/dash/example/task-notty.json" \
	exec ${CMD}

run-tty:
	DASH_DOMAIN="test.com" \
	DASH_ZK_HOSTS="localhost:2181" \
	DASH_NAME="test-tty" \
	${GODEP} go run main/dash.go --logtostderr \
		--service=infradash --version=develop \
		--daemon=true --ignore_child_process_fails=true \
		--custom_vars=EXEC_TS="{{.StartTimeUnix}},EXEC_DOMAIN={{.Domain}}" \
		--config_url="file:///Users/david/go/src/github.com/infradash/dash/example/task-tty.json" \
	exec ${CMD}

# Example: copy env from v0.1.2 to v0.1.3
run-publish-env:
	DASH_ZK_HOSTS="localhost:2181" \
	${GODEP} go run main/dash.go --logtostderr -publish -overwrite=false \
		--path=/sandbox.infradash.com/infradash/develop/env \
		--domain=production.infradash.com --service=infradash --version=develop \
	env

# Run a release
run-release:
	DASH_ZK_HOSTS="localhost:2181" \
	${GODEP} go run main/dash.go --logtostderr --commit \
		--release --commit \
		--domain=test.infradash.com \
		--service=infradash \
		--version=develop \
		--build=4287.133 \
		--image=infradash/infradash:develop-4287.133 \
	registry

run-release-scheduler-trigger:
	DASH_ZK_HOSTS="localhost:2181" \
	${GODEP} go run main/dash.go --logtostderr --commit \
		--release --commit \
		--image=qorio/passport:v1.0 \
		--scheduler_trigger_path="/test2.qoriolabs.com/passport/release" \
		--scheduler_image_path="/test2.qoriolabs.com/passport" \
	registry

# Run a setlive
run-setlive:
	DASH_ZK_HOSTS="localhost:2181" \
	${GODEP} go run main/dash.go --logtostderr \
		--setlive --commit --setlive_nowait \
		--domain=test.infradash.com \
		--service=infradash \
		--version=develop \
		--build=test \
		--image=infradash/infradash:develop-test \
	registry

run-writepath:
	DASH_ZK_HOSTS="localhost:2181" \
	${GODEP} go run main/dash.go --logtostderr \
		--commit --writepath=/test.infradash.com/test \
		--writevalue=test123 \
	registry

run-readpath:
	DASH_ZK_HOSTS="localhost:2181" \
	${GODEP} go run main/dash.go --logtostderr \
		--read \
		--readpath=/code.infradash.com/infradash \
	registry

run-circleci:
	DASH_ZK_HOSTS="localhost:2181" \
	${GODEP} go run main/dash.go --logtostderr \
		--circle_user=qorio \
		--circle_project=passport \
		--circle_token=d84e7b3e53035b9d8fc8a5aadbc2ad4237064e20 \
		--circle_buildnum=213 \
		--build_artifact_dir=/tmp/passport \
	circleci

run-circleci-zk:
	DASH_ZK_HOSTS="localhost:2181" \
	${GODEP} go run main/dash.go --logtostderr \
		--circle_auth_zkpath=/code.infradash.com/circleci/passport \
		--circle_buildnum=213 \
		--build_artifact_dir=/tmp/passport \
	circleci

test:
	${GODEP} go test ./... -check.vv -v 
