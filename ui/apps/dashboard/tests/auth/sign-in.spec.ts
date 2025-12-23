import { test, expect } from '@playwright/test';

test.describe('Dashboard Authentication', () => {
  test.beforeEach(async () => {
    // Prevent rate limiting from Clerk
    await new Promise((resolve) => setTimeout(resolve, 1000));
  });

  test('should redirect unauthenticated users to sign-in', async ({ page }) => {
    // Clear any existing authentication
    await page.context().clearCookies();
    await page.context().clearPermissions();

    // Try to access a protected route
    await page.goto('/env/production/functions');

    // Should redirect to sign-in page
    await expect(page).toHaveURL(/sign-in/);

    // Should show sign-in form
    await expect(
      page.locator('form, [data-testid="sign-in-form"]'),
    ).toBeVisible();
  });

  test('should show proper sign-in page layout', async ({ page }) => {
    await page.goto('/sign-in');

    // Should show Inngest branding
    await expect(
      page.locator('[alt*="Inngest"], [data-testid="logo"]'),
    ).toBeVisible();

    // Should have sign-in options
    await expect(
      page.getByRole('button', { name: 'Sign in with GitHub' }),
    ).toBeVisible();
    await expect(
      page.getByRole('button', { name: 'Sign in with Google' }),
    ).toBeVisible();
    await expect(page.getByRole('button', { name: 'Continue' })).toBeVisible();

    // Should be responsive
    await page.setViewportSize({ width: 375, height: 667 });
    await expect(page.locator('body')).toBeVisible();
  });

  test('should handle authentication errors gracefully', async ({ page }) => {
    // Mock authentication failure
    await page.route('**/oauth/**', (route) => route.abort());
    await page.route('**/api/auth/**', (route) => route.abort());

    await page.goto('/sign-in');

    // Try to trigger authentication (this will depend on your auth provider)
    const authButton = page
      .locator('button:has-text("Sign"), button:has-text("Login")')
      .first();
    if (await authButton.isVisible()) {
      await authButton.click();
    }

    // Should handle the error gracefully (no crash)
    await expect(page.locator('body')).toBeVisible();
  });

  test('should preserve redirect URL after sign-in', async ({ page }) => {
    // Clear authentication state first
    await page.context().clearCookies();
    await page.context().clearPermissions();

    // Try to access specific page while unauthenticated
    await page.goto('/env/production/functions/my-function', {
      waitUntil: 'networkidle',
    });

    // Wait for auth redirect to complete - check for either sign-in page OR Clerk handshake
    await page.waitForURL(
      (url) =>
        url.pathname.includes('sign-in') || url.hostname.includes('clerk'),
      { timeout: 10000 },
    );

    // Verify we're in an auth flow
    expect(
      page.url().includes('sign-in') || page.url().includes('clerk'),
    ).toBeTruthy();

    // Verify redirect URL is preserved in query params
    expect(page.url()).toContain('my-function');
  });
});
