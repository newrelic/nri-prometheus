FROM golang:1.15.2-buster

ARG GH_VERSION='1.1.0'
#    
RUN apt-get update \
    && apt-get -y install \
    rpm \
    gnupg2 \
    gpg-agent \
    debsigs \
    unzip \
    zip \
    && go get golang.org/dl/go1.13.15 \
    && /go/bin/go1.13.15 download

RUN wget -q https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_amd64.deb -O /tmp/gh_${GH_VERSION}_linux_amd64.deb \
    && apt-get -y install /tmp/gh_${GH_VERSION}_linux_amd64.deb