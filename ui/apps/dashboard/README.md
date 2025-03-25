# Inngest Dashboard

This is the web application for Inngest Cloud. It’s a Next.js app that uses GraphQL to communicate
with the [Inngest Cloud "App API"](https://github.com/inngest/monorepo). The app is hosted on Vercel and is the
primary way to interact with Inngest Cloud.

## Setup

Before being able to run the app for the first time, you need to follow the steps below:

### Prerequisites

- Set up the [Cloud monorepo](https://github.com/inngest/monorepo) and have all backend services running locally
- [Node.js 18](https://nodejs.org/en/download/)
- Join the company's Vercel account

### Instructions

1. Clone this repository
2. Install [`pnpm`](https://pnpm.io/) with
   [Corepack](https://nodejs.org/docs/latest-v18.x/api/corepack.html) by running
   `corepack enable; corepack prepare`
3. Install dependencies by running `pnpm install`
4. Link local project to our `ui` Vercel project by running `pnpm vercel link -p ui --yes`
5. Download environment variables by running `pnpm env:pull`

## Developing

### Running the App

#### Development Mode

To start the app in development mode, run the following command:

```sh
$ pnpm dev
```

This will start a local server that will automatically rebuild the app and refresh the page when you
make changes to the code. The app will be available at
[http://localhost:3000](http://localhost:3000).

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

### Environment Variables

Environment variables are managed with the [Vercel CLI](https://vercel.com/docs/cli/env). Use the
following commands to manage them:

```sh
# Link the project on Vercel
$ pnpm vercel link -p ui # and follow the steps

# Download development environment variables for running the app locally
$ pnpm env:pull

# Add a new environment variable
$ pnpm env:add

# Remove an environment variable
$ pnpm env:rm
```

Check the [Vercel documentation](https://vercel.com/docs/concepts/projects/environment-variables)
for more information.

You should **never commit environment variables** to the repository. If you need to add a new
environment variable, add it with the `pnpm env:add` command and then download it with the
`pnpm env:pull` command.

### Sign In

Once you've run `make test-events` in the [Backend Monorepo](https://github.com/inngest/monorepo),
you can sign in using these credentials:

- Username: `test@example.com`
- Password: `testing123`

## Style Guide

### Naming Conventions

- ID abbreviations should follow our Backend conventions.

```javascript
// Good
const environmentID = '';

// Bad
const environmentId = '';
```

- Naming (for both copy and code) should follow our [Product nomenclature](https://www.notion.so/inngest/Nomenclature-Taxonomy-aba427349a724183b90784f0b80d5a35).

```javascript
// Good - terminology we use for external comms
const environment = '';

// Bad - deprecated terminology
const workspace = '';
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
