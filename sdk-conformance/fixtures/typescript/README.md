# TypeScript SDK conformance fixture

An independent, directly probeable HTTP fixture for the published TypeScript
SDK `inngest@4.21.0`. It does not use the legacy conformance runner.

## Run

Requires Node.js 20 or newer.

```bash
npm ci
npm run check
PORT=3000 npm start
```

The process binds to `127.0.0.1` and exposes:

- `GET /__conformance` — fixture manifest;
- `POST /__conformance/reset` — reset mutable fixture state (this fixture is
  currently stateless and returns `204`);
- `/api/inngest` — the normal Inngest SDK serve handler.

For example:

```bash
curl http://127.0.0.1:3000/__conformance
curl http://127.0.0.1:3000/api/inngest
```

The app ID is `sdk-conformance`. Function IDs, trigger event names, step IDs,
outputs, sleep duration, wait timeout, and error messages are fixed in
`src/server.ts` so protocol probes are repeatable. The wait fixture is resumed
by `conformance/wait-for-event.resume`.
