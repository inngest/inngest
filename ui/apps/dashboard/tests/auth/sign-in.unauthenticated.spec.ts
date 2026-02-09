import { test, expect } from '@playwright/test';

// These tests run WITHOUT authentication to test the sign-in flow itself
// They use the "unauthenticated" project which doesn't load saved auth state

test.describe('Sign-in Flow Tests', () => {
  test('should redirect unauthenticated users to sign-in', async ({
    page,
    context,
  }) => {
    // Clear all cookies and local storage to ensure unauthenticated state
    await context.clearCookies();
    await context.clearPermissions();

    await page.goto('/env/production/functions');

    // Should redirect to sign-in page
    await expect(page).toHaveURL(/sign-in/);
  });

  test('should show sign-in form', async ({ page }) => {
    await page.goto('/sign-in');

    // Should show sign-in elements
    await expect(page.locator('form')).toBeVisible();
    await expect(
      page.getByRole('textbox', { name: 'Email address' }),
    ).toBeVisible();
    await expect(page.getByRole('textbox', { name: 'Password' })).toBeVisible();
  });

  // Add your recorded sign-in test here if you want to test the actual flow
  // test('should sign in successfully', async ({ page }) => {
  //   // PASTE YOUR RECORDED SIGN-IN STEPS HERE
  // });
});
