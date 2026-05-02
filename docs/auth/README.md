# Host Authentication for Self-Hosted Inngest

Optional password-based authentication for the Inngest self-hosted dashboard. When enabled, users must sign in before accessing the web UI or GraphQL API.

![Login Screen](login-screenshot.png)

## Quick Start

Set two environment variables and restart the server:

```bash
export INNGEST_HOST_EMAIL="admin@yourcompany.com"
export INNGEST_HOST_PASSWORD="your-secure-password"

inngest start \
  --signing-key=<your-signing-key> \
  --event-key=<your-event-key> \
  --sqlite-dir=/data/inngest
```

Visit `http://localhost:8288` and you'll be redirected to the login page.

## How It Works

- **Auth is optional** -- when `INNGEST_HOST_EMAIL` and `INNGEST_HOST_PASSWORD` are not set, the dashboard works as before with no login required.
- **No signup/registration** -- the only valid credentials are the ones in the env vars.
- **JWT in HTTP-only cookie** -- on successful login, a 24-hour JWT is stored in an `inngest_session` cookie (HttpOnly, SameSite=Lax).
- **SDK endpoints are unaffected** -- event ingestion (`/e/{key}`), function registration (`/fn/register`), and other SDK endpoints continue to use signing key auth, independent of host auth.

## Configuration

| Environment Variable | Required | Description |
|---|---|---|
| `INNGEST_HOST_EMAIL` | Both must be set to enable auth | Email address for login |
| `INNGEST_HOST_PASSWORD` | Both must be set to enable auth | Password for login |

- Email comparison is **case-insensitive** (`Admin@Co.com` matches `admin@co.com`)
- Password comparison is **case-sensitive** and uses constant-time comparison
- Sessions expire after **24 hours** -- the user must re-login

## Docker / Docker Compose

```yaml
services:
  inngest:
    image: inngest/inngest:latest
    ports:
      - "8288:8288"
    environment:
      - INNGEST_HOST_EMAIL=admin@yourcompany.com
      - INNGEST_HOST_PASSWORD=your-secure-password
    command: >
      inngest start
      --signing-key=<your-signing-key>
      --event-key=<your-event-key>
      --sqlite-dir=/data/inngest
    volumes:
      - inngest-data:/data/inngest

volumes:
  inngest-data:
```

## API Endpoints

These endpoints are added when host auth is enabled:

| Method | Path | Auth Required | Description |
|---|---|---|---|
| `GET` | `/auth/status` | No | Returns `{authRequired, authenticated}` |
| `POST` | `/auth/login` | No | Accepts `{email, password}`, sets session cookie |
| `POST` | `/auth/logout` | No | Clears session cookie |

### Example: Check auth status

```bash
curl http://localhost:8288/auth/status
# Auth disabled: {"authRequired":false,"authenticated":false}
# Auth enabled, not logged in: {"authRequired":true,"authenticated":false}
# Auth enabled, logged in: {"authRequired":true,"authenticated":true}
```

### Example: Login via API

```bash
curl -c cookies.txt -X POST http://localhost:8288/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@yourcompany.com","password":"your-secure-password"}'

# Use the cookie for authenticated requests
curl -b cookies.txt http://localhost:8288/v0/gql \
  -H "Content-Type: application/json" \
  -d '{"query":"{ apps { id name } }"}'
```

## What Is Protected

| Resource | Protected by Host Auth? | Notes |
|---|---|---|
| Dashboard UI | Yes | Redirects to `/login` |
| GraphQL API (`/v0/gql`) | Yes | Returns 401 without cookie |
| Event ingestion (`/e/{key}`) | No | Uses signing key auth |
| Function registration (`/fn/register`) | No | Uses signing key auth |
| OTEL traces (`/dev/traces`) | No | SDK endpoint |
| Static assets (`/assets/*`) | No | Needed for login page |

## Security Notes

- The JWT signing secret is derived from the password using HMAC-SHA256. Changing the password invalidates all existing sessions.
- Cookies are set with `HttpOnly` and `SameSite=Lax` flags.
- For production deployments behind HTTPS, consider using a reverse proxy (nginx, Traefik, Caddy) that terminates TLS.
- The password is stored in an environment variable -- use your platform's secrets management (Docker secrets, Kubernetes secrets, etc.) rather than hardcoding it.
