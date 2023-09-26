### SDK request versions

`driver.SDKRequestContext.HashVersion` && `state.Metadata.RequestVersion` indicate the request
version used to POST data to the SDK.  This is critically important for replay;  changing the
request payload breaks the exactly-once guarantees of functions.

Request versions can change with any of the following:

- Step hashing changes
- Input types change
- Input data changes (eg. per-step errors)

v1 & v2 of the TS SDK used a hashing method for steps which is implementation-specific.  We changed
the way steps are hashed in v3 of the TS SDK, allowing cross-language, cross-platform live
migrations of state.

The format is as follows:

* -1: The hash version is unset, and needs to be set on the first SDK response.
* [n]: The first hash version is as specified

**Current versions**

- `1`: Steps are hashed in the following format: `fmt.Sprintf("%s:%d", stepID, idx)`

**How it works**

Whenever we instantiate a new fn, the hash version is set to `-1` - ie. unknown.  The first SDK
request should respond with a hash version which is set in state metadata.

When running steps, we can compare the stored hash version with the SDK's latest response.  If
versions change, we can warn or fail depending on strictness.

NOTE: v1 and v2 of the TS SDKs do _not_ support hash versions.  This implicitly means that a '0'
hash version uses the TS-specific style of hashing from Jan 23-Sep 23.

**SDK Compatibility**

All SDKs **must** respond with an `x-inngest-req-version` header indicating the version used.
