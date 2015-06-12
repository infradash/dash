.PHONY: _pwd_prompt dec enc

all: dash

agent:
	cd pkg/agent && $(MAKE)

executor:
	cd pkg/executor && $(MAKE)

dash: agent executor compile


# 'private' task for echoing instructions
_pwd_prompt: mk_dirs

# Make directories based the file paths
mk_dirs:
	@mkdir -p encrypt decrypt ;

# Decrypt files in the encrypt/ directory
decrypt: _pwd_prompt
	@echo "Decrypt the files in a given directory (those with .cast5 extension)."
	@read -p "Source directory: " src && read -p "Password: " password ; \
	mkdir -p decrypt/$${src} && echo "\n" ; \
	for i in `ls encrypt/$${src}/*.cast5` ; do \
		echo "Decrypting $${i}" ; \
		openssl cast5-cbc -d -in $${i} -out decrypt/$${src}/`basename $${i%.*}` -pass pass:$${password}; \
		chmod 600 decrypt/$${src}/`basename $${i%.*}` ; \
	done ; \
	echo "Decrypted files are in decrypt/$${src}"

# Encrypt files in the decrypt/ directory
encrypt: _pwd_prompt
	@echo "Encrypt the files in a directory using a password you specify.  A directory will be created under /encrypt."
	@read -p "Source directory name: " src && read -p "Password: " password && echo "\n"; \
	mkdir -p encrypt/`basename $${src}` ; \
	echo "Encrypting $${src} ==> encrypt/`basename $${src}`" ; \
	for i in `ls $${src}` ; do \
		echo "Encrypting $${src}/$${i}" ; \
		openssl cast5-cbc -e -in $${src}/$${i} -out encrypt/`basename $${src}`/$${i}.cast5 -pass pass:$${password}; \
	done ; \
	echo "Encrypted files are in encrypt/`basename $${src}`"


GIT_REPO:=`git config --get remote.origin.url | sed -e 's/[\/&]/\\&/g'`
GIT_TAG:=`git describe --abbrev=0 --tags`
GIT_BRANCH=`git rev-parse --abbrev-ref HEAD`
GIT_COMMIT_HASH:=`git rev-list --max-count=1 --reverse HEAD`
GIT_COMMIT_MESSAGE:=`git log -1 | tail -1 | sed -e "s/^[ ]*//g"`
BUILD_TIMESTAMP:=`date +"%Y-%m-%d-%H:%M"`
DOCKER_IMAGE:=infradash/dash:$(GIT_TAG)-$(BUILD_LABEL)

LDFLAGS:=\
-X github.com/qorio/omni/version.gitRepo $(GIT_REPO) \
-X github.com/qorio/omni/version.gitTag $(GIT_TAG) \
-X github.com/qorio/omni/version.gitBranch $(GIT_BRANCH) \
-X github.com/qorio/omni/version.gitCommitHash $(GIT_COMMIT_HASH) \
-X github.com/qorio/omni/version.buildTimestamp $(BUILD_TIMESTAMP) \
-X github.com/qorio/omni/version.buildNumber $(BUILD_NUMBER) \


compile:
	echo "Building dash"
	${GODEP} go build -o bin/dash -ldflags "$(LDFLAGS)" main/dash.go

godep:
	echo "Building dash with godep"
	godep go build -o bin/dash -ldflags "$(LDFLAGS)" main/dash.go


# Deploy the compiled binary to another git repo

DEPLOY_REPO_URL:=git@github.com:infradash/public.git
DEPLOY_REPO_BRANCH:=gh-pages
DEPLOY_LOCAL_REPO:=build/deploy
DEPLOY_USER_EMAIL:=deploy@infradash.com
DEPLOY_USER_NAME:=deploy
DEPLOY_DIR:=dash/latest

deploy-git-checkout:
	mkdir -p ./build/deploy
	git clone $(DEPLOY_REPO_URL) $(DEPLOY_LOCAL_REPO)
	cd $(DEPLOY_LOCAL_REPO) && git config --global user.email $(DEPLOY_USER_EMAIL) && git config --global user.name $(DEPLOY_USER_NAME) && git checkout $(DEPLOY_REPO_BRANCH)

deploy-git: deploy-git-checkout
	mkdir -p $(DEPLOY_LOCAL_REPO)/$(DEPLOY_DIR) && cp -r ./bin $(DEPLOY_LOCAL_REPO)/$(DEPLOY_DIR) && echo $(DOCKER_IMAGE) > $(DEPLOY_LOCAL_REPO)/$(DEPLOY_DIR)/DOCKER 
	cd $(DEPLOY_LOCAL_REPO) && git add -v $(DEPLOY_DIR) && git commit -m "Version $(GIT_TAG) Commit $(GIT_COMMIT_HASH) Build $(CIRCLE_BUILD_NUM)" -a && git push


# Simple local example -- assumes localhost zookeeper or SSH tunnel to zookeeper
# Local ssh tunnel:
# ssh -i decrypt/keys/bastion.cer -L 8080:zk1.prod.infradash.com:8080  -L 2181:zk1.prod.infradash.com:2181 ubuntu@bastion.infradash.com
run-local-agent:
	DASH_HOST=`hostname` \
	DASH_DOMAIN="accounts.qor.io" \
	DASH_TAGS="appserver,frontend" \
	DASH_DOCKER_NAME="dash" \
	ZOOKEEPER_HOSTS="localhost:2181" \
	DOCKER_PORT="tcp://192.168.59.103:2376" \
	go run main/dash.go --logtostderr --v=500 --self_register=false \
		--ui_docroot=$(HOME)/go/src/github.com/infradash/dash/www \
		--tlscert=$(HOME)/.boot2docker/certs/boot2docker-vm/cert.pem \
		--tlskey=$(HOME)/.boot2docker/certs/boot2docker-vm/key.pem \
		--tlsca=$(HOME)/.boot2docker/certs/boot2docker-vm/ca.pem \
		--config_source_url="file:///Users/david/go/src/github.com/infradash/dash/example/passport.json" \
	agent

run-local-agent-godep:
	DASH_HOST=`hostname` \
	DASH_DOMAIN="accounts.qor.io" \
	DASH_TAGS="appserver,frontend" \
	DASH_DOCKER_NAME="dash" \
	ZOOKEEPER_HOSTS="localhost:2181" \
	DOCKER_PORT="tcp://192.168.59.103:2376" \
	godep go run main/dash.go --logtostderr --v=500 --self_register=false \
		--tlscert=$(HOME)/.boot2docker/certs/boot2docker-vm/cert.pem \
		--tlskey=$(HOME)/.boot2docker/certs/boot2docker-vm/key.pem \
		--tlsca=$(HOME)/.boot2docker/certs/boot2docker-vm/ca.pem \
		--enable_ui --ui_docroot=/Users/david/go/src/github.com/infradash/dash/docker/dash/www \
		--config_source_url="file:///Users/david/go/src/github.com/infradash/dash/example/passport.json" \
	agent


run-exec-bash-export:
	DASH_DOMAIN="test.infradash.com" \
	ZOOKEEPER_HOSTS="localhost:2181" \
	go run main/dash.go \
		--service=infradash --version=develop \
		--custom_vars=EXEC_TS="{{.StartTimeUnix}},EXEC_DOMAIN={{.Domain}}" \
		--stdout --quote="'" --newline --bash_export \
	exec

run-exec-nginx:
	DASH_DOMAIN="test.infradash.com" \
	ZOOKEEPER_HOSTS="localhost:2181" \
	go run main/dash.go --logtostderr \
		--service=infradash --version=develop \
		--custom_vars=EXEC_TS="{{.StartTimeUnix}},EXEC_DOMAIN={{.Domain}}" \
		--daemon \
	    	--no_source_env=false \
		--config_source_url="http://infradash.github.io/ops-release/dash/profiles/test-nginx.json" \
	exec echo 'now={{.EXEC_TS}} and domain={{.EXEC_DOMAIN}} and db={{.DATABASE_HOST}}'

run-exec-nginx-local-godep:
	DASH_DOMAIN="test.infradash.com" \
	ZOOKEEPER_HOSTS="localhost:2181" \
	godep go run main/dash.go --logtostderr \
		--service=infradash --version=develop \
		--custom_vars=EXEC_TS="{{.StartTimeUnix}},EXEC_DOMAIN={{.Domain}}" \
		--daemon \
	    	--no_source_env=false \
		--config_source_url="file:///Users/david/go/src/github.com/infradash/dash/example/executor.json" \
	exec echo 'now={{.EXEC_TS}} and domain={{.EXEC_DOMAIN}} and db={{.DATABASE_HOST}}'

run-local-exec:
	DASH_DOMAIN="test.infradash.com" \
	ZOOKEEPER_HOSTS="localhost:2181" \
	godep go run main/dash.go --logtostderr \
		--service=infradash --version=develop \
		--custom_vars=EXEC_TS="{{.StartTimeUnix}},EXEC_DOMAIN={{.Domain}}" \
		--daemon=true \
	    	--no_source_env=false \
		--stdout --newline \
		--config_source_url="file:///Users/david/go/src/github.com/infradash/dash/example/executor-local.json" \
	exec echo {{.ENVIRONMENT_NAME}}


# Example: copy env from v0.1.2 to v0.1.3
run-publish-env:
	ZOOKEEPER_HOSTS="localhost:2181" \
	go run main/dash.go --logtostderr -publish -overwrite=false \
		--path=/sandbox.infradash.com/infradash/develop/env \
		--domain=production.infradash.com --service=infradash --version=develop \
	env

# Run a release
run-release:
	ZOOKEEPER_HOSTS="localhost:2181" \
	godep go run main/dash.go --logtostderr --commit \
		--release --commit \
		--domain=test.infradash.com \
		--service=infradash \
		--version=develop \
		--build=4287.133 \
		--image=infradash/infradash:develop-4287.133 \
	registry

run-release-scheduler-trigger:
	ZOOKEEPER_HOSTS="localhost:2181" \
	godep go run main/dash.go --logtostderr --commit \
		--release --commit \
		--image=qorio/passport:v1.0 \
		--scheduler_trigger_path="/test2.qoriolabs.com/passport/release" \
		--scheduler_image_path="/test2.qoriolabs.com/passport" \
	registry

# Run a setlive
run-setlive:
	ZOOKEEPER_HOSTS="localhost:2181" \
	go run main/dash.go --logtostderr \
		--setlive --commit --setlive_nowait \
		--domain=test.infradash.com \
		--service=infradash \
		--version=develop \
		--build=test \
		--image=infradash/infradash:develop-test \
	registry

run-writepath:
	ZOOKEEPER_HOSTS="localhost:2181" \
	go run main/dash.go --logtostderr \
		--commit --writepath=/test.infradash.com/test \
		--writevalue=test123 \
	registry

run-readpath:
	ZOOKEEPER_HOSTS="localhost:2181" \
	go run main/dash.go --logtostderr \
		--read \
		--readpath=/code.infradash.com/infradash \
	registry

run-circleci:
	ZOOKEEPER_HOSTS="localhost:2181" \
	go run main/dash.go --logtostderr \
		--circle_user=qorio \
		--circle_project=passport \
		--circle_token=d84e7b3e53035b9d8fc8a5aadbc2ad4237064e20 \
		--circle_buildnum=213 \
		--build_artifact_dir=/tmp/passport \
	circleci

run-circleci-zk:
	ZOOKEEPER_HOSTS="localhost:2181" \
	go run main/dash.go --logtostderr \
		--circle_auth_zkpath=/code.infradash.com/circleci/passport \
		--circle_buildnum=213 \
		--build_artifact_dir=/tmp/passport \
	circleci

test:
	go test ./... -check.vv -v 
