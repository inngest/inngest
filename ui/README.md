# UI Monorepo

This repository contains the frontend applications (Cloud and Dev Server) and shared UI components of Inngest. For the marketing website and our docs go to [website](https://github.com/inngest/website) instead.

## Folder structure

```
/ui
  ├── apps/
    ├── dashboard/      // Inngest Cloud
    ├── dev-server-ui/  // Inngest Dev Server
  ├── packages/
    ├── components/     // Shared UI components
```

## Tech stack

Typescript + Tailwind CSS. Next.js apps that use GraphQL to communicate with the API.

## Installation & setup

- Inngest Cloud [instructions](https://github.com/inngest/inngest/tree/main/ui/apps/dashboard#setup)
- Inngest Dev Server [instructions](https://github.com/inngest/inngest/tree/main/ui/apps/dev-server-ui#development)
- Shared components: run storybook to check them

## Code linting and formatting

Code linting is handled by ESLint and code formatting is handled by Prettier.

## Color token system and dark mode

Our light and dark themes are automatically handled by using a color token system.
The color tokens are defined in a [CSS file](https://github.com/inngest/inngest/blob/main/ui/packages/components/src/AppRoot/globals.css#L72-L371) using CSS variables.
We integrate the tokens into the shared [Tailwind configuration](https://github.com/inngest/inngest/blob/main/ui/packages/components/tailwind.config.ts). All our colors must be defined using the color tokens.

```javascript
// Good
<div className="bg-canvasBase">

// Bad
<div className="bg-white">
```

## CSS best practices

The default approach is to use Tailwind CSS classes.
Exceptions that justify the usage of inline styles or CSS in global.css are when Tailwind doesn't provide a solution or when injecting styles on 3rd parties.

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

- The default language of the project is English(US). Exception made for the word "Cancelled" for legacy purposes.

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

- Use "Sentence case" when adding copy of headings, titles, tags, navbar items and buttons.

```javascript
// Good
<button>Click me</button>;

// Bad
<button>Click Me</button>;
```

## Security

### PNPM

This project uses `pnpm`. Additionally, `.npmrc` defines the [`minimum-release-age`](https://pnpm.io/settings#minimumreleaseage) setting to limit how recently an installed package is allowed to have been published. In case this must be overriden for a particular package, [`minimum-release-age-exclude`](https://pnpm.io/settings#minimumreleaseageexclude) can be used.
