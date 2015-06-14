##############################################################################
#
# Targets for generate build info
#

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
