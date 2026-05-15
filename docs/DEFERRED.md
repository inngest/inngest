# Deferred Work

## Release-owner review for migration notes

Context: `docs/plans/004-automated-release-cycle.org` proposes collecting
release notes and migration notes from PRs, then combining them into the release
PR preview and GitHub release body.

Deferred idea: add a separate release-owner review path for PRs with release
risk, distinct from normal CODEOWNERS review.

Possible future shape:

- Keep `CODEOWNERS` responsible for code/domain review.
- Add a small GitHub team such as `@inngest/release-owners` or
  `@inngest/platform-release`.
- Request that team only when a PR has non-empty migration notes, a breaking
  change marker, or a `release-review-required` label.
- Use automation/checks to make that review required only for those triggered
  PRs.

Reason for deferral: the first release automation pass should focus on reliable
note collection, `release/next` PR generation, tag creation, and release-note
assembly without adding new reviewer gates.
