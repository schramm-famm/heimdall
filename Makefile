APP_NAME=heimdall
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

rsa: 				## create tmp/ and generate RSA keys
	if [ -d "./tmp" ]; then rm -rf ./tmp; fi
	mkdir tmp
	openssl genrsa -out ./tmp/id_rsa 2048
	openssl rsa -in ./tmp/id_rsa -pubout > ./tmp/id_rsa.pub
	printf '\n\n\n\n\n\n\n' | openssl req -new -x509 -sha256 -key ./tmp/id_rsa \
		-out ./tmp/server.crt -days 3650

build: rsa 			## build the app binaries
	go build -o ./tmp ./...

test: build 		## build and test the module packages
	export KAREN_HOST="localhost" && export PRIVATE_KEY="../tmp/id_rsa" && \
		export SERVER_CERT="../tmp/server.crt" && go test ./...

run: build 			## build and run the app binaries
	export KAREN_HOST="localhost" && export PRIVATE_KEY="tmp/id_rsa" && \
		export SERVER_CERT="tmp/server.crt" && ./tmp/app

docker: rsa 		## build the docker image
	docker build -t $(APP_NAME) .

docker-run: docker 	## start the built docker image in a container
	docker run -d -p 80:80 -p 443:443 --name $(APP_NAME) $(APP_NAME)

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
