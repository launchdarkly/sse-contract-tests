FROM golang:1.14 as builder

RUN mkdir /testharness

COPY . /testharness

RUN cd /testharness && CGO_ENABLED=0 go build
# CGO_ENABLED=0 is necessary in order to run the resulting binary on alpine


FROM alpine:3.14

RUN mkdir /testharness
COPY --from=builder /testharness/sse-contract-tests /testharness/sse-contract-tests

ENTRYPOINT [ "/testharness/sse-contract-tests" ]
