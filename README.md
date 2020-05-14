# active

![img](https://github.com/fosskers/active/workflows/Tests/badge.svg)

Keep your Github Action versions up-to-date.

### TODO

First step: perform this on a local repo!

- [x] CLI flags
  - [x] Echo some arg
  - [x] Specify a project to check
- [ ] `.yaml` config file parsing
  - [x] List of projects to check
  - [x] Oauth token
  - [ ] CLI flag for changing config location
- [ ] Github interaction
  - [x] Release ver lookup
  - [ ] Apply the change automatically?
  - [ ] Push up a branch?
  - [ ] Open a PR from that branch? (seems possible with the `github` lib!)
- [x] Basic file IO
- [x] Parse a `.yaml` and look for usage of Actions
- [x] Produce a diff of the change
- [x] Flag for automatically applying the diff
- [x] Authenticated / non-authenicated modes (being auth'd increases GH rate limit)
- [x] Good use of concurrency
- [x] Set up CI
- [ ] Localisation
  - [ ] `-j` flag
  - [ ] Config field for Japanese output
  - [ ] README 和訳
- [ ] README
  - [x] CI badge
  - [ ] Usage examples and sample output
  - [ ] Setting up OAuth
  - [ ] Table of Contents
- [ ] Changelog
- [ ] Official release
