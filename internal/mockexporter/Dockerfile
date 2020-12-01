FROM golang:alpine as build

LABEL maintainer="Roberto Santalla <rsantalla@newrelic.com>"

WORKDIR /app

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o mockexporter .


FROM alpine:latest

RUN mkdir -p /app
COPY --from=build /app/mockexporter /app

WORKDIR /app
ENTRYPOINT ["/app/mockexporter"]
