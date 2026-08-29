import { expect, test } from '@playwright/test';

const cases = [
  'function-running-gaps',
  'function-backlog-independent-gaps',
  'scoped-throughput-independent-gaps',
  'app-backlog-leading-trailing-gaps',
  'app-concurrency-aggregated-zeroes',
  'account-concurrency-limit',
] as const;

test.beforeEach(async ({ page }) => {
  await page.goto('/');
  await page.waitForFunction(() => document.fonts.status === 'loaded');
  await page.locator('html[data-visual-ready="true"]').waitFor();
});

for (const name of cases) {
  test(name, async ({ page }) => {
    const chart = page.getByTestId(name);
    await expect(chart).toBeVisible();
    await expect(chart).toHaveScreenshot(`${name}.png`);
  });
}
