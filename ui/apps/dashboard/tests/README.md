# Playwright Testing - Inngest Dashboard

> **📖 See the [main testing guide](../../../TESTING.md) for comprehensive documentation.**

This directory contains E2E tests for the Inngest Dashboard.

## Quick Start

### Running Tests Against Development Server

```bash
# Terminal 1: Start the dev server
cd ui/apps/dashboard
pnpm dev

# Terminal 2: Run tests
cd ui/apps/dashboard
pnpm test:e2e
```

### Running Tests Against Production Build (Recommended)

```bash
# Terminal 1: Build and start production server
cd ui/apps/dashboard
pnpm start:build:e2e

# Terminal 2: Run tests
cd ui/apps/dashboard
pnpm test:e2e
```

**Why use the production build?**

- **Faster:** Production builds respond much quicker than dev servers, making tests faster and more reliable
- **Less flaky:** The optimized build keeps up better with Playwright's automation speed
- **Production-like:** Tests run against code similar to what users will experience
- **CI/CD ready:** Same approach used in continuous integration

> **Tip:** Use `pnpm dev` when writing new tests (hot-reload is helpful), then switch to `pnpm start:build:e2e` for running the full test suite.

### Other Test Commands

```bash
# Run tests with visible browser
pnpm test:e2e:headed

# Run tests in debug mode
pnpm test:e2e:debug

# Open Playwright UI
pnpm test:e2e:ui

# Generate new tests
pnpm test:e2e:codegen http://localhost:5173
```

## Dashboard-Specific Details

### Port & URL

- **Development:** `http://localhost:5173` (via `pnpm dev`)
- **E2E Testing:** `http://localhost:5173` (via `pnpm start:build:e2e`)
- **Base URL configured in:** `playwright.config.ts`

### Authentication

The Dashboard uses Clerk for authentication. Tests require authentication setup:

1. Create `playwright/.auth/` directory
2. Create a setup test that saves auth state
3. Configure projects to use the saved auth state

See the [Authentication Setup section](../../../TESTING.md#authentication-setup) in the main guide for detailed instructions.

### Test Structure

```
tests/
├── README.md              # This file
├── example.spec.ts        # Basic configuration tests
├── auth/
│   ├── auth.setup.ts     # Authentication setup
│   └── sign-in.spec.ts   # Auth flow tests
├── functions/
│   ├── list.spec.ts
│   ├── details.spec.ts
│   └── actions.spec.ts
├── environments/
│   └── switching.spec.ts
└── utils/
    └── test-helpers.ts
```

### Environment Variables

Create `.env.test.local`:

```bash
# Clerk testing keys
CLERK_PUBLISHABLE_KEY=your_test_key
CLERK_SECRET_KEY=your_test_secret

# Test user credentials
TEST_USER_EMAIL=test@example.com
TEST_USER_PASSWORD=test-password

# Base URL
PLAYWRIGHT_BASE_URL=http://localhost:3000
```

## Common Dashboard Test Patterns

### Testing Environment Switching

```typescript
test('should switch between environments', async ({ page }) => {
  await page.goto('/env/production');

  await page.click('[data-testid="env-switcher"]');
  await page.click('text=Development');

  await expect(page).toHaveURL(/env\/development/);
});
```

### Testing Function Management

```typescript
test('should display function list', async ({ page }) => {
  await page.goto('/env/production/functions');

  await expect(page.locator('[data-testid="functions-table"]')).toBeVisible();
  await expect(page.locator('text=my-function-name')).toBeVisible();
});
```

## Available Commands

### Server Commands

| Command                | Description                                                   |
| ---------------------- | ------------------------------------------------------------- |
| `pnpm dev`             | Start development server with hot-reload                      |
| `pnpm start:e2e`       | Start production server on port 5173 (requires prior build)   |
| `pnpm start:build:e2e` | Build and start production server (recommended for test runs) |

### Test Commands

| Command                 | Description                                |
| ----------------------- | ------------------------------------------ |
| `pnpm test:e2e`         | Run all E2E tests in headless mode         |
| `pnpm test:e2e:headed`  | Run E2E tests with visible browsers        |
| `pnpm test:e2e:debug`   | Run E2E tests in debug mode                |
| `pnpm test:e2e:ui`      | Open Playwright UI for interactive testing |
| `pnpm test:e2e:codegen` | Launch code generator                      |
| `pnpm test:e2e:report`  | View HTML test report                      |

### Codegen Options

Generate tests with Codegen starting at a specific URL:

```bash
# Start at localhost:5173
npx playwright codegen http://localhost:5173

# Start at a specific page
npx playwright codegen http://localhost:5173/env/production/insights

# Save authenticated state for reuse
npx playwright codegen --save-storage=auth.json http://localhost:5173

# Load saved authentication state
npx playwright codegen --load-storage=auth.json http://localhost:5173
```

For more detailed information, see the [main testing guide](../../../TESTING.md).
