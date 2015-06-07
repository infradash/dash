
GIT_TAG:=`git describe --abbrev=0 --tags`
DOCKER_TAG:=$(GIT_TAG)-$(BUILD_LABEL)

image:
	@echo "Building infradash dash image with tag: ${DOCKER_TAG}"
	sudo docker build -t infradash/dash:latest .
	sudo docker tag -f infradash/dash:latest infradash/dash:${DOCKER_TAG}

push: image
	@echo "Pushing image to registry"
	sudo docker push infradash/dash:latest
	sudo docker push infradash/dash:${DOCKER_TAG}
