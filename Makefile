# Ref: https://about.gitlab.com/blog/2017/11/27/go-tools-and-gitlab-how-to-do-continuous-integration-like-a-boss/

# Reminder "@" means don't echo

SHELL := bash

.PHONY: build
.SHELLFLAGS := -eu -o pipefail -c

build:
	@docker build -f ./build/containers/Dockerfile . -t 536172818835.dkr.ecr.us-east-2.amazonaws.com/kv-test:latest --platform linux/amd64

build_publish: build
	@docker push 536172818835.dkr.ecr.us-east-2.amazonaws.com/kv-test:latest
