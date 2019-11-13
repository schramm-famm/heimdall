FROM golang:latest AS builder

# Copy the code from the host and compile it
WORKDIR $GOPATH/src/heimdall
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . ./
COPY ./tmp /tmp
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o /tmp ./...

FROM scratch
WORKDIR /
COPY --from=builder /tmp/* ./
ENV PRIVATE_KEY="id_rsa"
EXPOSE 8080
ENTRYPOINT ["/app"]
