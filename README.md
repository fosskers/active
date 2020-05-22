# active

![img](https://github.com/fosskers/active/workflows/Tests/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/fosskers/active)](https://goreportcard.com/report/github.com/fosskers/active)

*Keep your Github Action versions up-to-date. 使用の Github Actions を最新に！*

<!-- markdown-toc start - Don't edit this section. Run M-x markdown-toc-refresh-toc -->
**Table of Contents**

- [Overview](#overview)
- [Installation](#installation)
    - [Arch Linux](#arch-linux)
    - [Via `go get`](#via-go-get)
    - [From Source](#from-source)
- [Usage](#usage)
    - [Local Repository](#local-repository)
- [Configuration](#configuration)
    - [OAuth](#oauth)
- [TODOs](#todos)

<!-- markdown-toc end -->

# Overview

If you use Github CI, you'll recognize fields like this in your Workflow Config:

```yaml
- name: Check out my code
  uses: actions/checkout@v2
```

But wait a minute, these Actions [receive
updates](https://github.com/actions/checkout/releases), so our configs can fall
behind!

`active` scans your projects and queries Github for the latest Action releases,
and updates your configs for you:

```
> active --local
Checking the following files:
  --> .: go.yml

Updates available for .: go.yml:
  actions/setup-go 2 --> 2.0.3
  actions/checkout 2 --> 2.1.0
Would you like to apply them? [Y/n] Y
Updated.
```

# Installation

### Arch Linux

With an AUR-compatible package manager like
[Aura](https://aur.archlinux.org/packages/aura-bin/) installed:

```bash
sudo aura -Aa active
```

### Via `go get`

Hello.

### From Source

Assuming you have a [Golang environment set up](https://golang.org/doc/install):

```bash
git clone https://github.com/fosskers/active.git
cd active
go install
```

# Usage

Once `active` has been ran, it's up to you to review the changes, make a commit,
and open a PR.

### Local Repository

The simplest usage:

```
active --local
```

This will look for workflow files in `./.github/workflows/`.

### Batch Updates

`active` is meant to be configured and ran on multiple projects at once.
Assuming you've configured it (see below), the default usage "just works":

```
> active
Checking the following files:
  --> active:   go.yml
  --> aura:     ci.yaml
  --> org-mode: ci.yaml
  --> versions: ci.yaml

... etc ...
```

If you trust `active` to do the right thing, you can use `active -y` to
automatically accept all available updates.

# Configuration

A config file is not necessary to use `active`, but having one will make your
life easier. By default, `active` looks for its config file at
`$HOME/.config/active.yaml`. Its contents should look like this:

```yaml
projects:
  - /home/colin/code/go/active
  - /home/colin/code/haskell/aura
  - /home/colin/code/haskell/org-mode
  - /home/colin/code/haskell/versions

token: ... OAuth token here ...  # Optional.
```

If you want to specify an alternate config location, use `--config`.

### OAuth

If you have a Github account, then it's easy to generate a personal access token
for `active`.

# TODOs

First step: perform this on a local repo!

- [x] CLI flags
  - [x] Echo some arg
  - [x] Specify a project to check
- [x] `.yaml` config file parsing
  - [x] List of projects to check
  - [x] Oauth token
  - [x] CLI flag for changing config location
- [x] Github interaction
  - [x] Release ver lookup
  - [x] `--push` flag to make a commit and push a branch.
  - [x] Push up a branch?
  - [x] Open a PR from that branch? (seems possible with the `github` lib!)
- [x] Basic file IO
- [x] Parse a `.yaml` and look for usage of Actions
- [x] Produce a diff of the change
- [x] Flag for automatically applying the diff
- [x] Authenticated / non-authenicated modes (being auth'd increases GH rate limit)
- [x] Good use of concurrency
- [x] Set up CI
- [ ] README
  - [x] CI badge
  - [x] Usage examples and sample output
  - [ ] Setting up OAuth
  - [x] Table of Contents
  - [ ] 和訳
- [x] Changelog
- [ ] Official release
  - [ ] Github release
  - [ ] Gopkgs release?
  - [ ] AUR package
  - [ ] Brew package
