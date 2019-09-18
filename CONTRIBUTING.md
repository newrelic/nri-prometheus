# Contributing to Prometheus OpenMetrics Integration

At New Relic we welcome community code contributions to our code, and have
taken effort to make this process easy for both contributors and our development
team.

## How to contribute

- Read this CONTRIBUTING file.
- Read our [Code of Conduct](./CODE_OF_CONDUCT.md).
- Run Tests.
- Submit a PR.
- *Ensure you’ve signed the CLA, otherwise you’ll be asked to do so.*

## How to get help or ask questions

Do you have questions or are you experiencing unexpected behaviors after
modifying this Open Source Software? Please engage with the “Build on New
Relic” space in the [Explorers
Hub](https://discuss.newrelic.com/c/build-on-new-relic/Open-Source-Agents-SDKs),
New Relic’s Forum. Posts are publicly viewable by anyone, please do not include
PII or sensitive information in your forum post.

## Contributor License Agreement ("CLA")

We'd love to get your contributions to improve Prometheus OpenMetrics
Integration! Keep in mind when you submit your pull request, you'll need to
sign the CLA via the click-through using CLA-Assistant. You only have to sign
the CLA one time per project.

To execute our corporate CLA, which is required if your contribution is on
behalf of a company, or if you have any questions, please drop us an email at
open-source@newrelic.com.

## Filing Issues & Bug Reports

We use GitHub issues to track public issues and bugs. If possible, please
provide a link to an example app or gist that reproduces the issue. When filing
an issue, please ensure your description is clear and includes the following
information. Be aware that GitHub issues are publicly viewable by anyone, so
please do not include personal information in your GitHub issue or in any of
your contributions, except as minimally necessary for the purpose of supporting
your issue. New Relic will process any personal data you submit through GitHub
issues in compliance with the New Relic [Privacy
Notice](https://newrelic.com/termsandconditions/privacy).

- Project version (ex: 0.4.0)
- Custom configurations (ex: flag=true)
- Any modifications made to the project.


### A note about vulnerabilities

New Relic is committed to the privacy and security of our customers and their
data. We believe that providing coordinated disclosure by security researchers
and engaging with the security community are important means to achieve our
security goals.  If you believe you have found a security vulnerability in this
project or any of New Relic's products or websites, we welcome and greatly
appreciate you reporting it to New Relic through
[HackerOne](https://hackerone.com/newrelic).

## Setting up your environment

This Open Source Software can be used in a large number of environments, all of
which have their own quirks and best practices. As such, while we are happy to
provide documentation and assistance for unmodified Open Source Software, we
cannot provide support for your specific environment or your modifications to
the code.

## PR Guidelines

Failing to comply with the following guidelines may result in your PR being
rejected or not reviewed by the maintainers until the issues are fixed:

- CI job runs and passes.
- Adheres to the spirit of our various styleguides.
- Has thorough unit test coverage.
- Appropriate documentation is included.
- PR title summarizes the change.
- PR description includes:
  - A detailed summary of what changed.
  - The motivation for the change.
  - A link to each issue that is closed by the PR (e.g. Closes #123).

Keep in mind that these are just guidelines and may not apply in every case.

## Coding Style Guidelines

This project follows a coding style enforced using linters developed by
the Golang community.

### Running Coding Style Validation

Validating the code to see if it conforms to the project style is simple. Just
invoke:

```bash
$ make validate
```

## Testing Guidelines

This project includes a suite of unit tests with each package which should be
used to verify your changes don't break existing functionality.

### Running Tests

Running the test suite is simple. Just invoke:

```bash
$ make test
```

### Writing Tests

For most contributions it is strongly recommended to add additional tests which
exercise your changes.

This helps us efficiently incorporate your changes into our mainline codebase
and provides a safeguard that your change won't be broken by future
development.

There are some rare cases where code changes do not result in changed
functionality (e.g. a performance optimization) and new tests are not required.
In general, including tests with your pull request dramatically increase the
chances it will be accepted.

## License

By contributing to Prometheus OpenMetrics Integration, you agree that your
contributions will be licensed under the [LICENSE](./LICENSE) file in the root
directory of this source tree.
