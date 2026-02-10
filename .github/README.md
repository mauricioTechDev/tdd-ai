# Release Lifecycle

## How a Release Works

Pushing a version tag triggers the full release pipeline automatically.

```
git tag v0.5.0 && git push origin --tags
```

This kicks off the GitHub Actions workflow in `.github/workflows/release.yaml`, which runs three sequential jobs:

### Job 1: Test

Runs `go vet` and `go test ./... -v` on `ubuntu-latest`. If any test fails, the release is aborted — nothing gets published.

### Job 2: GoReleaser

Cross-compiles the Go binary for 6 platform/architecture combinations:

| OS      | Architectures   | Archive Format |
|---------|-----------------|----------------|
| Linux   | amd64, arm64    | `.tar.gz`      |
| macOS   | amd64, arm64    | `.tar.gz`      |
| Windows | amd64, arm64    | `.zip`         |

The version is injected at build time via `-X github.com/macosta/tdd-ai/cmd.version=<tag>` (stripping the `v` prefix). Archives and a checksum file are uploaded to a GitHub Release created from the tag.

Configuration: `/.goreleaser.yaml`

### Job 3: npm Publish

Patches `npm/package.json` with the version from the git tag, then runs `npm publish`. This makes the release available via:

```bash
npm install -g tdd-ai
npx tdd-ai
```

When a user installs the npm package, the `postinstall` script (`npm/install.js`) downloads the correct binary for their platform from the GitHub Release created in Job 2.

## Version Source of Truth

The **git tag** is the single source of truth. Every system derives its version from it:

| System         | How it gets the version                                      |
|----------------|--------------------------------------------------------------|
| Go binary      | GoReleaser injects via ldflags at build time                 |
| GitHub Release | GoReleaser creates the release from the tag                  |
| npm package    | CI patches `package.json` from the tag before `npm publish`  |
| Local dev      | `Makefile` has `VERSION?=x.y.z` for `make build`            |

## Required Secrets

| Secret      | Where to set it                                                    | Purpose                    |
|-------------|--------------------------------------------------------------------|----------------------------|
| `NPM_TOKEN` | [Repo settings > Secrets > Actions](../../settings/secrets/actions) | Authenticates `npm publish` |

`GITHUB_TOKEN` is provided automatically by GitHub Actions.

## Local Testing

Validate the GoReleaser config without publishing:

```bash
goreleaser release --snapshot --clean
```

Preview what npm would publish:

```bash
cd npm && npm pack --dry-run
```
