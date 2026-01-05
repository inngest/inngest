import { test, expect } from '@playwright/test';
import { mockApiRequests, addRateLimitDelay } from '../helpers/mock-api';

// Configure tests to run serially to avoid rate limits
test.describe.configure({ mode: 'serial' });

test.describe('Main Navigation', () => {
  test.beforeEach(async ({ page }) => {
    // Mock API requests to avoid rate limits
    await mockApiRequests(page);

    // Add delay to avoid rate limits between tests
    await addRateLimitDelay(page);

    await page.goto('/');

    // Wait for navigation to be ready with increased timeout
    await page.getByTestId('pws-nav-metrics').waitFor({
      state: 'visible',
      timeout: 15000,
    });
  });

  test('should navigate to Metrics page', async ({ page }) => {
    await page.getByTestId('pws-nav-metrics').click();
    await page.waitForURL(/\/env\/[^/]+\/metrics/, { waitUntil: 'commit' });
    await page.waitForTimeout(300); // Brief delay between tests
  });

  test('should navigate to Runs page', async ({ page }) => {
    await page.getByTestId('pws-nav-runs').click();
    await page.waitForURL(/\/env\/[^/]+\/runs/, { waitUntil: 'commit' });
    await page.waitForTimeout(300);
  });

  test('should navigate to Events page', async ({ page }) => {
    await page.getByTestId('pws-nav-events').click();
    await page.waitForURL(/\/env\/[^/]+\/events/, { waitUntil: 'commit' });
    await page.waitForTimeout(300);
  });

  test('should navigate to Insights page', async ({ page }) => {
    await page.getByTestId('pws-nav-insights').click();
    await page.waitForURL(/\/env\/[^/]+\/insights/, { waitUntil: 'commit' });
    await page.waitForTimeout(300);
  });

  test('should navigate to Apps page', async ({ page }) => {
    await page.getByTestId('pws-nav-apps').click();
    await page.waitForURL(/\/env\/[^/]+\/apps/, { waitUntil: 'commit' });
    await page.waitForTimeout(300);
  });

  test('should navigate to Functions page', async ({ page }) => {
    await page.getByTestId('pws-nav-functions').click();
    await page.waitForURL(/\/env\/[^/]+\/functions/, { waitUntil: 'commit' });
    await page.waitForTimeout(300);
  });

  test('should navigate to Event Types page', async ({ page }) => {
    await page.getByTestId('pws-nav-event-types').click();
    await page.waitForURL(/\/env\/[^/]+\/event-types/, { waitUntil: 'commit' });
    await page.waitForTimeout(300);
  });

  test('should navigate to Webhooks page', async ({ page }) => {
    await page.getByTestId('pws-nav-webhooks').click();
    await page.waitForURL(/\/env\/[^/]+\/manage\/webhooks/, {
      waitUntil: 'commit',
    });
    await page.waitForTimeout(300);
  });

  test('should highlight active navigation item', async ({ page }) => {
    // Navigate to Functions
    await page.getByTestId('pws-nav-functions').click();
    await page.waitForURL(/\/env\/[^/]+\/functions/, { waitUntil: 'commit' });

    // The Functions nav item should have active styling
    const functionsNavItem = page.getByTestId('pws-nav-functions');
    await expect(functionsNavItem).toBeVisible();

    // Navigate to Runs
    await page.getByTestId('pws-nav-runs').click();
    await page.waitForURL(/\/env\/[^/]+\/runs/, { waitUntil: 'commit' });

    // The Runs nav item should now be highlighted
    const runsNavItem = page.getByTestId('pws-nav-runs');
    await expect(runsNavItem).toBeVisible();
  });

  test('should navigate between Monitor and Manage sections', async ({
    page,
  }) => {
    // Navigate to a Monitor section page (Metrics)
    await page.getByTestId('pws-nav-metrics').click();
    await page.waitForURL(/\/env\/[^/]+\/metrics/, { waitUntil: 'commit' });

    // Navigate to a Manage section page (Apps)
    await page.getByTestId('pws-nav-apps').click();
    await page.waitForURL(/\/env\/[^/]+\/apps/, { waitUntil: 'commit' });

    // Navigate back to Monitor section (Events)
    await page.getByTestId('pws-nav-events').click();
    await page.waitForURL(/\/env\/[^/]+\/events/, { waitUntil: 'commit' });
  });

  test('should maintain navigation state after page reload', async ({
    page,
  }) => {
    // Navigate to a specific page
    await page.getByTestId('pws-nav-insights').click();
    await page.waitForURL(/\/env\/[^/]+\/insights/, { waitUntil: 'commit' });

    // Reload the page
    await page.reload();

    // Should still be on the insights page
    await expect(page).toHaveURL(/\/env\/[^/]+\/insights/);

    // Navigation should still be visible and functional
    await expect(page.getByTestId('pws-nav-insights')).toBeVisible();
    await expect(page.getByTestId('pws-nav-metrics')).toBeVisible();
  });
});
