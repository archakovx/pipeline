# Options.
#
PROJECT_NAME:=pipeline
ORG_PATH:=github.com
REPO_PATH:=$(ORG_PATH)/$(PROJECT_NAME)
BINARY_NAME?=pipeline
IMAGE_NAME?=$(REPO_PATH)/$(BINARY_NAME)
VERSION?=dev
GOOS ?=linux
SERVICE?=pipeline-service
API?=v1

build: Dockerfile
	# Building $(PROJECT_NAME)...
	docker build \
		--build-arg "VERSION=$(VERSION)" \
		--build-arg "APP_PKG_NAME=$(REPO_PATH)" \
		--build-arg "GOOS=$(GOOS)" \
		--build-arg "BINARY_NAME=$(BINARY_NAME)" \
		-t $(IMAGE_NAME) .