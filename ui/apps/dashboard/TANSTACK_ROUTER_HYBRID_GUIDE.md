# TanStack Router Hybrid Implementation - Complete Guide

This guide provides a comprehensive overview of implementing TanStack Router within a Next.js application, including all gotchas, solutions, and best practices discovered during development.

## 🏗️ Architecture Overview

- **Next.js** handles server-side routing and redirects
- **TanStack Router** handles client-side routing within `/tanstack`
- **File-based routing** with auto-discovery and hot reload
- **Type-safe navigation** with auto-generated route tree
- **Hybrid approach** allows incremental adoption

## 📁 Project Structure

```
src/
├── spa/                          # TanStack Router SPA
│   ├── routes/                   # File-based routes (watched by CLI)
│   │   ├── __root.tsx           # Root layout with providers
│   │   ├── index.tsx            # Home page (/)
│   │   ├── about.tsx            # About page (/about)
│   │   ├── env.$envSlug.tsx     # Environment layout route
│   │   └── env.$envSlug.functions.tsx # Functions page
│   ├── contexts/                # React contexts for data sharing
│   │   └── EnvironmentContext.tsx
│   ├── components/              # Shared components
│   ├── app.tsx                  # Main SPA entry point
│   └── routeTree.gen.ts         # 🤖 Auto-generated (DON'T EDIT)
├── app/
│   └── tanstack/
│       ├── page.tsx             # Next.js page that loads SPA
│       └── TanStackRouterApp.tsx # SPA wrapper
└── components/                   # Next.js components
```

## 🚀 Quick Start

### Installation & Setup

```bash
# Install dependencies
pnpm install

# Generate initial route tree
pnpm router:generate

# Start development server
pnpm dev
```

### Available Routes

- `/tanstack/` - Home page with navigation
- `/tanstack/about` - About page
- `/tanstack/env/production` - Environment page
- `/tanstack/env/production/functions` - Functions page

## ⚠️ Critical Gotchas & Solutions

Based on real implementation experience, here are the **most important gotchas** to avoid:

### 1. URQL Client Configuration 🔴

**Issue**: Missing exchanges array causes runtime errors
**Error**: `TypeError: Cannot read properties of undefined (reading 'reduceRight')`

```tsx
// ❌ WRONG - missing exchanges
const client = createClient({
  url: '/gql',
  // Missing exchanges array!
});

// ✅ CORRECT - always provide exchanges
const client = createClient({
  url: '/gql',
  exchanges: [cacheExchange, fetchExchange], // Required!
});
```

### 2. Data Sharing Between Routes 🟡

**Issue**: `Outlet context` can be unreliable for passing data
**Error**: Environment context not available in child routes

```tsx
// ❌ UNRELIABLE - Outlet context
<Outlet context={{ environment: env }} />

// ✅ RELIABLE - React Context
// Parent route
<EnvironmentProvider environment={env}>
  <Outlet />
</EnvironmentProvider>;

// Child route
const { environment } = useEnvironmentContext();
```

### 3. Next.js Rewrites Configuration 🟡

**Issue**: TanStack Router sub-routes return 404 in production
**Error**: 404 - Page Not Found for `/tanstack/env/production`

```javascript
// next.config.js
module.exports = {
  async rewrites() {
    return [
      {
        source: '/tanstack/:path*',
        destination: '/tanstack', // Serve SPA for all sub-routes
      },
    ];
  },
};
```

### 4. Build Process & Route Generation 🟡

**Issue**: Routes not generated during production build
**Error**: `tsr generate` command not found

```json
// package.json
{
  "scripts": {
    "build": "pnpm router:generate && next build",
    "router:generate": "tsr generate",
    "router:watch": "tsr watch"
  },
  "devDependencies": {
    "@tanstack/router-cli": "latest"
  }
}
```

### 5. TypeScript Integration Limitations 🟡

**Issue**: CLI doesn't provide full TypeScript integration
**Solution**: Use `getRouteApi` pattern with workarounds

```tsx
// ✅ Use getRouteApi pattern
import { getRouteApi } from '@tanstack/react-router';

const routeApi = getRouteApi('/env/$envSlug' as any);

function Component() {
  const { envSlug } = (routeApi as any).useParams();
  const search = (routeApi as any).useSearch();

  // Navigation with type assertions
  const navigate = useNavigate();
  navigate({
    to: '/env/$envSlug/functions' as any,
    params: { envSlug } as any,
  });
}
```

## 🛠️ Configuration Files

### 1. TanStack Router CLI Config (`tsr.config.json`)

```json
{
  "routesDirectory": "./src/spa/routes",
  "generatedRouteTree": "./src/spa/routeTree.gen.ts",
  "routeFileIgnorePrefix": "-",
  "quoteStyle": "single",
  "semicolons": true,
  "disableTypes": false,
  "addExtensions": false,
  "disableLogging": false,
  "disableManifestGeneration": false,
  "apiBase": "/api",
  "routeTreeFileHeader": [
    "/* prettier-ignore-start */",
    "/* eslint-disable */",
    "// @ts-nocheck",
    "// noinspection JSUnusedGlobalSymbols"
  ],
  "routeTreeFileFooter": ["/* prettier-ignore-end */"]
}
```

### 2. Next.js Configuration (`next.config.js`)

```javascript
/** @type {import('next').NextConfig} */
const nextConfig = {
  skipTrailingSlashRedirect: true,

  async redirects() {
    return [
      {
        source: '/',
        destination: '/env/production/apps',
        permanent: false,
      },
    ];
  },

  async rewrites() {
    return [
      {
        source: '/tanstack/:path*',
        destination: '/tanstack',
      },
    ];
  },
};

module.exports = nextConfig;
```

### 3. Git Ignore (`.gitignore`)

```gitignore
# TanStack Router Generated Files
src/spa/routeTree.gen.ts
```

## 📝 Creating New Routes

### Basic Route

```tsx
// src/spa/routes/new-page.tsx
import { createFileRoute } from '@tanstack/react-router';

function NewPageComponent() {
  return (
    <div className="p-4">
      <h1>New Page</h1>
    </div>
  );
}

export const Route = createFileRoute('/new-page')({
  component: NewPageComponent,
});
```

### Dynamic Route with Params

```tsx
// src/spa/routes/posts.$postId.tsx
import { createFileRoute, getRouteApi } from '@tanstack/react-router';

const routeApi = getRouteApi('/posts/$postId' as any);

function PostComponent() {
  const { postId } = (routeApi as any).useParams();

  return <h1>Post {postId}</h1>;
}

export const Route = createFileRoute('/posts/$postId')({
  component: PostComponent,
});
```

### Route with Data Loading

```tsx
// src/spa/routes/profile.tsx
import { createFileRoute } from '@tanstack/react-router';
import { useClient } from 'urql';

function ProfileComponent() {
  const client = useClient(); // Authenticated URQL client

  useEffect(() => {
    // Load data using authenticated client
  }, [client]);

  return <div>Profile</div>;
}

export const Route = createFileRoute('/profile')({
  component: ProfileComponent,
});
```

## 🔄 Development Workflow

### Commands

```bash
# Development (runs all watchers)
pnpm dev

# Generate routes manually
pnpm router:generate

# Watch for route changes
pnpm router:watch

# Build for production
pnpm build

# Start production server
pnpm start
```

### Hot Reload Behavior

| Change Type           | Behavior                          |
| --------------------- | --------------------------------- |
| **Component content** | ✅ Hot reload (instant)           |
| **Route config**      | ✅ Hot reload (instant)           |
| **New route file**    | 🔄 Route tree regeneration (1-2s) |
| **Delete route file** | 🔄 Route tree regeneration (1-2s) |

## 🎯 Navigation Patterns

### Programmatic Navigation

```tsx
import { useNavigate } from '@tanstack/react-router';

function NavigationExample() {
  const navigate = useNavigate();

  const handleNavigation = () => {
    navigate({
      to: '/env/$envSlug/functions' as any,
      params: { envSlug: 'production' } as any,
      search: { tab: 'details' } as any,
    });
  };

  return <button onClick={handleNavigation}>Navigate</button>;
}
```

### Link Components

```tsx
import { Link } from '@tanstack/react-router';

function LinkExample() {
  return (
    <Link
      to="/env/$envSlug"
      as
      any
      params={{ envSlug: 'production' } as any}
      className="text-blue-600"
    >
      Environment
    </Link>
  );
}
```

## 🔌 Real Data Integration

### URQL + GraphQL Setup

```tsx
// In __root.tsx
import { ClerkProvider, useAuth } from '@clerk/tanstack-react-start';
import { Provider as URQLProvider, authExchange, createClient } from 'urql';

function RootWithAuth() {
  const { getToken, signOut } = useAuth();

  const urqlClient = useMemo(() => {
    return createClient({
      url: `${process.env.NEXT_PUBLIC_API_URL}/gql`,
      exchanges: [
        authExchange(async (utils) => {
          let sessionToken = await getToken();
          return {
            addAuthToOperation: (operation) => {
              if (!sessionToken) return operation;
              return utils.appendHeaders(operation, {
                Authorization: `Bearer ${sessionToken}`,
              });
            },
            didAuthError: (error) => {
              return error.graphQLErrors.some((e) => e.extensions.code === 'UNAUTHENTICATED');
            },
            refreshAuth: async () => {
              sessionToken = await getToken({ skipCache: true });
            },
          };
        }),
        cacheExchange,
        fetchExchange,
      ],
    });
  }, [getToken, signOut]);

  return (
    <URQLProvider value={urqlClient}>
      <Outlet />
    </URQLProvider>
  );
}
```

### React Context for Data Sharing

```tsx
// src/spa/contexts/EnvironmentContext.tsx
import { createContext, useContext } from 'react';

interface EnvironmentContextType {
  environment: Environment | null;
  loading: boolean;
  error: string | null;
}

const EnvironmentContext = createContext<EnvironmentContextType | undefined>(undefined);

export function EnvironmentProvider({ children, environment, loading, error }) {
  return (
    <EnvironmentContext.Provider value={{ environment, loading, error }}>
      {children}
    </EnvironmentContext.Provider>
  );
}

export function useEnvironmentContext() {
  const context = useContext(EnvironmentContext);
  if (context === undefined) {
    throw new Error('useEnvironmentContext must be used within an EnvironmentProvider');
  }
  return context;
}
```

## 🚀 Production Deployment

### Vercel Deployment

1. **Environment Variables**: Set in Vercel dashboard

   - `NEXT_PUBLIC_API_URL`
   - `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`

2. **Build Process**: Automatically handled

   ```bash
   pnpm router:generate && next build
   ```

3. **Routing**: Next.js rewrites handle SPA sub-routes

## 🐛 Common Debugging Steps

1. **Route not found**: Check file naming and route tree regeneration
2. **Type errors**: Run `pnpm router:generate` and restart TypeScript
3. **Auth errors**: Verify provider hierarchy in `__root.tsx`
4. **URQL errors**: Check exchanges array in client configuration
5. **Build errors**: Ensure route generation runs before Next.js build

## 🎯 What Works Despite CLI Limitations

✅ **File-based routing** - Auto-discovery from file structure  
✅ **Hot reload** - Route changes update automatically  
✅ **Data loading** - Component-level URQL integration  
✅ **Search params** - Type-safe with workarounds  
✅ **Dynamic routes** - Parameterized routes work perfectly  
✅ **Navigation** - Programmatic navigation with type assertions  
✅ **DevTools** - TanStack Router DevTools in development  
✅ **Authentication** - Full Clerk integration  
✅ **Real data** - URQL GraphQL with auth

## 🔧 What Requires Workarounds

⚠️ **Type safety** - Requires `as any` assertions due to CLI limitations  
⚠️ **Hook usage** - Must use `getRouteApi` pattern for reliability  
⚠️ **Link types** - Need explicit type assertions  
⚠️ **Data sharing** - Use React Context instead of `Outlet context`  
⚠️ **Provider hierarchy** - Careful ordering required for auth

---
