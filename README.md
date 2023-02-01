changed-objects
===============

Get changed objects within the commit histories by comparing between two points.

```console
$ changed-objects --help
Usage:
  changed-objects [OPTIONS]

Application Options:
  -v, --version                       Show version
  -b, --default-branch=               Specify default branch name (default: main)
  -m, --merge-base=                   Specify a Git reference as good common ancestors as possible for a merge
      --type=[added|modified|deleted] Specify the type of changed objects
      --ignore=                       Specify a pattern to skip when showing changed objects
      --group-by=                     Specify a pattern to make into one group when showing changed objects

Help Options:
  -h, --help                          Show this help message
```

## Usage

```console
$ changed-objects
{"files":[{"name":"ditto.go","path":"ditto/ditto.go","type":"deleted","parent_dir":{"path":"ditto","exist":false}},{"name":"go.mod","path":"go.mod","type":"modified","parent_dir":{"path":".","exist":true}},{"name":"go.sum","path":"go.sum","type":"modified","parent_dir":{"path":".","exist":true}},{"name":"detect.go","path":"internal/detect/detect.go","type":"added","parent_dir":{"path":"internal/detect","exist":true}},{"name":"file.go","path":"internal/detect/file.go","type":"added","parent_dir":{"path":"internal/detect","exist":true}},{"name":"git.go","path":"internal/git/git.go","type":"added","parent_dir":{"path":"internal/git","exist":true}},{"name":"main.go","path":"main.go","type":"modified","parent_dir":{"path":".","exist":true}}],"dirs":[{"path":"ditto","files":[{"name":"ditto.go","path":"ditto/ditto.go","type":"deleted","parent_dir":{"path":"ditto","exist":false}}]},{"path":".","files":[{"name":"go.mod","path":"go.mod","type":"modified","parent_dir":{"path":".","exist":true}},{"name":"go.sum","path":"go.sum","type":"modified","parent_dir":{"path":".","exist":true}},{"name":"main.go","path":"main.go","type":"modified","parent_dir":{"path":".","exist":true}}]},{"path":"internal/detect","files":[{"name":"detect.go","path":"internal/detect/detect.go","type":"added","parent_dir":{"path":"internal/detect","exist":true}},{"name":"file.go","path":"internal/detect/file.go","type":"added","parent_dir":{"path":"internal/detect","exist":true}}]},{"path":"internal/git","files":[{"name":"git.go","path":"internal/git/git.go","type":"added","parent_dir":{"path":"internal/git","exist":true}}]}]}
```

## Installation

Download the binary from [GitHub Releases][release] and drop it in your `$PATH`.

- [Darwin / Mac][release]
- [Linux][release]

## License

[MIT][license]

[release]: https://github.com/b4b4r07/changed-objects/releases/latest
[license]: https://b4b4r07.mit-license.org
