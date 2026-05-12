# Releasing new versions

Stable releases are prepared through the automated `release/next` pull request.

## Stable releases

The release PR workflow keeps a `release/next` PR open against `main` when there
are unreleased changes.

The release PR:

- computes the next SemVer version from Conventional Commit PR titles
- updates `CHANGELOG.md`
- previews the GitHub release notes
- preserves manual release and migration note blocks in the PR body

To publish a stable release, review and merge the `release/next` PR. When that
PR merges, the release-tag workflow creates the matching `vX.Y.Z` tag on the
merged release commit. The tag triggers the Release workflow, which uses
GoReleaser to publish binaries and Docker images, publishes the npm package, and
completes the Linear release for stable tags.

Do not manually create stable release tags unless the automated release PR path
is unavailable.

## Pre-releases

Alpha, beta, and release-candidate artifacts are published from a PR by comment.
This lets maintainers share artifacts with customers before the PR is merged.

Supported commands:

```text
/prerelease alpha
/prerelease beta
/prerelease rc
/prerelease beta --dry-run
/prerelease alpha v1.2.23-alpha.1
```

Rules:

- The commenter must have maintainer permission.
- Forked PRs are rejected for publishing because release credentials would run
  against untrusted code.
- Without an explicit version, the workflow computes the next prerelease tag for
  the selected channel.
- Dry runs validate the release build without creating a tag.
- Published prereleases tag the PR head SHA and then use the normal tag-triggered
  Release workflow.

Installing a prerelease from npm works as:

```bash
# Use a specific release
npx inngest-cli@v1.2.23-beta.1 dev

# Use the latest beta release
npx inngest-cli@beta dev
```

## Testing before merge

Some of this can be tested before the release automation branch is merged, but
not all of it can be tested against production triggers until the workflow files
exist on the default branch.

Local checks:

```bash
go test ./tools/release-notes
ruby -e 'require "yaml"; YAML.load_file(".github/workflows/release-pr.yml")'
ruby -e 'require "yaml"; YAML.load_file(".github/workflows/release-tag.yml")'
ruby -e 'require "yaml"; YAML.load_file(".github/workflows/prerelease.yml")'
```

Release note assembly can be tested locally with fixture PR JSON:

```bash
scripts/release/collect-pr-notes \
  --input /tmp/prs.json \
  --output /tmp/release-pr-notes.json

scripts/release/build-release-notes \
  --notes /tmp/release-pr-notes.json \
  --changelog CHANGELOG.md \
  --tag v1.2.3 \
  --output /tmp/RELEASE_NOTES.md

scripts/release/render-release-pr-body \
  --tag v1.2.3 \
  --compare-url "https://github.com/inngest/inngest/compare/v1.2.2...main" \
  --latest-tag v1.2.2 \
  --preview /tmp/RELEASE_NOTES.md \
  --output /tmp/release-pr-body.md
```

GitHub Actions checks before merge:

- A workflow can be manually dispatched from this branch after the branch is
  pushed if the workflow file exists on that branch.
- The PR comment prerelease workflow cannot be exercised from normal PR comments
  until `.github/workflows/prerelease.yml` exists on the default branch.
- After merge, validate prerelease behavior first with `/prerelease beta
  --dry-run` on a test PR.
- Validate the stable release path with a low-risk release PR before relying on
  it for a customer-facing release.

## GitHub actions

- Stable release PR: [release-pr.yml](/.github/workflows/release-pr.yml)
- Stable release tag gate: [release-tag.yml](/.github/workflows/release-tag.yml)
- Artifact publishing: [release.yml](/.github/workflows/release.yml)
- PR prereleases: [prerelease.yml](/.github/workflows/prerelease.yml)
