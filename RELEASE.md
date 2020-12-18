# Release proccess

Releases are triggered by creating a new **pre-release** on github.
On a successful build the job will run [GoReleaser](https://goreleaser.com).
This will generate artifacts, docker images, and kubernetes manifest which will be uploaded the same step.
After verifying everything is correct, the pre-release (already containing the artifacts) can be promoted to a release.
Pre-release to release promotion will not trigger any additional job, as everything is done in the pre-release step.

The `Update Helm Chart POMI version` (`helm.yml`) GitHub Action will be triggered creating a new PR on https://github.com/newrelic/helm-charts/ with the version specified in the tag. After POMI is released this PR should be merged and released.
 
To create a new release you need to tag the main branch with the new release version.

## Version naming scheme

All release follow the [semantic versioning](https://semver.org/) scheme.

For the release in this project we tag main with the release version, with a prefix `v` before the version number.
E.g. so release `1.2.3` would mean you tag main with the tag `v1.2.3` 

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

Pipeline progress can be viewed in the "Actions" tab in Github.
