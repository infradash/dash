#!/bin/bash

# Git commit hash / message
GIT_REPO=$(git config --get remote.origin.url | sed -e 's/[\/&]/\\&/g')
GIT_TAG=$(git describe --abbrev=0 --tags)
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
GIT_COMMIT_HASH=$(git rev-list --max-count=1 --reverse HEAD)
GIT_COMMIT_MESSAGE=$(git log -1 | tail -1 | sed -e "s/^[ ]*//g")
BUILD_TIMESTAMP=$(date +"%Y-%m-%d-%H:%M")

echo "Remote=$GIT_REPO Tag=$GIT_TAG Branch=$GIT_BRANCH Commit=$GIT_COMMIT_HASH Message=$GIT_COMMIT_MESSAGE On=$BUILD_TIMESTAMP"

if [[ "$@" == "" ]]; then
    echo "No file to process."
    exit
fi

sed -ri "s/@@GIT_REPO@@/${GIT_REPO}/g" $@
sed -ri "s/@@GIT_TAG@@/${GIT_TAG}/g" $@
sed -ri "s/@@GIT_BRANCH@@/${GIT_BRANCH}/g" $@
sed -ri "s/@@GIT_COMMIT_HASH@@/${GIT_COMMIT_HASH}/g" $@
sed -ri "s/@@GIT_COMMIT_MESSAGE@@/${GIT_COMMIT_MESSAGE}/g" $@
sed -ri "s/@@BUILD_TIMESTAMP@@/${BUILD_TIMESTAMP}/g" $@
