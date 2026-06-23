# Overdue-Invoice Banner — Design & Implementation Plan

Adds an account-wide banner warning of overdue/unpaid invoices, plus a richer
banner on `/billing`. Severity escalates with how long the invoice has been
past due. Modeled on the existing `ActiveBanners` flow.

## Goals

- On any authenticated page load, fetch the account's payment status and show a
  top-of-app banner when there are overdue/unpaid invoices.
- Escalating severity:
  - Failed payment / < grace threshold → "Update your credit card" (warning).
  - Past stricter threshold (e.g. 7–14 days) → stronger warning that the card
    **must** be updated to keep the account open (error).
  - Downgraded / suspended → "contact support" messaging.
- On `/billing`, render a detailed banner below the Overview with per-invoice
  info, downgrade date, and a support CTA.
- Banner links to the billing dashboard to update card / pay invoices.

## Design principle: compute severity in the API, not the UI

The thresholds in the request ("< 2 days", "> 7 or 14 days") are **business
policy** and should live in one place. The API returns a computed `severity`
and `stage`; the dashboard maps those to copy and `Banner` severity. This keeps
collection thresholds tunable without a frontend deploy and keeps the same
logic consistent across email, in-app, and any future surfaces.

---

## GraphQL endpoint (proposal for the API repo)

Expose a nullable field on `Account` — consistent with `account.activeBanners`.
**Returns `null` when the account is in good standing** (no overdue invoices,
no failed payment). This also naturally excludes free / marketplace accounts
that can't have overdue invoices.

```graphql
extend type Account {
  """
  Collections / payment status for the account. Null when the account is in
  good standing. Drives the in-app overdue-invoice banner.
  """
  paymentStatus: AccountPaymentStatus
}

type AccountPaymentStatus {
  "Highest severity across all open/overdue invoices. Drives banner color."
  severity: PaymentStatusSeverity!

  """
  Machine-readable collections stage. Drives banner copy and the /billing
  detail messaging. Computed server-side from invoice age + payment state.
  """
  stage: PaymentCollectionStage!

  "Total past-due amount across all overdue invoices, in cents."
  amountDueCents: Int!
  "Pre-formatted amount for display, e.g. \"$240.00\"."
  amountDueLabel: String!
  currency: String!

  "Days the oldest overdue invoice is past due (0 if a payment failed but nothing is overdue yet)."
  daysPastDue: Int!

  "Most recent payment attempt failed (card declined, etc.)."
  hasFailedPayment: Boolean!

  "When the pending action takes effect (downgrade/suspension), if scheduled."
  actionDate: Time
  "What happens at actionDate."
  pendingAction: PaymentPendingAction

  "Per-invoice detail for the /billing page banner."
  overdueInvoices: [OverdueInvoice!]!

  "Deep link to resolve: hosted billing portal or invoice pay URL."
  resolveURL: String!
}

enum PaymentStatusSeverity {
  WARNING   # failed payment / within grace window
  CRITICAL  # past stricter threshold, suspension/downgrade imminent or active
}

enum PaymentCollectionStage {
  PAYMENT_FAILED      # card declined / retrying, not yet past grace (e.g. < 2d)
  PAST_DUE            # overdue, within grace window
  FINAL_NOTICE        # overdue beyond stricter threshold (e.g. 7–14d)
  DOWNGRADE_PENDING   # scheduled to downgrade on actionDate
  DOWNGRADED          # already downgraded for non-payment
  SUSPENDED           # account suspended
}

enum PaymentPendingAction {
  DOWNGRADE
  SUSPEND
}

type OverdueInvoice {
  id: ID!
  amountCents: Int!
  amountLabel: String!
  currency: String!
  dueAt: Time!
  daysPastDue: Int!
  status: String!       # underlying invoice status (open, uncollectible, …)
  invoiceURL: String    # hosted invoice / pay link
  attemptedAt: Time     # last payment attempt
  failureReason: String # e.g. "card_declined"
}
```

### Notes for the API implementation

- `severity` and `stage` are derived server-side from invoice age and payment
  state. Suggested mapping (thresholds owned by the API, shown here only to
  illustrate the request's intent):
  - `PAYMENT_FAILED` / `PAST_DUE` → `WARNING`
  - `FINAL_NOTICE` / `DOWNGRADE_PENDING` / `DOWNGRADED` / `SUSPENDED` → `CRITICAL`
- `resolveURL` should be the most direct path to payment (hosted invoice URL if
  a single invoice, otherwise the billing portal / update-card flow).
- Keep the resolver cheap — it's hit on every authenticated page load. Cache or
  back it with already-synced invoice state rather than a live Stripe call.
- Return `null` (not an empty object) for healthy accounts so the client can
  early-return without rendering logic.

---

## Dashboard implementation

### 1. Client query — `src/components/PaymentStatusBanner/usePaymentStatus.ts`

The query runs **client-side** via urql (`useSkippableGraphQLQuery`), the same
exchange the rest of the dashboard's client queries use — a browser CORS request
to `/gql` carrying the Clerk bearer token. The document and result types are
co-located in the banner folder (`types.ts`).

```ts
import { gql } from 'urql';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { useSkippableGraphQLQuery } from '@/utils/useGraphQLQuery';
import { type AccountPaymentStatus } from './types';

const paymentStatusQuery = gql<PaymentStatusResult, Record<string, never>>`
  query PaymentStatus {
    account { id paymentStatus { severity stage amountDueLabel daysPastDue
      hasFailedPayment actionDate pendingAction resolveURL
      overdueInvoices { id amountLabel dueAt daysPastDue status invoiceURL failureReason } } }
  }
`;

export function usePaymentStatus(): AccountPaymentStatus | null {
  const { value: enabled } = useBooleanFlag('overdue-invoice-banner');
  const res = useSkippableGraphQLQuery({
    query: paymentStatusQuery,
    variables: {},
    skip: !enabled,
    pollIntervalInMilliseconds: 5 * 60_000,
  });
  return res.data ? res.data.account.paymentStatus : null;
}
```

**As-built note:** because `account.paymentStatus` does not exist in the
introspected schema yet, the document uses urql's plain `gql` tag (with
hand-written types in `types.ts`), **not** the codegen `graphql()` tag — so it
can't break codegen for the rest of the app. Once the API field lands, run
`pnpm graphql-codegen`, switch to the `graphql()` document + generated
`PaymentStatusQuery` types, and delete the manual types.

### 2. Global banner — `src/components/PaymentStatusBanner/`

- `usePaymentStatus.ts` — shared client query (above), gated by the
  `overdue-invoice-banner` flag via `skip`. urql dedupes the global and billing
  banners into one request; polls every 5 minutes so a resolved payment clears
  the banner. No data while loading/skipped/errored → renders nothing.
- `PaymentStatusBanner.tsx` — thin container; renders nothing when the hook
  returns `null` (good standing or flag off).
- `PaymentStatusBannerView.tsx` — maps `stage` → copy + `Banner` severity,
  renders the `Banner` with a CTA `Button` linking to `pathCreator.billing()`.

Stage → presentation map (single source of copy on the client):

| stage               | Banner severity | Copy (summary)                                                                | Dismissible |
| ------------------- | --------------- | ----------------------------------------------------------------------------- | ----------- |
| `PAYMENT_FAILED`    | warning         | "Your last payment failed. Update your credit card to avoid issues."          | per-session |
| `PAST_DUE`          | warning         | "You have an overdue invoice ({amountDueLabel}). Update your card."           | per-session |
| `FINAL_NOTICE`      | error           | "Overdue {daysPastDue}d. Update your card to keep your account open."         | no          |
| `DOWNGRADE_PENDING` | error           | "Account will be downgraded on {actionDate}. Pay to keep your plan."          | no          |
| `DOWNGRADED`        | error           | "Account downgraded for non-payment. Pay overdue invoices / contact support." | no          |
| `SUSPENDED`         | error           | "Account suspended for non-payment. Contact support."                         | no          |

Dismissal uses `useBooleanLocalStorage('PaymentStatusBanner:visible:${stage}',
true)` for the `WARNING` tier only — keyed by `stage` so dismissing a soft
warning doesn't suppress a later, more severe notice. CTA button uses
`kind="danger"` for error, `secondary` for warning.

### 3. Mount in the layout

`src/components/Layout/Layout.tsx` is the only authenticated layout (there is no
`Layout` in the current tree). `<PaymentStatusBanner />` is mounted right after
`<ActiveBanners />`:

```tsx
<IncidentBanner />
<ActiveBanners />
<PaymentStatusBanner />
{children}
```

### 4. `/billing` detail banner — `BillingPaymentStatusBanner.tsx`

Rendered at the top of the Overview content in
`src/routes/_authed/billing/index.tsx` (above the plan grid). It calls the same
`usePaymentStatus()` hook — **no loader change**, and React Query dedupes it with
the global banner's request. Uses `ContextualBanner` (title + `.List`) to show:

- Title from `stage` (e.g. "Your account will be downgraded on {actionDate}").
- A list of `overdueInvoices` — amount, due date, days past due, with a
  per-invoice "Pay invoice" link (`invoiceURL`) where present.
- A "View invoices" CTA and, for `DOWNGRADED` / `SUSPENDED`, a "Contact support"
  `ContextualBanner.Link`.

### 5. Feature flag

Both surfaces are gated by the `overdue-invoice-banner` boolean flag inside
`usePaymentStatus()` (`enabled` on the query). The query does not fire while the
flag is off, so nothing breaks before the API field ships — flip the flag on
once `account.paymentStatus` is live.

---

## Open questions for the API team

1. Final threshold values for each `stage` (the request suggests ~2d for
   failed-payment and ~7–14d for the stricter notice).
2. Should `resolveURL` be a hosted Stripe invoice URL, the billing portal, or
   our in-app update-card flow?
3. Is `DOWNGRADED` distinct from `SUSPENDED` for our plans, or do we collapse to
   one terminal stage?
4. Should free/marketplace accounts always return `null`, or can they carry
   overdue one-off invoices?
