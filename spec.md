# In-Band Sync

In-band sync is a synchronous request/response sync: the server sends one
signed `PUT` to the SDK and receives the full register payload in the same
response. Both directions are signed.

Reference implementation: `monorepo/pkg/applogic/sdkhandlers/{api,registration,synchronization}.go`.

## Wire protocol

### Request (server → SDK)

- `PUT <appURL>` with body `{"url": "<appURL>"}`.
- Required headers:
  - `content-type: application/json`
  - `x-inngest-server-kind: cloud` (or `dev`)
  - `x-inngest-sync-kind: in_band` (`inngestgo.SyncKindInBand`)
  - `x-inngest-signature: <inngestgo.Sign(skey, body)>`

A signing key is required.

### Response (SDK → server)

A response is accepted iff it is 2xx, has a valid `x-inngest-signature`, and
parses as the body shape below. The response sync-kind header is not checked.

- Required headers:
  - `x-inngest-signature: <hmac of body with skey>`
- Body (JSON), per the inngestgo handler:
  ```
  {
    "app_id":       string,
    "env":          *string,
    "framework":    *string,
    "platform":     *string,
    "sdk_author":   string,
    "sdk_language": string,
    "sdk_version":  string,
    "url":          string,
    "functions":    []sdk.SDKFunction,
    "inspection":   object   // polymorphic; capabilities live at inspection.capabilities
  }
  ```

## Package: `pkg/appsync`

Sends the request and parses the response. Does **not** process the resulting
register payload. The caller does that (this repo and the cloud monorepo
process differently).

Behavior:
1. Validate `x-inngest-signature` against the body via
   `inngestgo.ValidateResponseSignature(skey, body)`. Invalid → fail.
2. Unmarshal into the in-band response struct.
3. Convert to `*sdk.RegisterRequest` (mapping: `app_id` → `AppName`,
   capabilities pulled from `inspection.capabilities`,
   `SDK = "<sdk_language>:<sdk_version>"`).

## Errors

- Missing or invalid response signature → fail.
- Cloudflare interstitial (`Cf-Mitigated` header) → fail.
- Non-2xx status → fail with the body surfaced.
