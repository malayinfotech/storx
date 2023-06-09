# Contributing to Storx

[license]: https://github.com/storx/storx/tree/main/LICENSE
[cla]: https://docs.google.com/forms/d/e/1FAIpQLSdVzD5W8rx-J_jLaPuG31nbOzS8yhNIIu4yHvzonji6NeZ4ig/viewform
[contributing]: https://github.com/storx/storx/tree/main/CONTRIBUTING.md
[white paper]: https://storx/whitepaper
[code of conduct]: https://github.com/storx/storx/tree/main/CODE_OF_CONDUCT.md
[writing tests]: https://github.com/storx/storx/wiki/Testing
[storx-sim]: https://github.com/storx/storx/wiki/Test-network

Hi! Thanks for your interest in contributing to the Storx Network!

Contributions to this project are released under the [AGPLv3 License][license].
For code released under the AGPLv3, we request that contributors sign our
[Contributor License Agreement (CLA)][cla] so that we can re-license parts of the code 
under a less restrictive license, like Apache v2, if that would help the adoption of Storx in the future.

## Topics
* [Reporting Security Issues](#reporting-security-issues)
* [Issue tracking and roadmap](#issue-tracking-and-roadmap)
* [Quick Contribution Tips and Guidelines](#quick-contribution-tips-and-guidelines)
* [Testing](#testing)
* [Resources](#resources)

## Reporting security issues
If you believe you've found a security vulnerability, please send your report to security-reports@storx.

We greatly value security, and we may publicly thank you for your report, although we keep your name confidential if you request it.

## Issue tracking and roadmap

See the breakdown of what we're building by checking out the following resources:

* [White paper][]

## Quick contribution tips and guidelines

### Pull requests are always welcome

To encourage active collaboration, pull requests are strongly encouraged, not just bug reports.

We accept pull requests for bug fixes and features where we've discussed the approach in an issue and given the go-ahead for a community member to work on it.

"Bug reports" may also be sent in the form of a pull request containing a failing test. We'd also love to hear about ideas for new features as issues.

Please do:

* Check existing issues to verify that the bug or feature request has not already been submitted.
* Open an issue if things aren't working as expected.
* Open an issue to propose a significant change.
* open an issue to propose a feature
* Open a pull request to fix a bug.
* Open a pull request for an issue with the [`Help Wanted`](https://github.com/storx/storx/labels/Help%20Wanted) or [`Good first issue`](https://github.com/storx/storx/labels/Good%20First%20Issue) label and leave a comment claiming it.

Please avoid:

* Opening pull requests for issues marked `Need Design`, `Need Investigation`, `Waiting For Feedback` or `Blocked`.
* opening pull requests that are not related to any open issue unless they are bug reports in the form pull requests containing failing tests.

Please note that this project adheres to a [Contributor Code of Conduct][code of conduct]. By participating in this project you agree to abide by its terms.

### Install required packages

To get started running Storx locally, download and install the latest release of Go (at least Go 1.16) at [golang.org](https://golang.org).

You will also need [Git](https://git-scm.com/). (`brew install git`, `apt-get install git`, etc).
If you're building on Windows, you also need to install and have [gcc](https://gcc.gnu.org/install/binaries.html) setup correctly.

We support Linux, Mac, and Windows operating systems. Other operating systems supported by Go should also be able to run Storx.

### Download and compile Storx

> **Aside about GOPATH**: Go 1.11 supports a new feature called Go modules,
> and Storx has adopted Go module support. If you've used previous Go versions,
> Go modules no longer require a GOPATH environment variable. Go by default
> falls back to the old behavior if you check out code inside of the directory
> referenced by your GOPATH variable, so make sure to use another directory,
> `unset GOPATH` entirely, or set `GO111MODULE=on` before continuing with these
> instructions.

First, fork our repo and clone your copy of our repository.

```bash
git clone git@github.com:<your-username>/storx storx
cd storx
```

Then, let's install Storx.

```bash
go install -v ./cmd/...
```

### Make changes and test

Make the changes you want to see! Once you're done, you can run all the unit tests:

```bash
go test -v ./...
```

You can also execute only a single test package if you like. For example:
`go test ./pkg/identity`. Add `-v` for more information about the executed unit
tests.

See our guide for [writing tests][] and setting a local instance of the V3 components using the [storx-sim][].

### Commit Messages

Our guide on good commits can be found at https://github.com/storx/storx/wiki/Git.

### Push up a pull request

Use Git to push your changes to your fork:

```bash
git commit -a -m 'my changes!'
git push origin main
```

Use GitHub to open a pull request!

## Testing
- See our guide for writing tests here:
  - [Testing][writing tests]
- You can spin up a local instance of the V3 Network components (Uplink, Satellite, Storage Nodes), see:
  - [Learn how to run the storx-sim test network][storx-sim]

## Resources

- [Storx White paper v3][white paper]
- [Wiki Page](https://github.com/storx/storx/wiki)
- [How to Contribute to Open Source](https://opensource.guide/how-to-contribute/)
- [Using Pull Requests](https://help.github.com/articles/about-pull-requests/)
