# Storx V3 Network

[![Go Report Card](https://goreportcard.com/badge/storx/storx)](https://goreportcard.com/report/storx/storx)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://pkg.go.dev/storx/storx)
[![Coverage Status](https://img.shields.io/badge/coverage-master-green.svg)](https://build.dev.storx/job/storx/job/main/cobertura)

<img src="https://github.com/storx/storx/raw/main/resources/logo.png" width="100">

Storx is building a decentralized cloud storage network.
[Check out our white paper for more info!](https://storx/storx.pdf)

----

Storx is an S3-compatible platform and suite of decentralized applications that
allows you to store data in a secure and decentralized manner. Your files are
encrypted, broken into little pieces and stored in a global decentralized
network of computers. Luckily, we also support allowing you (and only you) to
retrieve those files!

## Table of Contents

- [Contributing](#contributing-to-storx)
- [Start using Storx](#start-using-storx)
- [License](#license)
- [Support](#support)

# Contributing to Storx

All of our code for Storx v3 is open source. If anything feels off, or if you feel that 
some functionality is missing, please check out the [contributing page](https://github.com/storx/storx/blob/main/CONTRIBUTING.md). 
There you will find instructions for sharing your feedback, building the tool locally, 
and submitting pull requests to the project.

### A Note about Versioning

While we are practicing [semantic versioning](https://semver.org/) for our client
libraries such as [uplink](https://github.com/storx/uplink), we are *not* practicing
semantic versioning in this repo, as we do not intend for it to be used via
[Go modules](https://blog.golang.org/using-go-modules). We may have
backwards-incompatible changes between minor and patch releases in this repo.

# Start using Storx

Our wiki has [documentation and tutorials](https://github.com/storx/storx/wiki).
Check out these three tutorials:

 * [Using the Storx Test Network](https://github.com/storx/storx/wiki/Test-network)
 * [Using the Uplink CLI](https://github.com/storx/storx/wiki/Uplink-CLI)
 * [Using the S3 Gateway](https://github.com/storx/storx/wiki/S3-Gateway)

# License

This repository is currently licensed with the [AGPLv3](https://www.gnu.org/licenses/agpl-3.0.en.html) license.

For code released under the AGPLv3, we request that contributors sign our
[Contributor License Agreement (CLA)](https://docs.google.com/forms/d/e/1FAIpQLSdVzD5W8rx-J_jLaPuG31nbOzS8yhNIIu4yHvzonji6NeZ4ig/viewform) so that we can relicense the
code under Apache v2, or other licenses in the future.

# Support

If you have any questions or suggestions please reach out to us on
[our community forum](https://forum.storx/) or file a ticket at
https://support.storx/.
