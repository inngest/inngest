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
    await page.goto('/env/production/functions', {
      waitUntil: 'domcontentloaded',
    });

    // Should redirect to sign-in page
    await expect(page).toHaveURL(/sign-in/, { timeout: 10000 });

    // Should show sign-in form
    await expect(
      page.locator('form, [data-testid="sign-in-form"], input[type="email"]'),
    ).toBeVisible({ timeout: 10000 });
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
    // Use 'domcontentloaded' instead of 'networkidle' - auth flows have persistent connections
    await page.goto('/env/production/functions/my-function', {
      waitUntil: 'domcontentloaded',
      timeout: 15000,
    });

    // Wait for redirect to sign-in by checking for sign-in UI elements
    await Promise.race([
      page.waitForURL(/sign-in/, { timeout: 10000 }),
      page.waitForSelector('input[type="email"], button:has-text("Sign in")', {
        timeout: 10000,
        state: 'visible',
      }),
    ]);

    // Verify we're on sign-in page or in auth flow
    const currentUrl = page.url();
    expect(currentUrl).toMatch(/sign-in|clerk/);

    // Verify redirect URL is preserved - check both URL params and sign-up link
    const signUpLink = await page
      .locator('a[href*="sign-up"]')
      .getAttribute('href');
    const redirectPreserved =
      currentUrl.includes('my-function') ||
      (signUpLink && signUpLink.includes('my-function'));
    expect(redirectPreserved).toBeTruthy();
  });
});
