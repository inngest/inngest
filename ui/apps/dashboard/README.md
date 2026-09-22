# Inngest Dashboard

This is the web application for Inngest Cloud. It’s a Tanstack Start app that uses GraphQL to communicate
with the [Inngest Cloud "App API"](https://github.com/inngest/monorepo). The app is hosted on Vercel and is the
primary way to interact with Inngest Cloud.

## Setup

Before being able to run the app for the first time, you need to follow the steps below:

### Prerequisites

- Set up the [Cloud monorepo](https://github.com/inngest/monorepo) and have all backend services running locally
- [Node.js](https://nodejs.org/en/download/)
- Install 1Password Desktop App

### Instructions

1. Clone this repository
2. Install [`pnpm`](https://pnpm.io/) with
   [Corepack](https://nodejs.org/docs/latest-v18.x/api/corepack.html) by running
   `corepack enable; corepack prepare`
3. Install dependencies by running `pnpm install`
4. Setup local environment variables following [this guide](https://www.notion.so/inngest/Setup-env-local-for-development-351b64753bbd8068bceed9825558ec33?source=copy_link)

## Developing

### Running the App

#### Development Mode

To start the app in development mode, run the following command:

```sh
$ pnpm dev
```

This will start a local server that will automatically rebuild the app and refresh the page when you
make changes to the code. The app will be available at
[http://localhost:5173](http://localhost:5173).

This is how you will run the app most of the time.

#### Production Mode

To run the app in production mode, run the following commands in order:

```sh
# Build the app for production usage
$ pnpm build

# Start the app in production mode
$ pnpm start
```

This can be useful for testing the app in production mode locally.

### Code Linting

Code linting is handled by [ESLint](https://eslint.org/). You can use the following command for
linting all project's files:

```sh
$ pnpm lint
```

Staged files are automatically linted before commits. Be sure to **fix all linting errors before
committing**.

We recommend using an [editor integration for ESLint](https://eslint.org/docs/user-guide/integrations).

### Code Formatting

Code formatting is handled by [Prettier](https://prettier.io/). You can use the following command to
format all project’s files:

```sh
$ pnpm format
```

Staged files are automatically formatted when committing.

We recommend using an [editor integration for Prettier](https://prettier.io/docs/en/editors.html).

### Sign In

Once you've run `make test-events` in the [Backend Monorepo](https://github.com/inngest/monorepo),
you can sign in using these credentials:

- Username: `test@example.com`
- Password: `testing123`

### Sandbox Secrets

The `/env/$envSlug/sandboxes` page uses the shared right helper panel for secret
management, behind the existing `sandbox_api` flag. Organization admins can list,
create, replace and delete secrets for the selected environment. This requires
the Cloud API's `envSecrets`, `createEnvSecret`, `updateEnvSecretValue` and
`archiveEnvSecret` GraphQL fields and configured secret storage. The browser
uses the existing authenticated GraphQL client; it never accesses KMS directly.

Paste `.env` contents into a name field to populate editable rows, then explicitly
save. Imports support comments, `export`, quoted and multiline values, and empty
values. Double-quoted `\n` and `\r` are expanded; variables and commands are not.
Invalid lines reject the entire paste. Duplicate and already-saved names are
flagged; imports never replace an existing value. Replacement is a separate
action. Import limits are 256 entries / 1 MiB, with the backend's 256-byte name
and 64-KiB value limits enforced per entry.

Names match exactly and become sandbox environment variable names:
`secrets: ["OPENAI_API_KEY"]`. Saved values cannot be retrieved by this UI.
Replacement affects future retrievals; running sandboxes and existing snapshots
retain values already received. Deleting prevents future retrievals of that
secret identity.

Imports save sequentially and stop on the first failure. Successful rows are
cleared, leaving only unsaved or unconfirmed rows for review. A failed network
response can leave the last save's result unknown; check the list before trying
again. Closing a dirty editor prompts before discarding. Changing environments
or navigating away clears the draft; drafts are not persisted.

Focused tests: `pnpm test src/components/Sandboxes/Secrets --maxWorkers=1`.

## Style Guide

### Naming Conventions

- ID abbreviations should follow our Backend conventions.

```javascript
// Good
const environmentID = "";

// Bad
const environmentId = "";
```

- Naming (for both copy and code) should follow our [Product nomenclature](https://www.notion.so/inngest/Nomenclature-Taxonomy-aba427349a724183b90784f0b80d5a35).

```javascript
// Good - terminology we use for external comms
const environment = "";

// Bad - deprecated terminology
const workspace = "";
```

### Language and Copy Conventions

- The default language of the project is English(US).

```javascript
// Good
function analyzeStats() {
  console.log(foo);
}

// Bad
function analyseStats() {
  console.log(foo);
}
```

- Use Title Case when adding copy of headings, titles, tags, navbar items and buttons.

```javascript
// Good
<button>Click Me</button>;

// Bad
<button>Click me</button>;

// Bad
<button>click me</button>;
```
