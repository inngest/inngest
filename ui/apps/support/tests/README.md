# Playwright Testing - Inngest Support Portal

> **📖 See the [main testing guide](../../../TESTING.md) for comprehensive documentation.**

This directory contains E2E tests for the Inngest Support Portal.

## Quick Start

```bash
# Navigate to the support directory
cd ui/apps/support

# Run tests
pnpm test:e2e

# Run tests with visible browser
pnpm test:e2e:headed

# Generate new tests
pnpm test:e2e:codegen http://localhost:3002
```

## Support Portal-Specific Details

### Port & URL

- **Development:** `http://localhost:3002`
- **Base URL configured in:** `playwright.config.ts`

### Authentication

The Support Portal uses Clerk for authentication, similar to the Dashboard.

See the [Authentication Setup section](../../../TESTING.md#authentication-setup) in the main guide for detailed instructions.

### Test Structure

```
tests/
├── README.md              # This file
├── example.spec.ts        # Basic configuration tests
├── auth/
│   ├── auth.setup.ts     # Authentication setup
│   └── sign-in.spec.ts   # Auth flow tests
├── tickets/
│   ├── list.spec.ts
│   ├── details.spec.ts
│   └── timeline.spec.ts
├── api/
│   ├── plain-integration.spec.ts
│   └── error-handling.spec.ts
└── utils/
    ├── auth-helpers.ts
    └── plain-helpers.ts
```

### Environment Variables

Create `.env.test.local`:

```bash
# Clerk testing keys
CLERK_PUBLISHABLE_KEY=your_test_key
CLERK_SECRET_KEY=your_test_secret

# Test user credentials
TEST_USER_EMAIL=support-test@example.com
TEST_USER_PASSWORD=test-password

# Plain API testing
PLAIN_API_KEY=test_plain_key

# Inngest API testing
VITE_API_URL=http://localhost:8080

# Base URL
PLAYWRIGHT_BASE_URL=http://localhost:3002
```

## Common Support Portal Test Patterns

### Testing Plain API Integration

```typescript
test("should handle Plain API responses", async ({ page }) => {
  // Intercept Plain API calls
  await page.route("**/graphql", (route) => {
    if (route.request().postData()?.includes("getCustomerByEmail")) {
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          data: {
            customer: {
              id: "test-customer-id",
              email: "test@example.com",
              tickets: [],
            },
          },
        }),
      });
    } else {
      route.continue();
    }
  });

  await page.goto("/support");

  await expect(page.locator('[data-testid="customer-email"]')).toContainText(
    "test@example.com",
  );
});
```

### Testing Ticket Management

```typescript
test("should display ticket list", async ({ page }) => {
  await page.goto("/support");

  // Wait for tickets to load from Plain API
  await expect(page.locator('[data-testid="tickets-list"]')).toBeVisible();

  // Check for ticket entries
  await expect(
    page.locator('[data-testid="ticket-item"]').first(),
  ).toBeVisible();
});

test("should view ticket details", async ({ page }) => {
  await page.goto("/support");

  // Click on a specific ticket
  await page.click('[data-testid="ticket-item"]');

  // Should navigate to ticket detail page
  await expect(page).toHaveURL(/case\/[^/]+/);

  // Verify ticket details are shown
  await expect(page.locator('[data-testid="ticket-title"]')).toBeVisible();
  await expect(page.locator('[data-testid="ticket-timeline"]')).toBeVisible();
});
```

### Testing SSR Features

```typescript
test("should render content server-side", async ({ page }) => {
  // Disable JavaScript to test SSR
  await page.setJavaScriptEnabled(false);

  await page.goto("/support");

  // Should still show content (rendered server-side)
  await expect(page.locator('[data-testid="support-layout"]')).toBeVisible();

  // Re-enable JavaScript for hydration
  await page.setJavaScriptEnabled(true);
  await page.reload();

  // Should be interactive after hydration
  await page.click('[data-testid="interactive-button"]');
  await expect(
    page.locator('[data-testid="interaction-result"]'),
  ).toBeVisible();
});
```

## Available Commands

| Command                 | Description                                |
| ----------------------- | ------------------------------------------ |
| `pnpm test:e2e`         | Run all E2E tests in headless mode         |
| `pnpm test:e2e:headed`  | Run E2E tests with visible browsers        |
| `pnpm test:e2e:debug`   | Run E2E tests in debug mode                |
| `pnpm test:e2e:ui`      | Open Playwright UI for interactive testing |
| `pnpm test:e2e:codegen` | Launch code generator                      |
| `pnpm test:e2e:report`  | View HTML test report                      |

For more detailed information, see the [main testing guide](../../../TESTING.md).
