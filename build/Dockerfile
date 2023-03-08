FROM golang:1.19-buster

ARG GH_VERSION='2.23.0'

RUN apt-get update \
    && apt-get -y install \
    rpm \
    gnupg2 \
    gpg-agent \
    debsigs \
    unzip \
    zip

# Since the user does not match the owners of the repo "git rev-parse --is-inside-work-tree" fails and goreleaser does not populate projectName
# https://stackoverflow.com/questions/72978485/git-submodule-update-failed-with-fatal-detected-dubious-ownership-in-repositor
RUN git config --global --add safe.directory '*'
RUN wget -q https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_amd64.deb -O /tmp/gh_${GH_VERSION}_linux_amd64.deb \
    && apt-get -y install /tmp/gh_${GH_VERSION}_linux_amd64.deb
