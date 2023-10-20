# Contributing

## Organization

This package should be relatively flat to discourage consumers from importing internals

### Components

Exported components are not nested deeper than `src/<Component>`. Use a `src/<Component>/index.ts` file to communicate which things are meant to be exported for each component. For example, the `Button` component would go in `src/Button/Button.tsx`, and would be reexported in `src/Button/index.ts`.

### Icons

While icons are technically components, they are nested within `src/icons`. This is because we want to avoid cluttering `src/` with a bunch of icons folders. There isn't a `src/icons/index.ts` file since that might cause larger bundle sizes for consumers.

### Utils

Utility functions are in `src/utils/`. There isn't a `src/utils/index.ts` file since that might cause larger bundle sizes for consumers. We may want to move utils into a separate package in the future, since they aren't components.
