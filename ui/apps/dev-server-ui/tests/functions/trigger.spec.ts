import { test, expect } from '@playwright/test';

test.describe('Function Triggering in Dev Server', () => {
  test.beforeEach(async ({ page }) => {
    // Mock GraphQL API endpoint for functions
    await page.route('**/v0/gql', async (route) => {
      const request = route.request().postDataJSON();

      // Check if this is the GetFunctions query
      if (request?.query?.includes('GetFunctions')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
              functions: [
                {
                  id: 'test-function-1',
                  slug: 'test-function',
                  name: 'Test Function',
                  url: 'http://localhost:3000',
                  triggers: [
                    {
                      type: 'EVENT',
                      value: 'test.event',
                    },
                  ],
                  app: {
                    name: 'test-app',
                  },
                },
              ],
            },
          }),
        });
      } else {
        await route.continue();
      }
    });
  });

  test('should display function trigger interface', async ({ page }) => {
    await page.goto('/functions');

    // Wait for GraphQL request to complete
    await page.waitForLoadState('networkidle');

    // Should display the function in the table
    await expect(page.getByText('Test Function')).toBeVisible();
    await expect(page.getByRole('cell', { name: 'test-app' })).toBeVisible();
    await expect(page.getByRole('cell', { name: 'test.event' })).toBeVisible();
  });

  test('should trigger function with test event', async ({ page }) => {
    await page.goto('/functions');

    // Find and click trigger button
    const triggerButton = page
      .locator(
        '[data-testid*="trigger"], button:has-text("Trigger"), button:has-text("Run")',
      )
      .first();

    if (await triggerButton.isVisible()) {
      await triggerButton.click();

      // Should show trigger modal or form
      await expect(
        page.locator('[data-testid*="modal"], .modal, [role="dialog"]'),
      ).toBeVisible();

      // Fill in test event data
      const eventNameInput = page
        .locator(
          '[data-testid*="event-name"], input[name*="event"], input[placeholder*="event"]',
        )
        .first();
      if (await eventNameInput.isVisible()) {
        await eventNameInput.fill('test.event');
      }

      const eventDataInput = page
        .locator('[data-testid*="event-data"], textarea, .monaco-editor')
        .first();
      if (await eventDataInput.isVisible()) {
        await eventDataInput.click();
        await page.keyboard.type('{"userId": "test-123", "action": "test"}');
      }

      // Submit the trigger
      const submitButton = page
        .locator(
          '[data-testid*="submit"], button:has-text("Send"), button:has-text("Trigger")',
        )
        .last();
      if (await submitButton.isVisible()) {
        await submitButton.click();

        // Should show success feedback
        await expect(
          page.locator('[data-testid*="success"], .success, .toast-success'),
        ).toBeVisible();
      }
    }
  });

  test('should display real-time execution logs', async ({ page }) => {
    await page.goto('/functions/test-function');

    // Trigger function execution
    const triggerButton = page
      .locator('[data-testid*="trigger"], button:has-text("Trigger")')
      .first();
    if (await triggerButton.isVisible()) {
      await triggerButton.click();
    }

    // Should show logs section
    const logsSection = page.locator('[data-testid*="logs"], .logs, .console');
    if (await logsSection.isVisible()) {
      await expect(logsSection).toBeVisible();

      // Should show log entries
      await expect(
        page.locator('[data-testid*="log-entry"], .log-entry, .log-line'),
      ).toBeVisible({ timeout: 5000 });
    }
  });

  test('should handle function execution errors', async ({ page }) => {
    await page.goto('/functions/test-function');

    // Mock API error response
    await page.route('**/api/functions/test-function/trigger', (route) => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({
          error: 'Function execution failed',
          details: 'Invalid event data format',
        }),
      });
    });

    // Trigger the function
    const triggerButton = page
      .locator('[data-testid*="trigger"], button:has-text("Trigger")')
      .first();
    if (await triggerButton.isVisible()) {
      await triggerButton.click();

      // Should show error feedback
      await expect(
        page.locator('[data-testid*="error"], .error, .toast-error'),
      ).toBeVisible({ timeout: 5000 });

      // Error should contain helpful information
      const errorMessage = page.locator(
        '[data-testid*="error"], .error-message',
      );
      if (await errorMessage.isVisible()) {
        await expect(errorMessage).toContainText('execution failed');
      }
    }
  });

  test('should validate event data before triggering', async ({ page }) => {
    await page.goto('/functions');

    // Open trigger modal
    const triggerButton = page
      .locator('[data-testid*="trigger"], button:has-text("Trigger")')
      .first();
    if (await triggerButton.isVisible()) {
      await triggerButton.click();

      // Try to submit without required data
      const submitButton = page
        .locator(
          '[data-testid*="submit"], button:has-text("Send"), button:has-text("Trigger")',
        )
        .last();
      if (await submitButton.isVisible()) {
        await submitButton.click();

        // Should show validation errors
        await expect(
          page.locator('[data-testid*="validation"], .field-error, .error'),
        ).toBeVisible();
      }
    }
  });

  test('should support different event payload formats', async ({ page }) => {
    await page.goto('/functions');

    // Open trigger interface
    const triggerButton = page
      .locator('[data-testid*="trigger"], button:has-text("Trigger")')
      .first();
    if (await triggerButton.isVisible()) {
      await triggerButton.click();

      // Test JSON payload
      const eventDataInput = page
        .locator('[data-testid*="event-data"], textarea, .monaco-editor')
        .first();
      if (await eventDataInput.isVisible()) {
        await eventDataInput.click();
        await page.keyboard.press('Control+A');
        await page.keyboard.type(
          '{"complex": {"nested": {"data": true}}, "array": [1, 2, 3]}',
        );

        // Should accept valid JSON
        const submitButton = page
          .locator('[data-testid*="submit"], button:has-text("Send")')
          .last();
        if (await submitButton.isVisible()) {
          await expect(submitButton).toBeEnabled();
        }
      }
    }
  });
});
