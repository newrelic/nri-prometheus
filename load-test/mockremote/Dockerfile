FROM golang:alpine as build

WORKDIR /app

COPY . .
RUN GOOS=linux go build -o mockremote .


FROM alpine:latest

RUN mkdir -p /app
COPY --from=build /app/mockremote /app

WORKDIR /app
ENTRYPOINT ["/app/mockremote"]
