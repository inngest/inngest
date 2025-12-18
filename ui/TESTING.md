# Playwright Testing Guide - Inngest UI Apps

This guide covers end-to-end testing for all Inngest UI applications using Playwright.

## Applications Covered

- **Dashboard** (`ui/apps/dashboard`) - Main Inngest dashboard on port 5173
- **Support Portal** (`ui/apps/support`) - Customer support portal on port 3002
- **Dev Server UI** (`ui/apps/dev-server-ui`) - Development server interface on port 5173

## Table of Contents

- [Quick Start](#quick-start)
- [Available Commands](#available-commands)
- [Writing Tests](#writing-tests)
- [Code Generation](#code-generation)
- [Test Organization](#test-organization)
- [Authentication Setup](#authentication-setup)
- [Configuration](#configuration)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Quick Start

### Prerequisites

**Important:** The application server must be running before you can run tests. You can use either the development server or a production build.

#### Option 1: Development Server (Slower)

```bash
# In one terminal, start the dev server for your app
cd ui/apps/{dashboard|support|dev-server-ui}
pnpm dev

# Dashboard runs on: http://localhost:5173
# Support runs on:    http://localhost:3002
# Dev Server UI runs on: http://localhost:5173
```

#### Option 2: Production Build (Recommended for E2E)

```bash
# In one terminal, start the built app for your app
cd ui/apps/{dashboard|support|dev-server-ui}
pnpm start:build:e2e

# Dashboard runs on: http://localhost:5173
# Support runs on:    http://localhost:3002
# Dev Server UI runs on: http://localhost:5173
```

**Why use the production build for testing?**

- **Faster response times:** Production builds are optimized and respond much quicker than dev servers
- **More reliable:** Tests can keep up with the app better, reducing flakiness
- **Closer to production:** Tests run against code that's similar to what users will see
- **Better for CI/CD:** Faster test execution in continuous integration

> **Note:** Use `pnpm dev` for writing and debugging tests since it has hot-reload. Switch to `pnpm start:build:e2e` for running full test suites.

### Running Tests

**Important:** All test commands must be run from the specific app directory, and the dev server must be running.

```bash
# In a second terminal, navigate to your app directory and run tests
cd ui/apps/{dashboard|support|dev-server-ui}
pnpm test:e2e

# Examples:
# For Dashboard (ensure pnpm dev is running on port 5173)
cd ui/apps/dashboard
pnpm test:e2e

# For Support Portal (ensure pnpm dev is running on port 3002)
cd ui/apps/support
pnpm test:e2e

# For Dev Server UI (ensure pnpm dev is running on port 5173)
cd ui/apps/dev-server-ui
pnpm test:e2e

# Dev Server UI also has unit tests (no dev server needed)
cd ui/apps/dev-server-ui
pnpm test  # Runs vitest
```

### Full Test Workflow

#### Using Development Server (for writing tests)

```bash
# Terminal 1: Start the dev server
cd ui/apps/{dashboard|support|dev-server-ui}
pnpm dev

# Terminal 2: Run tests
cd ui/apps/{dashboard|support|dev-server-ui}
pnpm test:e2e
```

#### Using Production Build (recommended for test runs)

```bash
# Terminal 1: Build and start the production server
cd ui/apps/{dashboard|support|dev-server-ui}
pnpm start:build:e2e

# Terminal 2: Run tests
cd ui/apps/{dashboard|support|dev-server-ui}
pnpm test:e2e
```

#### Available Test Commands

```bash
# Run E2E tests (headless)
pnpm test:e2e

# Run E2E tests with visible browser
pnpm test:e2e:headed

# Run E2E tests in debug mode (inspector)
pnpm test:e2e:debug

# Run E2E tests in interactive UI mode
pnpm test:e2e:ui

# View the latest test report
pnpm test:e2e:report
```

### Generating Tests with Code Generation

```bash
# Navigate to your app directory
cd ui/apps/{dashboard|support|dev-server-ui}

# Start the dev server in one terminal
pnpm dev

# In another terminal, generate tests by interacting with the app
pnpm test:e2e:codegen <url>

# Examples for each app:
# Dashboard:
pnpm test:e2e:codegen http://localhost:3000

# Support:
pnpm test:e2e:codegen http://localhost:3002

# Dev Server UI:
pnpm test:e2e:codegen http://localhost:5173
```

## Available Commands

All commands must be run from the app directory (`ui/apps/{dashboard|support|dev-server-ui}`).

### Server Commands

| Command                | Description                                                      |
| ---------------------- | ---------------------------------------------------------------- |
| `pnpm dev`             | Start development server with hot-reload (for writing tests)     |
| `pnpm start:e2e`       | Start production server on E2E test port (requires prior build)  |
| `pnpm start:build:e2e` | Build and start production server on E2E test port (recommended) |

### Test Commands

| Command                 | Description                                           |
| ----------------------- | ----------------------------------------------------- |
| `pnpm test:e2e`         | Run all E2E tests in headless mode                    |
| `pnpm test:e2e:headed`  | Run E2E tests with visible browsers                   |
| `pnpm test:e2e:debug`   | Run E2E tests in debug mode with Playwright Inspector |
| `pnpm test:e2e:ui`      | Open Playwright UI for interactive test running       |
| `pnpm test:e2e:codegen` | Launch code generator for recording interactions      |
| `pnpm test:e2e:report`  | View HTML test report                                 |

**Dev Server UI only:**
| Command | Description |
|---------|-------------|
| `pnpm test` | Run unit tests with Vitest |

## Writing Tests

### Basic Test Structure

```typescript
import { expect, test } from '@playwright/test';

test.describe('Feature Name', () => {
  test('should perform specific action', async ({ page }) => {
    await page.goto('/your-route');

    // Wait for elements to be visible
    await expect(page.locator('[data-testid="main-content"]')).toBeVisible();

    // Interact with elements
    await page.click('button[data-testid="submit-button"]');

    // Assert results
    await expect(page.locator('.success-message')).toContainText('Success');
  });
});
```

**Note:** Examples use specific `data-testid` attributes for clarity. In practice, use flexible selectors or inspect the actual UI. See existing test files for examples of resilient selector strategies using wildcards like `[data-testid*="trigger"]`.

### App-Specific Testing Patterns

#### Dashboard - Testing with TanStack Router

```typescript
test('should navigate using router', async ({ page }) => {
  await page.goto('/');

  // Navigate to functions page
  await page.click('nav a[href*="/functions"]');

  // Verify route change
  await expect(page).toHaveURL(/functions/);

  // Verify page content loaded
  await expect(page.locator('h1')).toContainText('Functions');
});

test('should switch between environments', async ({ page }) => {
  await page.goto('/env/production');

  // Click environment switcher
  await page.click('[data-testid="env-switcher"]');

  // Select development environment
  await page.click('text=Development');

  // Verify URL changed
  await expect(page).toHaveURL(/env\/development/);
});
```

#### Support Portal - Testing with Clerk Auth

```typescript
test('should handle Clerk authentication flow', async ({ page }) => {
  await page.goto('/support');

  // Should redirect to Clerk sign-in if not authenticated
  await expect(page).toHaveURL(/sign-in/);

  // Or verify authenticated state shows support content
  await expect(page.locator('[data-testid="user-profile"]')).toBeVisible();
});

test('should handle Plain API responses', async ({ page }) => {
  // Intercept Plain API calls
  await page.route('**/graphql', (route) => {
    if (route.request().postData()?.includes('getCustomerByEmail')) {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            customer: {
              id: 'test-customer-id',
              email: 'test@example.com',
              tickets: [],
            },
          },
        }),
      });
    } else {
      route.continue();
    }
  });

  await page.goto('/support');

  // Verify the mocked data is used
  await expect(page.locator('[data-testid="customer-email"]')).toContainText('test@example.com');
});
```

#### Dev Server UI - Testing Development Features

```typescript
test('should trigger function in dev environment', async ({ page }) => {
  await page.goto('/functions/my-function');

  // Click trigger button
  await page.click('[data-testid="trigger-function"]');

  // Wait for execution feedback
  await expect(page.locator('[data-testid="execution-result"]')).toBeVisible();

  // Verify logs appear
  await expect(page.locator('[data-testid="function-logs"]')).toContainText('Function started');
});

test('should edit function code in Monaco', async ({ page }) => {
  await page.goto('/functions/my-function/edit');

  // Wait for Monaco editor to fully load
  await expect(page.locator('.monaco-editor')).toBeVisible();
  await page.waitForTimeout(1000); // Monaco needs time to initialize

  // Focus the editor
  await page.click('.monaco-editor .view-lines');

  // Type new function code
  await page.keyboard.type('console.log("test");');

  // Save the function
  await page.click('[data-testid="save-function"]');

  // Verify save
  await expect(page.locator('[data-testid="save-success"]')).toBeVisible();
});
```

## Code Generation

### Recording Workflows

1. **Navigate to your app directory:**

   ```bash
   cd ui/apps/{dashboard|support|dev-server-ui}
   ```

2. **Start your development server:**

   ```bash
   pnpm dev
   ```

3. **Open the code generator (in another terminal):**

   ```bash
   cd ui/apps/{dashboard|support|dev-server-ui}

   # Dashboard
   pnpm test:e2e:codegen http://localhost:5173

   # Support
   pnpm test:e2e:codegen http://localhost:3002

   # Dev Server UI
   pnpm test:e2e:codegen http://localhost:5173
   ```

4. **Interact with your app:**

   - Click buttons, fill forms, navigate pages
   - Playwright records all interactions
   - Use the inspector to add assertions

5. **Copy generated code:**
   - The generated code appears in the Playwright Inspector
   - Copy and paste into your test files
   - Clean up and organize as needed

### Advanced Code Generation

```bash
# Generate tests with specific viewport
pnpm test:e2e:codegen --device="iPhone 12" http://localhost:<port>

# Generate tests with custom browser
pnpm test:e2e:codegen --browser=webkit http://localhost:<port>

# Generate tests with authentication context (Dashboard/Support)
pnpm test:e2e:codegen --load-storage=auth.json http://localhost:<port>

# Save authentication state first
pnpm test:e2e:codegen --save-storage=auth.json http://localhost:<port>
```

## Test Organization

### Recommended File Structure

Each app's `tests/` directory should follow this pattern:

```
tests/
├── README.md              # This file (or app-specific notes)
├── example.spec.ts        # Basic configuration tests
├── auth/                  # Authentication tests (Dashboard/Support)
│   ├── sign-in.spec.ts
│   └── permissions.spec.ts
├── [feature-name]/        # Group tests by feature
│   ├── list.spec.ts
│   ├── details.spec.ts
│   └── actions.spec.ts
└── utils/                 # Shared test utilities
    ├── test-helpers.ts
    └── api-mocks.ts
```

### App-Specific Organization

**Dashboard:**

```
tests/
├── auth/
├── functions/
├── environments/
├── billing/
└── utils/
```

**Support:**

```
tests/
├── auth/
├── tickets/
├── api/
├── responsive/
└── utils/
```

**Dev Server UI:**

```
tests/
├── example.spec.ts
├── functions/
│   └── trigger.spec.ts  # ✅ Exists
└── utils/
```

### Naming Conventions

- Use descriptive file names: `function-management.spec.ts`
- Group related tests in folders by feature area
- Use `describe` blocks for features: "Function Management"
- Use specific test names: "should pause function when pause button clicked"

## Authentication Setup

### Dashboard - Clerk Authentication

The Dashboard uses Clerk for authentication. Tests need to handle auth state:

#### Save Authentication State

```typescript
// In your test or setup file
test('save authentication state', async ({ page }) => {
  // Sign in through your auth flow
  await page.goto('/sign-in');
  await page.fill('[name="email"]', process.env.TEST_USER_EMAIL);
  await page.fill('[name="password"]', process.env.TEST_USER_PASSWORD);
  await page.click('button[type="submit"]');

  // Wait for redirect to confirm sign-in
  await page.waitForURL('/env/**');

  // Save authentication state
  await page.context().storageState({ path: 'playwright/.auth/user.json' });
});
```

#### Use Authentication in Tests

```typescript
// playwright.config.ts
export default defineConfig({
  projects: [
    // Setup project - runs first
    { name: 'setup', testMatch: /.*\.setup\.ts/ },

    // Authenticated tests
    {
      name: 'authenticated',
      use: {
        ...devices['Desktop Chrome'],
        storageState: 'playwright/.auth/user.json',
      },
      dependencies: ['setup'],
    },

    // Unauthenticated tests
    {
      name: 'unauthenticated',
      use: { ...devices['Desktop Chrome'] },
      testMatch: /.*\.unauthenticated\.spec\.ts/,
    },
  ],
});
```

#### Environment Variables

Create a `.env.test.local` file in the app directory:

```bash
# Dashboard
TEST_USER_EMAIL=test@example.com
TEST_USER_PASSWORD=test-password
PLAYWRIGHT_BASE_URL=http://localhost:5173
```

### Support Portal - Clerk Authentication

Similar to Dashboard, the Support Portal uses Clerk. Follow the same authentication setup pattern above.

```bash
# Support
TEST_USER_EMAIL=support-test@example.com
TEST_USER_PASSWORD=test-password
PLAYWRIGHT_BASE_URL=http://localhost:3002
PLAIN_API_KEY=test_plain_key
```

### Dev Server UI - No Authentication

The Dev Server UI doesn't require authentication. However, you may need environment variables:

```bash
# Dev Server UI - .env.test.local
VITE_ENV=test
INNGEST_DEV=true
```

## Configuration

### Current Configuration

All apps use similar Playwright configuration:

- **Browsers:** Chromium, Firefox, WebKit
- **Mobile Testing:** iPhone 12, Pixel 5
- **Parallel Execution:** Optimized for CI/CD
- **Retries:** 2 retries on CI, 0 locally
- **Timeouts:** 30s per test

### App-Specific Base URLs

| App           | Development URL         | Port |
| ------------- | ----------------------- | ---- |
| Dashboard     | `http://localhost:5173` | 5173 |
| Support       | `http://localhost:3002` | 3002 |
| Dev Server UI | `http://localhost:5173` | 5173 |

### Enabling Auto-Start Dev Server

To automatically start the dev server before tests, uncomment the `webServer` section in `playwright.config.ts`:

```typescript
webServer: {
  command: 'pnpm dev',
  url: 'http://localhost:<port>',
  reuseExistingServer: !process.env.CI,
},
```

## Best Practices

### 1. Use Data Test IDs

```typescript
// ❌ Fragile - depends on text content
await page.click('text=Submit');

// ✅ Robust - uses semantic attributes
await page.click('[data-testid="pws-submit-button"]');
```

#### Playwright Selector Convention (pws- prefix)

For test selectors that are specifically added for Playwright testing, we use the `pws-` prefix convention. This makes it clear which data attributes are purely for testing purposes and helps distinguish them from other data attributes that may have application logic purposes.

**Naming Convention:**

- Prefix: `pws-` (stands for "PlayWright Selector")
- Format: `pws-{component-type}-{element-name}`
- Examples: `pws-nav-metrics`, `pws-button-submit`, `pws-form-login`

**Usage in Components:**

```typescript
// Add the dataTestId prop to your component
<MenuItem text="Metrics" icon={<MetricsIcon />} to="/metrics" dataTestId="pws-nav-metrics" />
```

**Usage in Tests:**

```typescript
// Use getByTestId for cleaner test code
await page.getByTestId('pws-nav-metrics').click();

// Or use the attribute selector
await page.click('[data-testid="pws-nav-metrics"]');

// Verify visibility
await expect(page.getByTestId('pws-nav-metrics')).toBeVisible();
```

**Benefits:**

- **Clear Intent:** The `pws-` prefix makes it obvious that this attribute is for Playwright testing
- **Namespace Separation:** Distinguishes test selectors from other data attributes
- **Searchability:** Easy to find all Playwright-specific test IDs in the codebase
- **Durability:** These selectors are less likely to change than text content or CSS classes

**Examples from the Dashboard:**

Navigation menu items:

- `pws-nav-metrics` - Metrics page navigation
- `pws-nav-runs` - Runs page navigation
- `pws-nav-events` - Events page navigation
- `pws-nav-insights` - Insights page navigation
- `pws-nav-apps` - Apps page navigation
- `pws-nav-functions` - Functions page navigation
- `pws-nav-event-types` - Event Types page navigation
- `pws-nav-webhooks` - Webhooks page navigation

### 2. Wait for Elements Properly

```typescript
// ❌ Hard-coded delays
await page.waitForTimeout(2000);

// ✅ Wait for specific conditions
await expect(page.locator('[data-testid="loading"]')).toBeHidden();
await expect(page.locator('[data-testid="content"]')).toBeVisible();
```

### 3. Handle Asynchronous Operations

```typescript
// ✅ Wait for async operations
test('should handle async function execution', async ({ page }) => {
  await page.goto('/functions/my-async-function');

  // Trigger async function
  await page.click('[data-testid="trigger-function"]');

  // Wait for execution to start
  await expect(page.locator('[data-testid="execution-status"]')).toContainText('Running');

  // Wait for completion with appropriate timeout
  await expect(page.locator('[data-testid="execution-status"]')).toContainText('Completed', {
    timeout: 30000, // 30 second timeout
  });
});
```

### 4. Use Page Object Model for Complex Flows

```typescript
// tests/utils/pages/functions-page.ts
export class FunctionsPage {
  constructor(private page: Page) {}

  async goto() {
    await this.page.goto('/env/production/functions');
  }

  async pauseFunction(functionName: string) {
    await this.page.click(`[data-function="${functionName}"] [data-testid="pause-button"]`);
    await expect(this.page.locator('.toast-success')).toBeVisible();
  }
}

// In your test
const functionsPage = new FunctionsPage(page);
await functionsPage.goto();
await functionsPage.pauseFunction('my-function');
```

### 5. Mock External API Dependencies

```typescript
// ✅ Mock external APIs for reliable tests
test('should handle API failures gracefully', async ({ page }) => {
  // Mock API failure
  await page.route('**/api/**', (route) => {
    route.fulfill({ status: 500 });
  });

  await page.goto('/your-route');

  // Verify error handling
  await expect(page.locator('[data-testid="error-message"]')).toBeVisible();
});
```

### 6. Test Responsive Design

```typescript
test('should work on mobile', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 667 });
  await page.goto('/functions');

  // Test mobile-specific behavior
  await page.click('[data-testid="mobile-menu-toggle"]');
  await expect(page.locator('[data-testid="mobile-menu"]')).toBeVisible();
});
```

## Troubleshooting

### Common Issues

**Tests fail with `net::ERR_CONNECTION_REFUSED`**

This error means the dev server is not running. Make sure you have started the dev server before running tests:

```bash
# Terminal 1: Start dev server
cd ui/apps/{dashboard|support|dev-server-ui}
pnpm dev

# Terminal 2: Run tests
cd ui/apps/{dashboard|support|dev-server-ui}
pnpm test:e2e
```

Alternatively, enable auto-start by uncommenting the `webServer` section in `playwright.config.ts`:

```typescript
webServer: {
  command: 'pnpm dev',
  url: 'http://localhost:<port>',
  reuseExistingServer: !process.env.CI,
},
```

**Tests fail with "Element not found"**

```typescript
// Add explicit waits
await expect(page.locator('[data-testid="element"]')).toBeVisible();
```

**Tests are flaky**

```typescript
// Use retry assertions
await expect(async () => {
  await page.click('[data-testid="button"]');
  await expect(page.locator('[data-testid="result"]')).toBeVisible();
}).toPass({ timeout: 10000 });
```

**Authentication issues (Dashboard/Support)**

```typescript
// Clear storage before tests
test.beforeEach(async ({ page }) => {
  await page.context().clearCookies();
});

// Or ensure auth state is loaded
test.use({ storageState: 'playwright/.auth/user.json' });
```

**Monaco Editor not responding (Dev Server UI)**

```typescript
// Add proper waits for Monaco
await expect(page.locator('.monaco-editor')).toBeVisible();
await page.waitForFunction(() => {
  const editor = document.querySelector('.monaco-editor');
  return editor && editor.classList.contains('monaco-editor-loaded');
});
```

**Function executions timing out (Dev Server UI)**

```typescript
// Increase timeouts for development functions
test.setTimeout(60000); // 1 minute for dev functions

await expect(page.locator('[data-testid="execution-result"]')).toBeVisible({
  timeout: 30000, // 30 seconds for function execution
});
```

### Debugging Tips

1. **Use `page.pause()` to debug:**

   ```typescript
   test('debug test', async ({ page }) => {
     await page.goto('/');
     await page.pause(); // Opens browser inspector
   });
   ```

2. **Take screenshots on failure:**

   ```typescript
   test.afterEach(async ({ page }, testInfo) => {
     if (testInfo.status !== testInfo.expectedStatus) {
       await page.screenshot({ path: `test-failure-${testInfo.title}.png` });
     }
   });
   ```

3. **Enable verbose logging:**

   ```bash
   DEBUG=pw:api pnpm test:e2e
   ```

4. **Enable console logging:**

   ```typescript
   test('debug with console', async ({ page }) => {
     page.on('console', (msg) => console.log(`PAGE LOG: ${msg.text()}`));

     // Your test here...
   });
   ```

### Development Environment Setup

Make sure your development environment is properly configured:

```bash
# Navigate to the app directory
cd ui/apps/{dashboard|support|dev-server-ui}

# Ensure all dependencies are installed
pnpm install

# Install Playwright browsers (first time only)
npx playwright install

# Start the development server
pnpm dev

# In another terminal, verify the server is running
curl http://localhost:<port> || echo "Server not ready"

# Then run your tests
pnpm test:e2e
```

## Integration with CI/CD

Add this to your GitHub Actions workflow:

```yaml
- name: Install dependencies
  run: pnpm install

- name: Install Playwright browsers
  run: npx playwright install --with-deps

- name: Run Dashboard tests
  working-directory: ui/apps/dashboard
  run: pnpm test:e2e

- name: Run Support tests
  working-directory: ui/apps/support
  run: pnpm test:e2e

- name: Run Dev Server UI tests
  working-directory: ui/apps/dev-server-ui
  run: pnpm test:e2e

- name: Upload test results
  uses: actions/upload-artifact@v3
  if: always()
  with:
    name: playwright-reports
    path: |
      ui/apps/dashboard/playwright-report/
      ui/apps/support/playwright-report/
      ui/apps/dev-server-ui/playwright-report/
```

## Additional Resources

- [Playwright Documentation](https://playwright.dev)
- [Playwright Best Practices](https://playwright.dev/docs/best-practices)
- [TanStack Router Testing](https://tanstack.com/router/latest/docs/framework/react/guide/testing)
- [Clerk Testing Guide](https://clerk.com/docs/testing/overview)
- [Testing Monaco Editor](https://github.com/microsoft/monaco-editor/blob/main/docs/integrate-esm.md)
