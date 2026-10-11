# Releases

digest follows [Semantic Versioning](https://semver.org), and the commit messages since the last release decide the next version, following [Conventional Commits](https://www.conventionalcommits.org):

| Commits since the last release | Release |
|---|---|
| `feat!:`, `fix(scope)!:` or a `BREAKING CHANGE:` line in the body | major |
| `feat:` | minor |
| `fix:` or `perf:` | patch |
| only `docs:`, `ci:`, `chore:`, `test:` and the like | none |

Versions start at v0.0.0 under the module path `github.com/achandrapaul/digest`. The Go module proxy still lists v1.1.50 to v1.3.0 under the old `github.com/AnudeepChPaul/digest` path; those are stale and no longer updated.

The last release is the newest `v*` tag by version, read from git after `git fetch --tags`. `mise run release:level` looks at every commit after it. `mise run release:notes` writes the section for `.version`: a version that is already tagged lists the commits from the tag before it up to its own tag and is dated by its tagged commit; an untagged one lists the commits after the newest tag. With no earlier tag the section starts at the first commit.

## Cutting a release

Pushes to `main` never release on their own. To cut a version:

```sh
mise run release:level          # prints the level the commits call for
mise run release:trigger        # patch by default; pass minor or major to choose
```

`release:trigger` branches `release/v<next-version>` off the latest `origin/main` in a temporary worktree, so your own branch and working tree stay untouched. It bumps `.version`, adds the new [CHANGELOG.md](../CHANGELOG.md) section with `go run ./cmd/release notes`, commits both as `release:<patch|minor|major>`, pushes the branch and opens a pull request into `main`. It stops when that branch already exists on origin. The section lists every commit since the newest tag under Changes, as a link to the commit, its title and its description; release, changelog and benchmark commits are left out.

Merging the pull request is what releases: the push to `main` that changes `.version` runs the build workflow. It runs vet and tests and tags `v<version>`; the tag push starts the release workflow below.

## Publishing a release

The release workflow starts on every pushed `v*` tag. To rerun it or publish an older tag, use the Actions tab or `gh workflow run release.yml -f tag=v<version>`; without a tag it takes the latest one. It checks out the tag, publishes the macOS binaries on GitHub with that version's changelog section as release notes, asks the Go module proxy for the tag, and points the Homebrew formula at it. Older tags and releases stay; GitHub marks the newest one as Latest.

## Benchmarks

The release workflow records how fast digest starts, loads notes, redraws, searches and saves, at 1 to 10k notes, in [benchmark.md](benchmark/benchmark.md) as its own `chore: benchmark` commit.

`mise run bench` measures locally without recording, and `mise run bench:full` adds 100k notes.
