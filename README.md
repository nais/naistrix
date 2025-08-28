# Naistrix

[![Go Reference](https://pkg.go.dev/badge/github.com/nais/naistrix.svg)](https://pkg.go.dev/github.com/nais/naistrix)

Naistrix is an opinionated wrapper around the [Cobra library](https://github.com/spf13/cobra), used when building CLI applications in Go.

## Installation

Use `go get` to fetch the latest version of Naistrix:

```bash
go get github.com/nais/naistrix@latest
```

## Usage

Please have a look at the [examples](examples/) directory for usage examples.

## Local development

### Clone the repository:

```bash
git clone git@github.com:nais/naistrix.git
cd naistrix
```

### Install tools

Naistrix uses [mise](https://mise.jdx.dev/) to handle dependencies and tasks. After installing mise run the following command to install tools:

```bash
mise install
```

### Make changes

First, create a new branch for your changes:

```bash
git checkout -b my-feature-branch
```

Make the necessary changes to the codebase. When committing your changes, make sure to follow the [conventional commit specification](https://www.conventionalcommits.org/en/v1.0.0/). This helps maintain a clean and understandable project history along with a nice automated Changelog.

The repository contains a [`commit-msg`](script/semantic-commit-hook.sh) git hook that might be helpful. To enable this you can do the following from the project root:

```bash
ln -s ../../script/semantic-commit-hook.sh .git/hooks/commit-msg
```

There are also several mise tasks that you can use to validate your changes. See all available tasks by running:

```bash
mise run
```

### Create pull request

Once you have made your changes and committed them, push your branch and make a pull request against the `main` branch.

## License

MIT, see [LICENSE.txt](LICENSE.txt) for details.