# Pull Request (PR) Guidelines

## Submitting a Pull Request for Review

### Small, Atomic Changes
Each Pull Request should ideally focus on one single issue or feature. Avoid including unrelated changes. Break down larger changes into smaller, more manageable, and understandable units.

### Be Descriptive
Clearly explain the purpose and context of the Pull Request. Provide a concise summary of the changes made, why they are necessary, and any relevant information that can help reviewers understand the intention behind the code changes.

### Use Conventional Titles
Use a Conventional Commit title for every PR. The PR title drives changelog categorization and release-note eligibility.

Release-facing prefixes:

- `feat:` for user-visible features
- `fix:` for user-visible bug fixes
- `security:` for security fixes
- `perf:` for user-visible performance improvements

Non-release prefixes:

- `cloud:` for cloud-only changes that should not appear in CLI/self-host release notes
- `internal:` for repository maintenance, refactors, tests, tooling, and implementation-only changes
- `noop:` for changes with no release impact, such as metadata-only updates or intentionally empty commits

Other accepted prefixes include `doc:`, `test:`, `ci:`, `refactor:`, `style:`, `chore:`, and `revert:`.

### Write Release Notes at PR Time
Use the `Release note` section for user-facing impact. Use `None` when the PR does not need a release note.

Release notes are required for `feat:`, `fix:`, `security:`, breaking changes, and PRs labeled `release-note-required`.

Use the `Migration note` section for upgrade, deployment, config, schema, compatibility, deprecation, rollout, or rollback notes. Use `None` when no migration note is needed.

### Review Your Changes
Ensure that the code changes have been thoroughly tested and do not introduce new issues.
We recommend opening a [draft pull request](https://help.github.com/en/github/collaborating-with-issues-and-pull-requests/about-pull-requests#draft-pull-requests) to help you review your PR.

### Optimize the Reviewers' Time
Ensure that "All checks have passed", that you've tested and reviewed your PR, and that the description, release note, and migration note are properly filled before assigning reviewers. Ask for reviews on complete PRs, not drafts.

### Respond to Reviews
Address any concerns or questions raised by reviewers and make necessary adjustments to your code as suggested. Keep the Pull Request up to date with the latest changes.
