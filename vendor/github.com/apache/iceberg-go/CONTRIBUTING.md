<!--
 - Licensed to the Apache Software Foundation (ASF) under one or more
 - contributor license agreements.  See the NOTICE file distributed with
 - this work for additional information regarding copyright ownership.
 - The ASF licenses this file to You under the Apache License, Version 2.0
 - (the "License"); you may not use this file except in compliance with
 - the License.  You may obtain a copy of the License at
 -
 -   http://www.apache.org/licenses/LICENSE-2.0
 -
 - Unless required by applicable law or agreed to in writing, software
 - distributed under the License is distributed on an "AS IS" BASIS,
 - WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 - See the License for the specific language governing permissions and
 - limitations under the License.
 -->

# Contributing

## Get in Touch

- [Iceberg community](https://iceberg.apache.org/community/)
- [Iceberg-Go Slack](https://apache-iceberg.slack.com/archives/C05J3MJ42BD)

## Picking Up Issues

Before starting work on an issue:

1. **Check for existing PRs.** Search the [open pull requests](https://github.com/apache/iceberg-go/pulls) to make sure nobody is already working on it.
2. **Claim the issue.** Leave a comment on the issue (e.g., "I'd like to work on this") and wait for a maintainer to acknowledge before writing code.
3. **One at a time for new contributors.** If you haven't had a PR merged into iceberg-go yet, please work on one issue at a time. Get it reviewed, address feedback, get it merged — then pick up the next one. This helps us give your work the attention it deserves and avoids wasted effort from overlapping contributions.

If two PRs land for the same issue, we will generally keep the one from the contributor who claimed it first.

## Submitting a Pull Request

- Reference the issue number in your PR description (e.g., "Fixes #123").
- Keep PRs focused — one issue per PR.
- Run `make test` and `make lint` before pushing. CI runs all of these too, but catching issues locally saves a round-trip. `make lint-install` will install the linter if you don't have it yet.
- All commits must have a `Signed-off-by` line ([DCO](https://developercertificate.org/)).

## Code Review

- Maintainers may request changes. This is normal — it doesn't mean the PR is bad, it means we want to get it right.
- Respond to review comments by pushing new commits (don't force-push over reviewed code).
- If your PR has been waiting for review for more than a few days, ping on [Slack](https://apache-iceberg.slack.com/archives/C05J3MJ42BD).

## Development Setup

```bash
git clone https://github.com/apache/iceberg-go.git
cd iceberg-go
go build ./...
make test # runs all unit tests
make test-race # runs all unit tests with race detector
```

### Integration Tests

Integration tests require Docker and are gated behind a build tag:

```bash
docker compose -f internal/recipe/docker-compose.yml up -d rest object-store create-bucket --wait
go test -tags integration ./...
```
