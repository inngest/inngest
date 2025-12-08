# Inngest Support Portal

## Overview

The Inngest Support Portal is a customer support application built with TanStack Start (SSR React framework) that integrates with Plain (customer support platform) and the Inngest GraphQL API. It provides a unified interface for Inngest customers to view and manage their support tickets.

## Tech Stack

- **Framework**: TanStack Start v1.134.6 (React SSR framework)
- **Router**: TanStack Router v1.134.4 with SSR Query integration
- **State Management**: TanStack Query (React Query) v5.66.5
- **Authentication**: Clerk (`@clerk/tanstack-react-start`)
- **Styling**: Tailwind CSS with custom Inngest component library
- **Theme**: `next-themes` for dark/light mode
- **Server**: Nitro v3.0.1-alpha.1
- **Build Tool**: Vite v7.1.10
- **Package Manager**: pnpm v10.18.2

## Project Structure

```
ui/apps/support/
├── src/
│   ├── components/          # React components
│   │   ├── Clerk/          # Clerk authentication provider
│   │   ├── Error/          # Error boundaries and 404 pages
│   │   ├── Layout/         # Layout and sidebar components
│   │   ├── Navigation/     # Navigation, profile, and logo components
│   │   ├── SignIn/         # Sign-in views and error handling
│   │   └── Support/        # Support-specific components (status, etc.)
│   ├── data/               # Server functions and data fetching
│   │   ├── clerk.ts        # Clerk authentication helpers
│   │   ├── envs.ts         # Environment fetching (Inngest API)
│   │   ├── gqlApi.ts       # Inngest GraphQL API client
│   │   ├── nav.ts          # Navigation state
│   │   ├── plain.ts        # Plain API integration (tickets)
│   │   └── profile.ts      # User profile data
│   ├── gql/                # Generated GraphQL types (Inngest API)
│   ├── routes/             # TanStack Router routes
│   │   ├── __root.tsx      # Root layout with providers
│   │   ├── _authed.tsx     # Authenticated layout wrapper
│   │   ├── _authed/
│   │   │   ├── support/    # Support ticket listing
│   │   │   └── case.$ticketId.tsx  # Individual ticket detail
│   │   ├── index.tsx       # Landing/redirect page
│   │   ├── sign-in.$.tsx   # Sign-in page
│   │   └── sign-out.tsx    # Sign-out handler
│   ├── utils/              # Utility functions
│   ├── router.tsx          # Router configuration
│   ├── routeTree.gen.ts    # Generated route tree
│   └── start.ts            # TanStack Start entry point
├── vite.config.ts          # Vite configuration
├── package.json
└── CLAUDE.md              # This file
```

## Key Files

### Entry Points

- **src/start.ts** - TanStack Start server entry point with Clerk middleware
- **src/router.tsx** - Router configuration with QueryClient and SSR integration
- **src/routes/\_\_root.tsx** - Root component with global providers (Clerk, Theme, Tooltip)

### Authentication & API Integration

- **src/data/clerk.ts** - Clerk authentication helpers for server functions
- **src/data/gqlApi.ts** - Inngest GraphQL API client with Clerk token auth
- **src/data/plain.ts** - Plain API integration for support tickets
  - `getTicketsByEmail()` - Fetch all tickets for a customer by email
  - `getTicketById()` - Fetch detailed ticket information
  - `getTimelineEntriesForTicket()` - Fetch ticket timeline/comments

### Route Configuration

- **src/routes/\_authed.tsx** - Protected route wrapper that:
  - Validates Clerk authentication
  - Loads user profile from Inngest API
  - Provides Layout with sidebar and navigation
  - Handles "Inngest is down" errors

### Components

- **src/components/Clerk/Provider.tsx** - Styled Clerk provider matching Inngest design system
- **src/components/Layout/Layout.tsx** - Main layout with sidebar and content area
- **src/components/Navigation/** - Header, profile menu, logo, and status indicators

## External APIs

### Plain API (Customer Support Platform)

The Plain GraphQL schema is available at:

```
https://core-api.uk.plain.com/graphql/v1/schema.graphql
```

**Integration**: Uses `@team-plain/typescript-sdk` v3.0.1

**Key Operations**:

- Get customer by email
- Fetch threads (tickets) for a customer
- Get thread details
- Fetch timeline entries (comments/activities)

**Authentication**: Via `PLAIN_API_KEY` environment variable

### Inngest GraphQL API

**Endpoint**: `${VITE_API_URL}/gql`

**Authentication**: Clerk Bearer token via `Authorization` header

**Used For**:

- User profile data
- Environment information
- Account/workspace data

## Environment Variables

Required environment variables:

- `PLAIN_API_KEY` - Plain API authentication key
- `VITE_API_URL` - Inngest API base URL
- Clerk environment variables (auto-configured by `@clerk/tanstack-react-start`)

## Development

```bash
# Start dev server on port 3002
pnpm dev

# Build for production
pnpm build

# Run production server
pnpm start

# Lint
pnpm lint

# Format
pnpm format
```

## Common Issues

### Hydration Warning

The app uses `suppressHydrationWarning` on the `<html>` tag in `__root.tsx:85` to suppress expected hydration mismatches from `next-themes` (which reads from localStorage/system preferences on the client).

### Import Paths

- `@/` - Alias for `src/`
- `@inngest/components` - Alias for shared component library at `../../packages/components/src`

## Architecture Notes

### Server Functions

The app uses TanStack Start's `createServerFn()` for server-side data fetching:

- All functions in `src/data/` are server functions
- They run on the server and are called from client components
- Auth tokens are handled server-side for security

### SSR Query Integration

Uses `@tanstack/react-router-ssr-query` for:

- Deduplicating server and client queries
- Preloading data during SSR
- Seamless hydration of query data

### Route-based Code Splitting

TanStack Router automatically code-splits by route, reducing initial bundle size.

### Authentication Flow

1. User visits protected route
2. `_authed.tsx` `beforeLoad` hook checks Clerk auth
3. If authenticated, `loader` fetches profile from Inngest API
4. If not authenticated, redirects to sign-in
5. Clerk session persists across requests

## TODO/Future Work

Comments in the codebase indicate:

- Add page view tracking (see `__root.tsx:76`)
- Handle "Inngest is down" error specifically (see `_authed.tsx:40`)
- Improve typing for Plain API responses (see `plain.ts:169`)
