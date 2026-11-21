# Compute and Sandboxes verification UI

Early-access request, received, and approval screens are deferred.

UI-only implementation of the final [Figma section](https://www.figma.com/design/OGjjin5JHqKNlr0tiu8inX/Compute?node-id=12909-276). No verification API, authentication, SMS, Stripe, eligibility enforcement, access grant, persistence, or CLI behavior is implemented.

## Review locally

From `ui/apps/dashboard`, run `pnpm dev:verification` with the repository-supported Node 22 version and installed UI dependencies. Open <http://127.0.0.1:5174/verification-demo.html>.

Expand **Development demo · scenarios** to select any of the 34 scenarios. Each has a direct link using `?scenario=<id>`; for example `?scenario=saved`, `?scenario=sms-expired`, `?scenario=mismatch`, or `?scenario=complete-api`. Reset restores its supplied fixture. Controls are separate from the product UI.

Review screenshots: [phone entry](../../../docs/compute-verification/phone.jpg), [SMS error](../../../docs/compute-verification/sms-error.jpg), and [card error](../../../docs/compute-verification/card-error.jpg).

Happy path: open `?scenario=happy`, enter **(415) 555-0132**, paste **123456** into the SMS input, then click **Verify card**. The happy path prefills the read-only dummy card automatically. Other card fixtures can be populated using **Development demo · scenarios** → **Populate dummy card**. Test instructions stay inside the collapsed review controls; no demo helper text appears beside the verification screens. Other six-digit codes produce an incorrect-code error. The configurable countdown is represented by supplied remaining seconds; the demo starts it at 30 seconds. Loading fixtures stay pending until another scenario or Reset is selected.

Email recovery: **Manage email in your profile** simulates updating the primary email; **Continue verification** then resumes. Continuing before managing the email demonstrates the still-needs-attention error. Mismatch: the header renders the existing app `OrgMenu` and `AvatarMenu`/`ProfileMenu` components unchanged. Use **Switch organisations** or **Switch account** to navigate to a demo picker, then explicitly select a company or profile. The picker fixtures contain three of each and accept any selection; there is no automatic toggle or choice of the originating account. The mismatch panel is informational and stays visible until the user-selected company and profile both match. Switching either context discards local forms, errors, pending mock results, and account-specific progress.

## Components and integration contract

- `ComputeVerification` renders only from typed `VerificationState` and `VerificationActions`. The first incomplete check is selected in `state.ts`; phone entry and SMS share Step 1 of 2. Matching account and profile context precedes verification and completion. Non-admin users receive an admin-required explanation without forms or a per-check status breakdown.
- `PhoneScreen`, `CodeScreen`, and `OTPInput` handle local validation and editing. Phone values survive recoverable errors and change-number navigation. OTP supports a complete pasted/autofilled code, arrows, Home/End, Backspace, and Delete. Screen changes focus the heading; errors are announced and associated with fields.
- `CardScreen` accepts `renderCardFields({ disabled, errorID })` and `cardReady`. `errorID` describes a form-level verification failure and must not mark every input invalid. Field highlighting belongs only to specific validation errors reported by the field adapter; generic, bank, and eligibility failures remain a single form-level message. The future Stripe adapter supplies its controlled field area and readiness. `onSubmitCard()` receives no card details. Without an adapter, the form renders an unavailable placeholder and cannot submit.
- `EmailScreen`, `MismatchScreen`, `BlockedScreen`, and `CompletionScreen` expose their actions via callbacks. Status and permission decisions remain the caller's responsibility. Clicking a callback never changes supplied status inside these presentation components.
- `VerificationPage` composes the existing `TopBar`, organization menu, profile menu, search, and `BottomBar` beneath existing app providers, without a sidebar. Its `profile` and `showOnboardingWidget` props come from the host. Optional focused modes on the existing bars remove environment navigation and permit the footer to wrap. `VerificationLayout` also accepts header/footer slots; the standalone demo renders the same `OrgMenu`, `OrgButton`, `AvatarMenu`, and `ProfileMenu` components. A demo-only memory router preserves their navigation and presents explicit fixture choices on destination pages. The demo Vite config substitutes the plan, onboarding, Clerk, environment, and quick-search service hooks with `shellServices.ts`; it does not replace or edit the menus.

`VerificationState` supplies selected/originating account IDs and masked profiles, entry source (`cli`, `dashboard`, `api`), permissions, email/phone/card progress, pending state, typed errors, cooldown seconds, destination phone, and optional masked verified details. Changing the selected account/profile key resets local input state. The integration owner must supply the new account's progress; the UI does not transfer progress.

Callbacks cover sending/verifying/resending SMS, changing number, submitting card, managing/continuing email, switching company/profile, retrying, returning to the originating context, and contacting support. Callback promises disable related actions and reject into a recoverable local error. All copy variants live in `copy.ts` or their screen modules.

## Mock boundary

Everything under `demo/` is development-only and has no imports from production entry points. The existing search trigger and command palette are also reused: click search or press Command-K/Ctrl-K, search for demo-app or Demo function, or use the existing navigation shortcuts. Demo navigation stays local; production uses the existing app destinations.

The original company and profile menu modules are unchanged; the standalone demo supplies service fixtures and a memory router around them. The separate Vite config refuses production builds. No application route or auth configuration is changed. The integration owner must mount `VerificationPage` at the final verification URL and skip the normal outer layout there, as the existing OAuth approval pages do.

Card fixtures are **fixed and read-only**, with demo-only controls to populate them. They cannot accept real card data, are not submitted to callbacks, and are never stored or logged. Mock progress lives only in React memory. Return and support actions report their invocation in demo controls; they navigate nowhere and send nothing. No optional GitHub/LinkedIn steps or 3DS screens are included.

The integration engineer owns status retrieval, progress persistence, real account switching, permission enforcement, SMS/cooldown services, the vendor-controlled card adapter, bank authentication, email management, final route/destinations, and access grants. Existing shell integrations are reused unchanged by `VerificationPage`; none are used by the standalone demo.

## Copy/design follow-ups

1. The brief flags payment-method messaging on the phone screen. The current final phone frame (`12909:277`) contains only the US-mobile/VoIP description; the final card frame (`12909:363`) contains the default-payment-method message. This implementation follows those frames. Confirm the intended source of truth if the design changes.
2. **You won’t be charged** and **$0.00** describe the card verification check only. The default-payment-method explanation is retained. Confirm whether the reassurance needs qualification for later paid-plan billing.
3. Completion uses a single description for each entry source. CLI completion displays **Return to your terminal to continue your deploy.** without a button. API/direct-link completion displays **You can now use Compute and Sandboxes. Retry your original request.** without a button, since there is no known app destination. Dashboard completion retains **Back to dashboard**; the owner must connect that return callback.
4. Admin-required copy reads **An account admin needs to complete verification.** and omits the check-status breakdown. The current application shell differs from Figma's footer links/Ask Inngest treatment; the host wrapper reuses the existing shell rather than adding a new support/search integration.

## Validation

Run `pnpm exec vitest run src/components/ComputeVerification/ComputeVerification.test.tsx`, `pnpm type-check`, and targeted ESLint/Prettier checks. The suite covers all supplied scenarios, local phone validation, duplicate submissions, OTP paste/autofill/editing, disabled states, context resets, callbacks, source-specific completion, and the full mock path. The demo additionally supports visual review at desktop and narrow widths.
