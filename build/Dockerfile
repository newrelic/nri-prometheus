FROM golang:1.17-bullseye

# Add repo for `gh` tool
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | gpg --dearmor -o /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" > /etc/apt/sources.list.d/github-cli.list

RUN apt update \
    && apt -y install \
    rpm \
    gnupg2 \
    gpg-agent \
    debsigs \
    unzip \
    zip \
    gh
