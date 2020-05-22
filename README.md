# active

![img](https://github.com/fosskers/active/workflows/Tests/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/fosskers/active)](https://goreportcard.com/report/github.com/fosskers/active)

*Keep your Github Action versions up-to-date. 使用の Github Actions を最新に！*

<!-- markdown-toc start - Don't edit this section. Run M-x markdown-toc-refresh-toc -->
**Table of Contents**

- [Overview](#overview)
- [Installation](#installation)
- [Usage](#usage)
    - [Local Repository](#local-repository)
    - [Batch Updates](#batch-updates)
    - [Automatic PRs](#automatic-prs)
- [Configuration](#configuration)
    - [OAuth](#oauth)

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

Assuming you have a [Golang environment set up](https://golang.org/doc/install):

```bash
go get github.com/fosskers/active
```

# Usage

Once `active` has been ran, it's up to you to review the changes, make a commit,
and open a PR.

## Local Repository

The simplest usage:

```
active --local
```

This will look for workflow files in `./.github/workflows/`.

## Batch Updates

`active` is meant to be configured and ran on multiple projects at once.
Assuming you've configured it (see below), the default usage "just works":

```
> active
Checking the following files:
  --> aura:     ci.yaml
  --> org-mode: ci.yaml
  --> versions: ci.yaml

... etc ...
```

If you trust `active` to do the right thing, you can use `active -y` to
automatically accept all available updates.

## Automatic PRs

With the `--push` flag, `active` will automatically make a commit on a new
branch, push it to Github, and open a PR:

```
> active --push
Checking the following files:
  --> aura:     ci.yaml
  --> org-mode: ci.yaml
  --> versions: ci.yaml

... work ...

Successfully opened a PR for versions! (#35)
Successfully opened a PR for org-mode! (#15)
Successfully opened a PR for aura! (#314)
```

This requires a valid **Personal Access Token** from Github (see below), and
will also create a new Git *remote* called `active` for each project to ensure
that the token can be used properly for pushing.

# Configuration

A config file is not necessary to use `active`, but having one will make your
life easier. By default, `active` looks for its config file at
`$HOME/.config/active.yaml`. Its contents should look like this:

```yaml
projects:
  - /home/you/code/some-project
  - /home/you/code/another-project
  - /home/you/code/third-project

git:
  name:  Your Name      # (Optional) For --push
  email: you@email.com  # (Optional) For --push
  user:  you            # (Optional) For --push
  token: <oauth-token>  # (Optional) For --push, and higher API rate limits in general.
```

`name` and `email` are used for commiting. `user` is used for branch pushing,
and `token` for opening the PR.

If you want to specify an alternate config location, use `--config`.

## OAuth

If you have a Github account, then it's easy to generate a personal access token
for `active`. First visit the [Token
Settings](https://github.com/settings/tokens) on Github. Click **Generate new
token**, and give it `public_repo` permissions:

![](token.png)
