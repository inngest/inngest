# Sandboxes verification entry point

**Status:** Draft for review. This PR contains the spec only; implementation follows separately.

**Design:** [Sandboxes empty state and callout variants](https://www.figma.com/design/OGjjin5JHqKNlr0tiu8inX/Compute?node-id=12961-13162). **Verification screens:** [PR #5296](https://github.com/inngest/inngest/pull/5296). **Product context:** [Compute fraud checks](https://app.notion.com/p/inngest/Compute-Fraud-checks-for-early-access-417b64753bbd81308a49cb6af3ca006c).

## Goal and scope

Help users discover Sandboxes and enter or resume account verification from the dashboard. Build the linked Sandboxes page: introduction, four feature summaries, documentation link, copyable agent prompt, copyable code example, and the verification callout beside the content. Reuse the existing sidebar, company/profile menus, environment selector, search/Command-K, and footer. Stack the callout above the onboarding content on narrow screens without horizontal page scrolling.

This covers the Sandboxes dashboard entry point only. Reuse the verification flow from #5296; do not add another wizard, implement verification services, or add early-access request/approval screens. A future Compute entry point can reuse the callout, but is not included without a separate design/scope decision.

## Display and navigation

| Supplied account state                                            | Entry-point behavior                                                                                                                                                                                       |
| ----------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Eligible admin; verification not started                          | Show the verification explanation and **Start verification**.                                                                                                                                              |
| Eligible admin; verification started but incomplete               | Show saved progress and **Continue verification**. Resume the first incomplete check in the existing flow.                                                                                                 |
| Verification required; current user cannot verify                 | Show **Ask an admin to verify**, the agreed copy **An account admin needs to complete verification.**, and the non-interactive **Unverified** badge. No verification action or per-check status breakdown. |
| Explicitly verified; first-use/empty state                        | Show **Account verified** and the introductory content. No verification action.                                                                                                                            |
| Access allowed without verification being required                | Preserve normal access/onboarding. Do not describe an exempt or existing account as verified unless the supplied state says so.                                                                            |
| Existing sandbox usage and access allowed                         | Keep the existing sandbox list/detail experience.                                                                                                                                                          |
| Loading, failed status request, locked, suspended, or unavailable | Do not show a misleading Start/Continue/success state. Render an appropriate loading, retry, or blocked treatment from supplied state.                                                                     |

Start and Continue navigate in the same tab to the existing verification route, with the originating account, dashboard entry source, and return context for `/env/:envSlug/sandboxes`. Navigation never chooses a company/profile for the user and never grants access. Missing or mismatched account context remains the verification flow's responsibility.

When the user returns, refetch/invalidate account access and sandbox data using existing query conventions. Show the verified callout only after confirmed verification; preserve the selected environment. Do not optimistically unlock access or infer success from a redirect/query parameter.

## Frontend versus backend

| Area                    | Frontend work                                                                                                                | Backend dependency                                                                                                                                 |
| ----------------------- | ---------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| Page and callout        | Match the final layout, tokens, icons, copy, responsive behavior, and accessible states. Reuse suitable existing components. | None for fixture-based UI review.                                                                                                                  |
| Prompt, example, docs   | Copy the exact prompt/code, show copy success/failure, and link to the approved documentation.                               | No new API needed. SDK/example correctness needs technical review.                                                                                 |
| State rendering         | Render the supplied start, partial, verified, admin, and blocked states; expose navigation/retry callbacks.                  | Authoritative eligibility, account access, verification progress, and current-user verification permission.                                        |
| Start/resume navigation | Connect the CTA to #5296 and carry account/environment return context.                                                       | Agreed verification URL/context contract and a real route. #5296 currently provides UI components and a demo, not a mounted production route.      |
| Return and first use    | Refetch on return and select onboarding versus the existing list.                                                            | Persisted verification state, updated entitlements/access, and a reliable usage/first-use signal.                                                  |
| Enforcement             | Render restrictions and avoid active verification actions for non-admins.                                                    | Enforce admin-only verification and access checks for all sandbox/compute operations, including API/CLI callers. UI visibility is not enforcement. |

The frontend can ship a development-only fixture/demo first. Production state wiring is an integration task and cannot be completed from the current enabled/disabled entitlement alone. Backend owners retain rollout, eligibility, persistence, SMS, card verification, and access grants.

## Proposed data contract

Agree on an account-scoped dashboard read contract that supplies:

- Account identity, current user's `canVerify`, access outcome/reason, and verification status, distinguishing required, in-progress, verified, and not-required/exempt.
- Completion flags for the required checks, sufficient to render progress and resume without exposing phone/card details here.
- An approved verification destination and supported account/return-context parameters.
- A reliable first-use/usage indicator with an agreed account/environment scope. An empty date-filtered result is not proof that Sandboxes has never been used.

This is a proposed UI contract, not an existing endpoint/schema. The dashboard should use an authenticated read available to ordinary account members, so they can see the admin-required state.

## Current implementation and gaps

- [`sandboxes/index.tsx`](../src/routes/_authed/env/$envSlug/sandboxes/index.tsx) branches only on `getSandboxesEnabled()`: enabled renders `SandboxesList`; disabled renders the paid-plan upgrade empty state. [`entitlements.ts`](../src/queries/server/entitlements.ts) reads `account.ents.sandboxes.enabled`. Do not turn every disabled/free account into a verification candidate or remove valid upgrade paths; the backend must distinguish the reason.
- [`SandboxesEmptyState.tsx`](../src/components/Sandboxes/SandboxesEmptyState.tsx) and [`sandboxesContent.tsx`](../src/components/Sandboxes/sandboxesContent.tsx) provide the existing introduction/upgrade surface to adapt. [`SandboxesList.tsx`](../src/components/Sandboxes/SandboxesList.tsx) queries a selected time window, so its empty result cannot establish first-ever usage.
- [PR #5283](https://github.com/inngest/inngest/pull/5283) proposes `GET /v2/compute/access` with `allowed`, `code`, `message`, and `verifyUrl`. It checks deploy access and requires `images:write:create`; it is not by itself the member-readable dashboard contract for permissions, saved progress, and sandbox first-use state. Confirm reuse or a separate read with the backend owner.

## Decisions to confirm before implementation

1. **Progress:** Figma shows three segments, while #5296 has two UI steps (phone/card; email recovery is conditional). Confirm what the segments count. Do not hardcode the example 50% or include optional GitHub/LinkedIn checks.
2. **First use:** The annotation says “Verified, w/o deploys” on a Sandboxes page. Confirm whether this means no sandbox history in the selected environment or across the account, and how long the verified callout remains visible.
3. **Copy:** “Track accuracy or relevance across every run” under “Isolated microVMs” appears unrelated, and the code block repeats the prompt-copy header. Confirm those labels, the SDK example, and that “You won’t be charged” refers only to verification, not later usage/billing.
4. **Integration:** Confirm the dashboard status read, final verification URL, and return-context shape with the backend/verification owner.

## Acceptance criteria for the implementation PR

- Review all supplied states with fixtures, including partial progress, non-admins, exempt accounts, failures, verified first-use, and existing usage.
- Start/Continue reach the existing flow with the correct account and return context; no company/profile is selected automatically.
- Copy actions and docs links work; the page is keyboard accessible and responsive, with announced loading/error feedback.
- Returning refetches authoritative state; account switching clears stale account state. Existing authorized sandbox access and legitimate upgrade behavior remain intact.
- Include focused tests for state selection, CTA context, copying, and return refresh. No backend verification or access-enforcement implementation is part of the frontend PR.
