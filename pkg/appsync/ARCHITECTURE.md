# Architecture: pkg/appsync

## Purpose

In-band sync is server-initiated: the Inngest server makes a signed `PUT` to
the SDK's app URL and parses the signed register payload from the response
in one round-trip. Compare to `/fn/register`, which is SDK-initiated.

This package owns the HTTP exchange and signature handling. It does not
persist the result and does not decide what to do with it.

## Sync return contract

`Sync(ctx, opts) (*Response, *syscode.Error, error)`. Exactly one is non-nil.

- **`*Response`**: Success, body validated against signature.
- **`*syscode.Error`**: "Userland" error. This indicates something out of our control failed in the sync (wrong signing key, unreachable SDK, etc.).
- **`error`**: "System" error. This indicates a bug in our code (bad opts, marshaling).

## Security model

Does not trust the caller-supplied URL, even though it was probably authenticated at the API layer. This package is defensive.

### Signed both ways

Request and response both carry `X-Inngest-Signature` over the body using
the configured signing key. The response signature is validated *before*
parsing the body. Pre-validation bytes are untrusted.

### Network policy

- **Private networks**: always allowed. Self-hosted setups commonly run the
  SDK on the same box as Inngest, so blocking loopback/RFC1918 isn't useful.
- **Insecure HTTP**: opt-in via `Opts.AllowInsecureHTTP` (default `false`).
  `inngest dev` always passes `true`; `inngest start` exposes it via the
  `--allow-insecure-http` flag (default `false`).
- **Redirects**: refused outright (see below).

### Other safety properties

- **Body cap**: 10 MiB, checked before signature validation, bounds memory under a malicious endpoint.
- **Error messages**: Never echo upstream body bytes or Go's dial errors in `syscode.Error.Message`. Pre-signature bytes are attacker-controlled, and dial errors leak resolved IPs. Sanitize; keep raw bytes in logs.
- **Why redirects are refused**: the SDK URL is canonical. Following redirects would add a header-forwarding surface (the request carries a signed body) and another code path to maintain for negligible benefit.

## Out of scope

- **Persistence and downstream effects**: Caller's job.
- **Signing key selection**: `Opts.SigningKey` is taken as given.
- **HTTP status mapping**: This package categorizes failures via `syscode.Error.Code`; status translation is the caller's choice.
