# active

Keep your Github Action versions up-to-date.

### TODO

First step: perform this on a local repo!

- [ ] CLI flags
  - [x] Echo some arg
  - [x] Specify a project to check
- [ ] Read a file of projects to check
- [ ] Github interaction
  - [x] Release ver lookup
  - [ ] Apply the change automatically?
  - [ ] Push up a branch?
  - [ ] Open a PR from that branch? (seems possible with the `github` lib!)
- [x] Basic file IO
- [ ] Parse a `.yaml` and look for usage of Actions
- [ ] Produce a diff of the change
- [ ] Authenticated / non-authenicated modes (being auth'd increases GH rate limit)
- [x] Set up CI
- [ ] Full README
- [ ] Official release
