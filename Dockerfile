# This Dockerfile is for local testing. It is not the Docker image that will be published in
# releases; that is built by Goreleaser.

FROM golang:1.16 as builder

RUN mkdir /testharness

COPY . /testharness

RUN cd /testharness && CGO_ENABLED=0 go build
# CGO_ENABLED=0 is necessary in order to run the resulting binary on alpine


FROM alpine:3.14

RUN mkdir /testharness
COPY --from=builder /testharness/sse-contract-tests /testharness

EXPOSE 8111
ENTRYPOINT [ "/testharness/sse-contract-tests" ]
