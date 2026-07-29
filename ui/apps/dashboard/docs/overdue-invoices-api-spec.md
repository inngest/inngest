# API Spec: `account.paymentStatus` — overdue-invoice GraphQL endpoint

This spec is for the **API server repo**. It defines a new GraphQL field that
powers an in-app overdue-invoice banner in the dashboard. The dashboard side is
already built and gated behind a feature flag (`overdue-invoice-banner`); it
will not call this field in production until the flag is enabled, so you can
ship and verify the API independently.

## Goal

Expose the account's collections / payment state so the dashboard can warn users
about overdue or unpaid invoices and escalate the warning as the invoice ages.

**Design principle — compute policy server-side.** The dashboard intentionally
does _not_ know the day thresholds for "failed payment" vs "final notice" vs
"downgrade". The API returns a precomputed `severity` and `stage`; the client
maps those to copy and color. This keeps collection thresholds in one place
(tunable without a frontend deploy) and consistent with email/dunning logic.

---

## Schema additions

Add a nullable `paymentStatus` field to `Account`. **It must return `null` when
the account is in good standing** (no overdue invoices and no failed payment).
This lets the client early-return without any rendering logic, and naturally
excludes accounts that can't be in collections (free / non-billable).

```graphql
extend type Account {
  """
  Collections / payment status for the account. Null when the account is in
  good standing (no overdue invoices and no failed payment). Powers the in-app
  overdue-invoice banner.
  """
  paymentStatus: AccountPaymentStatus
}

type AccountPaymentStatus {
  "Highest severity across all open/overdue invoices. Drives banner color."
  severity: PaymentStatusSeverity!

  """
  Machine-readable collections stage, computed server-side from invoice age and
  payment state. Drives banner copy and /billing detail messaging.
  """
  stage: PaymentCollectionStage!

  "Total past-due amount across all overdue invoices, in cents."
  amountDueCents: Int!
  "Pre-formatted amount for display, e.g. \"$240.00\"."
  amountDueLabel: String!
  "ISO 4217 currency code, e.g. \"usd\"."
  currency: String!

  "Days the oldest overdue invoice is past due. 0 if a payment failed but nothing is overdue yet."
  daysPastDue: Int!

  "Whether the most recent payment attempt failed (card declined, etc.)."
  hasFailedPayment: Boolean!

  "When the pending action takes effect (downgrade/suspension), if scheduled. Null otherwise."
  actionDate: Time

  "What happens at actionDate. Null when nothing is scheduled."
  pendingAction: PaymentPendingAction

  "Per-invoice detail for the /billing page banner."
  overdueInvoices: [OverdueInvoice!]!

  "Most direct link to resolve payment (hosted invoice URL or billing portal). Must be https."
  resolveURL: String!
}

enum PaymentStatusSeverity {
  WARNING   # failed payment / within grace window
  CRITICAL  # past stricter threshold, downgrade/suspension imminent or active
}

enum PaymentCollectionStage {
  PAYMENT_FAILED      # card declined / retrying, not yet past grace
  PAST_DUE            # overdue, within grace window
  FINAL_NOTICE        # overdue beyond stricter threshold
  DOWNGRADE_PENDING   # scheduled to downgrade on actionDate
  DOWNGRADED          # already downgraded for non-payment
  SUSPENDED           # account suspended for non-payment
}

enum PaymentPendingAction {
  DOWNGRADE
  SUSPEND
}

type OverdueInvoice {
  id: ID!
  amountCents: Int!
  amountLabel: String!     # e.g. "$120.00"
  currency: String!
  dueAt: Time!
  daysPastDue: Int!
  status: String!          # underlying invoice status (open, uncollectible, …)
  invoiceURL: String       # hosted invoice / pay link (https), null if none
  attemptedAt: Time        # last payment attempt, null if none
  failureReason: String    # e.g. "card_declined", null if none
}
```

> `Time` is the existing custom scalar used elsewhere in the schema (RFC 3339).
> If the schema already has a money/currency scalar or shared invoice type,
> prefer reusing it over the `*Cents` + `*Label` pair shown here — just keep the
> field names the client reads (see the query below) intact, or coordinate a
> rename.

---

## Exact query the dashboard issues

The client sends this operation verbatim. The response shape must satisfy it.
Note the client requests a **subset** of the fields above — it does not select
`amountDueCents`, `currency`, `attemptedAt`, or per-invoice `amountCents`. Those
are in the schema for completeness/future use; the operation below is the
binding contract.

```graphql
query PaymentStatus {
  account {
    id
    paymentStatus {
      severity
      stage
      amountDueLabel
      daysPastDue
      hasFailedPayment
      actionDate
      pendingAction
      resolveURL
      overdueInvoices {
        id
        amountLabel
        dueAt
        daysPastDue
        status
        invoiceURL
        failureReason
      }
    }
  }
}
```

This runs as a **client-side (browser) request** to `/gql` — the same path and
exchange as every other dashboard client query (e.g. `VercelIntegration`,
`GetAccountEntitlements`) — not a server-to-server call. It carries the Clerk
bearer token (with session-cookie credentials as the fallback) and resolves the
account from that auth context. No arguments. CORS for the dashboard origin must
allow it, which it already does for the existing client queries.

---

## Stage & severity computation

`stage` and `severity` are derived from invoice age + payment state. The
thresholds below are a **proposal matching the product intent** ("update card"
for a recent failure, a stricter warning past ~1–2 weeks) — the API team owns
the final numbers. Put them behind config/constants so they're tunable.

Evaluate top-to-bottom; first match wins. `daysPastDue` = days since the oldest
unpaid invoice's `dueAt` (0 if none is past due yet).

| Condition                                                      | `stage`             | `severity` |
| -------------------------------------------------------------- | ------------------- | ---------- |
| Account already suspended for non-payment                      | `SUSPENDED`         | `CRITICAL` |
| Account already downgraded for non-payment                     | `DOWNGRADED`        | `CRITICAL` |
| Downgrade/suspension scheduled (`actionDate` set, not yet hit) | `DOWNGRADE_PENDING` | `CRITICAL` |
| Overdue ≥ stricter threshold (proposed: **7 days**)            | `FINAL_NOTICE`      | `CRITICAL` |
| Overdue ≥ 1 day, below stricter threshold                      | `PAST_DUE`          | `WARNING`  |
| Payment failed but nothing overdue yet (e.g. retrying)         | `PAYMENT_FAILED`    | `WARNING`  |
| Otherwise (paid up, no failed payment)                         | — return `null`     | —          |

Notes:

- `severity` is fully determined by `stage` (WARNING for `PAYMENT_FAILED` /
  `PAST_DUE`; CRITICAL for everything else). The client maps WARNING→`warning`
  (yellow, dismissible) and CRITICAL→`error` (red, pinned).
- `pendingAction` / `actionDate` should be populated for `DOWNGRADE_PENDING`
  (and may be populated for `FINAL_NOTICE` if a downgrade is already scheduled,
  which lets the client show the date earlier).
- `daysPastDue` reflects the **oldest** overdue invoice so the stage escalates
  with the worst invoice, not the newest.

---

## Behavioral requirements & edge cases

- **Good standing → `null`.** Do not return an empty `AccountPaymentStatus`.
- **Non-billable accounts** (free / trial / marketplace that can't owe an
  invoice) → `null`. If such an account can carry a one-off overdue invoice,
  return the status normally — confirm which applies (open question below).
- **Multiple overdue invoices.** Aggregate: `amountDueCents`/`amountDueLabel` =
  sum across all overdue invoices; `daysPastDue` = from the oldest; list each in
  `overdueInvoices` newest-due-first or oldest-due-first (pick one and be
  consistent — the client renders them in array order).
- **Currency.** Assume a single currency per account; if mixed, return the
  account's primary and ensure `amountDueLabel` is internally consistent.
- **Paid-just-now races.** Source from already-synced invoice/subscription
  state. A brief staleness window is fine (client caches 60s and polls every
  5m); don't block the request on a live provider round-trip.
- **`resolveURL` and `invoiceURL` must be `https`.** The client defends in depth
  but the API is the source of truth — never emit `javascript:`/`data:` or other
  schemes. `resolveURL` should be the most direct path to pay (hosted invoice
  when there's a single one, otherwise the billing portal / update-card flow).

## Performance

This field is fetched on **every authenticated dashboard page load** (cached
60s client-side, polled every 5m). Keep the resolver cheap:

- Back it with synced invoice/subscription state, not a synchronous Stripe call.
- Cache server-side if needed; the client tolerates ~60s staleness.
- Returning `null` (the common case) should be very fast — ideally a cheap check
  on the account's billing state without enumerating invoices.

## Acceptance criteria

1. Account paid up, no failed payment → `account.paymentStatus` is `null`.
2. Recent failed payment, nothing overdue → `stage: PAYMENT_FAILED`,
   `severity: WARNING`, `hasFailedPayment: true`, `daysPastDue: 0`.
3. One invoice 3 days overdue → `stage: PAST_DUE`, `severity: WARNING`,
   `daysPastDue: 3`, one entry in `overdueInvoices` with a valid `invoiceURL`.
4. One invoice 10 days overdue (≥ stricter threshold) → `stage: FINAL_NOTICE`,
   `severity: CRITICAL`.
5. Downgrade scheduled → `stage: DOWNGRADE_PENDING`, `severity: CRITICAL`,
   `actionDate` set, `pendingAction: DOWNGRADE`.
6. Already downgraded / suspended → `DOWNGRADED` / `SUSPENDED`, `CRITICAL`.
7. `amountDueLabel` equals the sum of `overdueInvoices` amounts; `daysPastDue`
   equals the oldest invoice's age.
8. `resolveURL` and every `invoiceURL` are `https` URLs.
9. The exact `PaymentStatus` query above resolves with no GraphQL errors for all
   of the states above.

## Open questions for the API team to decide

1. Final threshold values for each stage (proposal: failed/grace < 7d → WARNING,
   ≥ 7d → FINAL_NOTICE; adjust to match dunning policy).
2. Should `resolveURL` be a hosted invoice URL, the billing portal, or an in-app
   update-card flow?
3. Are `DOWNGRADED` and `SUSPENDED` distinct states for our plans, or do we
   collapse to one terminal stage?
4. Can free / marketplace accounts ever carry an overdue one-off invoice, or
   should they always return `null`?
5. Do we need to gate this field by RBAC (billing-admin only), or is any
   authenticated account member allowed to see payment status?
