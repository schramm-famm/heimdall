# Heimdall
Heimdall is the single point of entry to Riht. It creates and distributes JSON
Web Tokens (JWTs) to users to be used with their requests. Heimdall will check
requests to protected routes for an associated token before fowarding it to the
proper service.

## APIS
### POST /heimdall/api/token
This route handles the creation of the tokens. The request must have the header
`Content-Type: application/json` and a body containing the `email` and
`password` of the user.

### GET/POST/PUT /
This route is hit for any request to the system that isn't __POST
/heimdall/api/token__. It will check the `Authorization` header of the request
for the value `Bearer <token>` where `<token>` is a valid and non-expired token
that was generated by Heimdall. If it is valid, the request is fowarded to the
internal load balancer to be sent to the proper service. If not, the user will
be denied access.

## Developer Documentation
### Dependencies
Heimdall must be tested and ran in the virtual machine environment provided in
the [VM repo](https://github.com/schramm-famm/vm). This environment comes with
docker, golang, kubernetes, and many other dependencies pre-installed.

### Project
The project directory is laid out as following:
```
heimdall/
|-- Dockerfile
|-- go.mod
|-- go.sum
|-- Makefile
|-- README.md
|-- app/
|   |-- server.go
|-- handlers/
    |-- handlers.go
    |-- handlers_test.go
    |-- models.go
```

#### Dockerfile
The Dockerfile dictates how to build the docker container. This Dockerfile has
two stages. The first stage uses the golang container base and downloads the
module dependencies and builds the go app binary. The second stage uses the most
barebones container base and moves only the important binaries into the
container.

#### go.mod and go.sum
These files handle the dependencies of the project. `go.mod` was created by
running `go mod init` in the root of the directory and `go.sum` was created
during the first execution of `go run ./...` or `go build ./...`.

#### Makefile
The Makefile defines a set of tasks to be executed. When the command `make` or
`make help` is executed in the root directory, the usage of the command is
outputted. This file can be used to build the app binaries, run tests, build the
docker images, and more.

#### README.md
The README.md is a markdown file that provides documentation for the repository.

#### server.go
This file is the entrypoint of Heimdall. The `app` directory is the `main`
package so a binary for it is created when `go build ./...` is executed. When
executed, this file will start the HTTP server that listens on the 8080 port for
requests to the two routes outlined in [APIs](#apis).

#### handlers.go
This file contains the definitions of the route handlers for the server.

#### handlers_test.go
This is the test file for `handlers.go`. It contains unit tests for the handlers
defined in `handlers.go`.

#### models.go
This files contains the definitions of the `User` and `TokenClaims` structs.

### Developer Process
When implementing features, tests should be made in parallel with the
implementation process. Each new feature must have corresponding tests to
validate the new functionality.

To run Heimdall within the developer environment, `make run` can be executed to
build and run the binaries quickly. To create the docker image and start the
detached container, execute `make docker-run`.
