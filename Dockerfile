FROM golang:1.10
ADD . /go/src/github.com/furikuri/serverless-to-go
WORKDIR /go/src/github.com/furikuri/serverless-to-go
RUN go get ./
RUN go build

FROM alpine:3.7
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
WORKDIR /root/
COPY --from=0 /go/bin/serverless-to-go .

ENTRYPOINT ["/root/serverless-to-go"]