# Release proccess

First, run `release.sh` script with the new version that will be released. This should update the CHANGELOG, the version stored in the code and the manifest that gets uploaded to the download site.

Update the `CHANGELOG.md` file in this repository and create a [GH release](https://github.com/newrelic/nri-prometheus/releases/new).
Use the version of the release as input for the CI Job later on.

Create a Github release for the version that is about to be released. The title of the release should follow the template: `v0.0.0`. The changelog of each version should be part of the release description.

Trigger the [CI Release Job](#pending-link) to build and push the docker image, and to upload the Kubernetes
manifest template to download.newrelic.com.
