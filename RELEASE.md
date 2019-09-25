# Release proccess

The releases are triggered by creating a new tag on master.
[TravisCI](https://travis-ci.org/) will noticed the tag and will proceed to run the job.
On a successful build the job will run [GoReleaser](https://goreleaser.com). 
This will publish the build artifacts, push the docker image, and to upload the Kubernetes manifest template to download.newrelic.com.
The build will also fill in the changelog with commits that aren't filtered by the GoReleaser config found [here](.goreleaser.yml).
To documentation to change what appears in the changelog can be found [here](https://goreleaser.com/customization/#Release).
 
To create a new release you need to tag the master with the new release version.

## Version naming scheme

All release follow the [semantic versioning](https://semver.org/) scheme.

For the release in this project we tag master with the release version, with a prefix `v` before the version number.
E.g. so release `1.2.3` would mean you tag master with the tag `v1.2.3` 

## Tagging via the command line

To tag via the cli you need to have [Git](https://git-scm.com/) installed.
From a terminal run the command:

```shell script
$ git tag -a vX.Y.Z -m 'New release with cool feature'
```

To kick off the release you then need to push the tag to github.
This is done by running the following command:

```shell script
$ git push origin vX.Y.Z
```
Once the this is done it then triggers a release on [TravisCI](https://travis-ci.org/).
You can see the progress of the deployment [here](https://travis-ci.org/newrelic/nri-prometheus/builds).

## Tagging on Github

1. Click on [Releases](releases).
2. Click *Draft a new release*.
3. Type a version number (with the prefix `v`), e.g. `vX.Y.Z`
4. Set the release title, e.g. 'New release with cool feature'
5. Then hit *Publish release*

Once the this is done it then triggers a release on [TravisCI](https://travis-ci.org/).
You can see the progress of the deployment [here](https://travis-ci.org/newrelic/nri-prometheus/builds).
