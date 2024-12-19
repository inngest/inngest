# Releasing new versions

To create a new stable release for the CLI, create a new git tag on the `main` branch with a `v` prefix using SemVer. For example:

```
git tag v1.2.22
git push --tags
```

When this tag is pushed, the "Release" GitHub action will be run which uses [GoReleaser](https://goreleaser.com/) to create binaries and containers to publish to [npm](https://www.npmjs.com/package/inngest-cli) and [Docker hub](https://hub.docker.com/r/inngest/inngest).

## Pre-releases (beta, alpha)

To create a pre-release, create a tag with a `-<type>*` suffix on the end of the version. It's recommended to add a `.<number>` after this to be able to create multiple betas of a particular release version. The `<type>` wil be used as the tag on npm to be able to download.

```
git tag v1.2.23-beta.1
git tag v1.2.23-alpha.3
git tag v1.2.23-rc.1
```

Installing a given release from npm then works as:

```
# Use a specific release
npx inngest-cli@v1.2.23-beta.1 dev
# Use the latest beta release
npx inngest-cli@beta dev
```

## GitHub action

The entire release action is configured in [release.yml](/.github/workflows/release.yml).
