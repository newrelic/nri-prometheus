FROM golang:1.11.4-alpine3.8 AS build
RUN apk add --no-cache --update git make openssh-client
# Set up ssh credentials to fetch private dependencies.
ARG SSH_PRIVATE_KEY
RUN mkdir /root/.ssh/
RUN echo "${SSH_PRIVATE_KEY}" | base64 -d > /root/.ssh/id_rsa
RUN chmod 0600 /root/.ssh/id_rsa
RUN ssh-keyscan source.datanerd.us >> /root/.ssh/known_hosts

WORKDIR /go/src/github.com/newrelic/nri-prometheus
COPY Makefile Makefile
RUN make tools
# Trick for reusing the cache in case vendor.json doesn't change.
COPY vendor vendor
RUN make deps
COPY . .
RUN make compile-only
RUN chmod +x bin/nri-prometheus

FROM alpine:latest
RUN apk add --no-cache ca-certificates

USER nobody
COPY --from=build /go/src/github.com/newrelic/nri-prometheus/bin/nri-prometheus /bin/
ENTRYPOINT ["/bin/nri-prometheus"]
