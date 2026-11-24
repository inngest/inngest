# Compute verification integration

`/verify?account_id=<account UUID>` mounts the existing focused verification
shell under normal authentication and the browser URQL provider, without the
sidebar. Optional `source=cli|dashboard|api` controls completion copy; absent or
unknown source means API. Dashboard entry supplies a local sandbox `return_to`
path. Invalid links show an error; mismatched accounts must be selected by the
user with the existing menus. Nothing automatically switches accounts.

## Backend dependency

Requires the backend stack through inngest/monorepo#9657 (stacked on #9616).
Generated GraphQL types/documents were regenerated from #9657 commit
`95e486996781dc4da15191a881d65f1f2b3a1104`'s `api/schema.graphql` and
`api/mutations.graphql`. Once available in a running API, use the existing
`pnpm graphql-codegen` workflow pointed at that API.

The frontend requires `VITE_API_URL` and a `VITE_STRIPE_PUBLISHABLE_KEY` matching
the backend Stripe account/mode. Existing Clerk and application environment
configuration still applies. Do not enable verification rollout until the
backend and this route have passed an end-to-end environment test.

All verification mutations go from the browser to app-api, so per-IP limits
count the user's address. GraphQL payload errors and transport errors are
handled separately; phone/code/card data and client secrets are not logged or
persisted by this integration.

## Status and progress

`VERIFIED` and `EXEMPT` complete the flow even with no passed checks. No other
status can complete it. Three saved checks with `REQUIRED` show finalization
pending; checking email again can finish it after a rate limit clears.
`RATE_LIMITED` is recoverable and has no invented unlock timestamp or permanent
support-only lock.

The pure adapter maps backend summaries into the UI presentation state. The
controller owns transient SMS challenge data, pending state, cooldown, and
mutation failures. Missing email is checked once automatically for a matching
admin account; failed email checks require profile recovery and explicit
Continue. Reopening/reloading recovers passed checks but not an unfinished SMS
challenge, which the read contract does not expose.

Sandbox entry uses backend eligibility for both free and paid accounts. Required
accounts see verification; suspended accounts see the blocked treatment;
verified/exempt accounts see the sandbox list. Accounts outside verification
keep the existing entitlement-based list/upgrade behavior. List and detail
navigation share this policy. APIs remain responsible for permissions and
operation entitlements; the frontend grants neither.

## Stripe

Hosted CardElement fields collect number, expiry and CVC; required name and ZIP
are included in `createPaymentMethod` billing details. Link is disabled, and no
wallet controls are mounted. The page calls `addComputeVerificationCard` with
the PaymentMethod ID, never `createToken` or the billing token mutation.

The backend confirms the SetupIntent. If it returns `cardClientSecret`, the page
runs `handleNextAction` and calls `checkComputeVerificationCard` with the returned
SetupIntent ID. In-session retry retains that ID after an uncertain finish and
avoids another card submission. A failed response triggers a status read before
another write is offered. The API does not expose pending authentication in the
read query, so full reload recovery during 3DS is not supported by this contract.

## Validation and local review

Run `pnpm test src/components/ComputeVerification src/components/Sandboxes/SandboxesVerificationCallout.test.tsx src/components/Sandboxes/SandboxesAccess.test.tsx`,
`pnpm type-check`, and `pnpm build:vite` from `ui/apps/dashboard`.

Tests cover the connected page's email → phone → card flow, payload/network
failures, rate-limited finalization, account/permission guards, Stripe orchestration,
and backend-driven sandbox entry. Stripe tests mock the SDK and backend results.
They do not replace an environment test with real Stripe test-mode 3DS and the
running backend stack. Backend fake SMS uses `000000`; the isolated UI demo uses
`123456`.

The existing UI demos remain opt-in: `pnpm preview:verification` and
`pnpm preview:sandbox-verification`. They do not call the live verification APIs.
