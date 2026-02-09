import { test, expect } from "@playwright/test";

test.describe("Support Portal - Configuration Test", () => {
  test("Playwright configuration is valid", async ({ page }) => {
    // Test that we can create a page instance
    expect(page).toBeTruthy();

    // Test that we can set viewport
    await page.setViewportSize({ width: 1920, height: 1080 });

    // Test that basic navigation works (to about:blank)
    await page.goto("about:blank");
    await expect(page).toHaveURL("about:blank");
  });

  test("Browser capabilities work", async ({ page }) => {
    await page.goto(
      "data:text/html,<html><body><h1>Support Portal Test</h1></body></html>",
    );

    // Test basic selectors and assertions work
    await expect(page.locator("h1")).toHaveText("Support Portal Test");
    await expect(page.locator("body")).toBeVisible();
  });

  test("Environment variables are loaded", async () => {
    // Test that environment variables are accessible
    expect(process.env.VITE_ENV).toBe("test");
    expect(process.env.TEST_USER_EMAIL).toBeDefined();
    expect(process.env.TEST_USER_PASSWORD).toBeDefined();
  });
});
