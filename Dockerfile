FROM golang:1.13-alpine3.10 AS build
RUN apk add --no-cache --update git make

WORKDIR /go/src/github.com/newrelic/nri-prometheus
COPY Makefile Makefile
RUN make tools
# Trick for reusing the cache in case vendor.json doesn't change.
COPY go.mod .
RUN make deps
COPY . .
RUN make compile-only
RUN chmod +x bin/nri-prometheus

FROM alpine:latest
RUN apk add --no-cache ca-certificates

# When standalone is set to true nri-prometheus does not require an infrastructure agent to work and send data
ENV STANDALONE=TRUE

USER nobody
COPY --from=build /go/src/github.com/newrelic/nri-prometheus/bin/nri-prometheus /bin/
ENTRYPOINT ["/bin/nri-prometheus"]
