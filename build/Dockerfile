FROM golang:1.17-buster

ARG GH_VERSION='1.4.0'

RUN apt-get update \
    && apt-get -y install \
    rpm \
    gnupg2 \
    gpg-agent \
    debsigs \
    unzip \
    zip

RUN wget -q https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_amd64.deb -O /tmp/gh_${GH_VERSION}_linux_amd64.deb \
    && apt-get -y install /tmp/gh_${GH_VERSION}_linux_amd64.deb
