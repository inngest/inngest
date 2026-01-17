import { test as setup } from "@playwright/test";

const authFile = "playwright/.auth/user.json";

setup("authenticate", async ({ page }) => {
  console.log("🔐 Starting authentication setup for Support Portal...");

  // Get test credentials from environment variables
  const email = process.env.TEST_USER_EMAIL;
  const password = process.env.TEST_USER_PASSWORD;

  if (!email || !password) {
    throw new Error(
      "TEST_USER_EMAIL and TEST_USER_PASSWORD must be set in .env.test.local",
    );
  }

  // Navigate to sign-in page
  await page.goto("/sign-in");

  // TODO: Record your sign-in steps using pnpm test:codegen:record-auth
  // Then replace this comment with your recorded Clerk authentication steps
  // For example:
  // await page.getByRole('textbox', { name: 'Email address' }).fill(email);
  // await page.getByRole('button', { name: 'Continue' }).click();
  // await page.getByRole('textbox', { name: 'Password' }).fill(password);
  // await page.getByRole('button', { name: 'Continue' }).click();

  // Wait for successful authentication
  // await page.waitForURL(/support/, { timeout: 10000 });

  // Verify we're actually signed in
  // await expect(page.locator('nav, [data-testid*="user"], [data-testid*="profile"]')).toBeVisible({ timeout: 5000 });

  console.log("🔐 Authentication setup completed");

  // Save signed-in state to file
  await page.context().storageState({ path: authFile });
});

export { authFile };
