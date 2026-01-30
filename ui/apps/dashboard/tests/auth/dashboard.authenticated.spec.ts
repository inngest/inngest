import { test, expect } from '@playwright/test';

// These tests will automatically use the saved authentication state
// No need to sign in again - it's done once in auth.setup.ts

test.describe('Authenticated Dashboard Tests', () => {
  test('should access dashboard when already signed in', async ({ page }) => {
    await page.goto('/');

    // Should be on dashboard (not redirected to sign-in)
    await expect(page).not.toHaveURL(/sign-in/);

    // Should show authenticated user content
    await expect(
      page.locator(
        '[data-testid="user-menu"], [data-testid="user-profile"], nav',
      ),
    ).toBeVisible();
  });

  // Add more authenticated tests here
  // All of these will run with the user already signed in
});
