# active

Keep your Github Action versions up-to-date.

### TODO

First step: perform this on a local repo!

- [x] CLI flags
  - [x] Echo some arg
  - [x] Specify a project to check
- [ ] Read a file of projects to check
- [ ] Github interaction
  - [x] Release ver lookup
  - [ ] Apply the change automatically?
  - [ ] Push up a branch?
  - [ ] Open a PR from that branch? (seems possible with the `github` lib!)
- [x] Basic file IO
- [x] Parse a `.yaml` and look for usage of Actions
- [ ] Produce a diff of the change
- [ ] Flag for automatically applying the diff
- [ ] Authenticated / non-authenicated modes (being auth'd increases GH rate limit)
- [ ] Good use of concurrency
- [x] Set up CI
- [ ] Full README
- [ ] Official release
