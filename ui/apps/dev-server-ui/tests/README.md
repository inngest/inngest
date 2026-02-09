# Playwright Testing - Inngest Dev Server UI

> **📖 See the [main testing guide](../../../TESTING.md) for comprehensive documentation.**

This directory contains E2E tests for the Inngest Development Server UI.

**Current Status:** The test suite is in early stages with basic configuration tests and function triggering tests. The examples in this guide serve as patterns for writing additional tests as the test coverage expands.

## Quick Start

```bash
# Navigate to the dev-server-ui directory
cd ui/apps/dev-server-ui

# Run unit tests (vitest)
pnpm test

# Run E2E tests
pnpm test:e2e

# Run E2E tests with visible browser
pnpm test:e2e:headed

# Generate new tests
pnpm test:e2e:codegen http://localhost:5173
```

## Dev Server UI-Specific Details

### Port & URL

- **Development:** `http://localhost:5173`
- **Base URL configured in:** `playwright.config.ts`

### Authentication

The Dev Server UI does not require authentication for testing.

### Test Structure

**Current Structure:**

```
tests/
├── README.md              # This file
├── example.spec.ts        # Basic configuration and environment tests
└── functions/
    └── trigger.spec.ts    # ✅ Exists - Function triggering and execution tests
```

**Recommended Structure for Future Tests:**

```
tests/
├── README.md
├── example.spec.ts
├── functions/
│   ├── trigger.spec.ts    # ✅ Exists
│   ├── list.spec.ts       # Function listing in dev
│   ├── editor.spec.ts     # Code editing tests
│   └── logs.spec.ts       # Log viewing tests
├── events/
│   ├── send.spec.ts       # Event sending
│   └── history.spec.ts    # Event history
├── debugging/
│   └── runs.spec.ts       # Run inspection
└── utils/
    └── dev-helpers.ts     # Dev-specific test utilities
```

### Environment Variables

Create `.env.test.local`:

```bash
# Dev Server UI
VITE_ENV=test
INNGEST_DEV=true
```

These variables are validated in the configuration tests ([example.spec.ts](example.spec.ts)).

## Common Dev Server UI Test Patterns

### Testing Function Triggering

```typescript
test('should trigger function in dev environment', async ({ page }) => {
  await page.goto('/functions/my-function');

  // Click trigger button
  await page.click('[data-testid="trigger-function"]');

  // Wait for execution feedback
  await expect(page.locator('[data-testid="execution-result"]')).toBeVisible();

  // Verify logs appear
  await expect(page.locator('[data-testid="function-logs"]'))
    .toContainText('Function started');
});
```

### Testing Monaco Editor

```typescript
test('should edit function code in Monaco', async ({ page }) => {
  await page.goto('/functions/my-function/edit');

  // Wait for Monaco editor to fully load
  await expect(page.locator('.monaco-editor')).toBeVisible();
  await page.waitForTimeout(1000); // Monaco needs time to initialize

  // Focus the editor
  await page.click('.monaco-editor .view-lines');

  // Clear and type new code
  await page.keyboard.press('Control+A');
  await page.keyboard.type('console.log("test");');

  // Save the function
  await page.click('[data-testid="save-function"]');

  // Verify save
  await expect(page.locator('[data-testid="save-success"]')).toBeVisible();
});
```

### Testing Real-time Updates

```typescript
test('should show real-time function execution', async ({ page }) => {
  await page.goto('/');

  // Wait for real-time connection
  await expect(page.locator('[data-testid="connection-status"]'))
    .toContainText('Connected');

  // Trigger a function
  await page.goto('/functions/my-function');
  await page.click('[data-testid="trigger-function"]');

  // Go back to dashboard and verify real-time update
  await page.goto('/');
  await expect(page.locator('[data-testid="latest-execution"]')).toBeVisible();
});
```

### Testing Event Sending

```typescript
test('should send test event', async ({ page }) => {
  await page.goto('/events');

  // Click send event button
  await page.click('[data-testid="send-event"]');

  // Fill event data
  await page.fill('[data-testid="event-name"]', 'test.event');
  await page.fill('[data-testid="event-data"]', '{"userId": "123"}');

  // Send the event
  await page.click('[data-testid="submit-event"]');

  // Verify event was sent
  await expect(page.locator('[data-testid="event-success"]')).toBeVisible();
});
```

## Important Notes

### Selector Strategies

The examples in this guide use specific `data-testid` attributes for clarity. In practice, you may need to use flexible selectors or inspect the actual UI to find the correct attributes.

See [functions/trigger.spec.ts](functions/trigger.spec.ts) for examples of resilient selector strategies using wildcards like `[data-testid*="trigger"]` that match partial attribute values.

### Handling Timeouts

Dev server functions may take longer to execute:

```typescript
// Increase timeouts for development functions
test.setTimeout(60000); // 1 minute for dev functions

await expect(page.locator('[data-testid="execution-result"]')).toBeVisible({
  timeout: 30000 // 30 seconds for function execution
});
```

## Available Commands

| Command                 | Description                                |
| ----------------------- | ------------------------------------------ |
| `pnpm test`             | Run unit tests with Vitest                 |
| `pnpm test:e2e`         | Run E2E tests in headless mode             |
| `pnpm test:e2e:headed`  | Run E2E tests with visible browsers        |
| `pnpm test:e2e:debug`   | Run E2E tests in debug mode                |
| `pnpm test:e2e:ui`      | Open Playwright UI for interactive testing |
| `pnpm test:e2e:codegen` | Launch code generator                      |
| `pnpm test:e2e:report`  | View HTML test report                      |

For more detailed information, see the [main testing guide](../../../TESTING.md).
