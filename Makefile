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

help: 				## show options and their descriptions
	@perl -e '$(HELP_FUNC)' $(MAKEFILE_LIST)

all:  				## clean the working environment, build and test the packages, and then build the docker image
all: clean test docker

rsa: 				## create tmp/ and generate RSA keys
	if [ -d "./tmp" ]; then rm -rf ./tmp; fi
	mkdir tmp
	openssl genrsa -out ./tmp/id_rsa 2048
	openssl rsa -in ./tmp/id_rsa -pubout > ./tmp/id_rsa.pub

build: rsa 			## build the app binaries
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o ./tmp ./...

test: build 		## build and test the module packages
	go test ./...

run: build 			## build and run the app binaries
	./tmp/app

docker: rsa 		## build the docker image
	docker build -t $(APP_NAME) .

docker-run: docker 	## start the built docker image in a container
	docker run -d -p 8080:8080 --name $(APP_NAME) $(APP_NAME)

.PHONY: clean
clean: 				## remove tmp/ and old docker images
	rm -rf tmp
	docker rm -f $(APP_NAME)
	docker system prune
ifneq ("$(shell docker images | grep $(APP_NAME) | awk '{ print $$3; }')", "") 
	docker images | grep $(APP_NAME) | awk '{ print $$3; }' | xargs -I {} docker rmi {}
endif
