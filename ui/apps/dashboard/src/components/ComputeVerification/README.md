# Compute and Sandboxes verification

The production `/verify?account_id=<UUID>` route now uses browser URQL operations
and Stripe Elements. See [integration setup and validation](../../../docs/compute-verification-integration.md)
for backend dependencies, status mapping, rate limits, and the card/3DS flow.
The existing header, company/profile menus, search, footer, and Phone → Card
indicator are reused.

## Isolated UI demo

Run `pnpm preview:verification` from `ui/apps/dashboard` and open
<http://127.0.0.1:5174/verification-demo.html>. This opt-in Vite entry remains
separate from the production route and does not call verification services.

Expand **Development demo · scenarios** to select any of the 34 fixtures.
Direct links use `?scenario=<id>`, such as `saved`, `sms-expired`, `mismatch`,
or `complete-api`. Reset restores the fixture. The happy path accepts
**(415) 555-0132**, SMS code **123456**, and a populated read-only dummy card.
The backend development fake uses **000000**, not the UI demo's code.

Mock card fields stay fixed and read-only; the production adapter uses hosted
Stripe fields instead. Demo return/support callbacks remain local. The demo
reuses native menus with service fixtures and explicit organization/profile
choices. None of these fixture adapters is imported by the production route.

Review screenshots from the UI work: [phone](../../../docs/compute-verification/phone.jpg),
[SMS error](../../../docs/compute-verification/sms-error.jpg), and
[card error](../../../docs/compute-verification/card-error.jpg).

## Presentation contract

`VerificationState` is the frontend presentation type. The production adapter
maps backend status and checks into it; demo fixtures supply it directly.
Completion is an explicit `complete` status, never inferred from three passed
checks. Production also supports recoverable rate limits and pending finalization.

Changing account/profile context resets local inputs, errors, and card challenges.
Account selection remains user-controlled. Early-access request/approval screens
are deferred. CLI/API completion has no return button; dashboard completion
returns to the supplied local sandbox path or dashboard.

The card adapter receives no raw card values through UI callbacks. General
server card failures are form-level; only known field validation marks individual
inputs invalid. Verification itself is $0; a verified card becomes the default
payment method for future billing, as explained by the form.
