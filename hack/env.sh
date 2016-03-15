#!/bin/bash

PROJECT="github.com/infradash/dash"
echo "Current directory is $(pwd)"

if [ ! -d "$HOME/go/src/$PROJECT" ]; then
    echo "Creating $HOME/go as the root of go development and set up symlinks to point to this directory."
    IFS='/' read -a proj <<< "$PROJECT"
    mkdir -p $HOME/go/src/${proj[0]}/${proj[1]}
    ln -s $(pwd) $HOME/go/src/${proj[0]}/${proj[1]}/${proj[2]}
fi

export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export PATH=$HOME/go/bin:$PATH

go get github.com/docker/docker/pkg/units

# Godep dependency manager
if [[ $(which godep) == "" ]]; then
    echo "Installing godep."
    go get github.com/tools/godep
fi

# Development tools
if [[ "$CIRCLE_ENV" != "" ]]; then
    echo "Setting up development environment."
    # Run go oracle for development golang.org/x/tools/oracle
    if [[ $(which oracle) == "" ]]; then
	echo "Setting up go oracle for source code analysis."
	go get golang.org/x/tools/cmd/oracle
    fi

    if [[ $(which godoc) == "" ]]; then
	echo "Godoc not installed."
    fi
fi


# Docker stuff -- See DockerToolbox
export DOCKER_CERT_PATH=${HOME}/.docker/machine/machines/default
export DOCKER_HOST=tcp://192.168.99.100:2376
export DOCKER_MACHINE_NAME=default
export DOCKER_TLS_VERIFY=1
