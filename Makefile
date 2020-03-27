APP_NAME=heimdall
REGISTRY?=343660461351.dkr.ecr.us-east-2.amazonaws.com
TAG?=latest
KAREN_HOST?=localhost:8080
PATCHES_HOST?=localhost:8081
HELP_FUNC = \
    %help; \
    while(<>) { \
        if(/^([a-z0-9_-]+):.*\#\#(?:@(\w+))?\s(.*)$$/) { \
            push(@{$$help{$$2 // 'targets'}}, [$$1, $$3]); \
        } \
    }; \
    print "usage: make [target]\n\n"; \
    for ( sort keys %help ) { \
        print "$$_:\n"; \
        printf("  %-20s %s\n", $$_->[0], $$_->[1]) for @{$$help{$$_}}; \
        print "\n"; \
    }

.PHONY: help
help: 				## show options and their descriptions
	@perl -e '$(HELP_FUNC)' $(MAKEFILE_LIST)

all:  				## clean the working environment, build and test the packages, and then build the docker image
all: clean test docker

tmp: 				## create ./tmp
	if [ -d "./tmp" ]; then rm -rf ./tmp; fi
	mkdir tmp

rsa: tmp			## generate RSA keys
	openssl genrsa -out ./tmp/id_rsa 2048
	openssl rsa -in ./tmp/id_rsa -pubout > ./tmp/id_rsa.pub

cert: rsa
	printf 'CA\nOntario\nOttawa\nschramm-famm\n\n\n\n' | openssl req -new -x509 -sha256 -key ./tmp/id_rsa \
		-out ./tmp/server.crt -days 3650

build: rsa			## build the app binaries
	go build -o ./tmp ./...

test: build 		## build and test the module packages
	export PRIVATE_KEY="../tmp/id_rsa" && go test ./...

run: build 			## build and run the app binaries
	export KAREN_HOST=$(KAREN_HOST) && export PATCHES_HOST=$(PATCHES_HOST) \
		&& export PRIVATE_KEY="tmp/id_rsa" && ./tmp/app

docker: cert		## build the docker image
	docker build -t $(REGISTRY)/$(APP_NAME):$(TAG) .

docker-run: docker 	## start the built docker image in a container
	docker run -d -p 80:80 -p 8080:8080 -e KAREN_HOST=$(KAREN_HOST) \
		-e PATCHES_HOST=$(PATCHES_HOST) -e PRIVATE_KEY="id_rsa" \
		--name $(APP_NAME) $(REGISTRY)/$(APP_NAME):$(TAG)

.PHONY: clean
clean: 				## remove tmp/ and old docker images
	rm -rf tmp
ifneq ("$(shell docker container list -a | grep heimdall)", "")
	docker rm -f $(APP_NAME)
endif
	docker system prune
ifneq ("$(shell docker images | grep $(APP_NAME) | awk '{ print $$3; }')", "")
	docker images | grep $(APP_NAME) | awk '{ print $$3; }' | xargs -I {} docker rmi -f {}
endif
