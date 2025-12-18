import { type Page } from '@playwright/test';

/**
 * Mock API requests to avoid rate limits and speed up tests
 */
export async function mockApiRequests(page: Page) {
  // Mock general API requests
  await page.route('**/api/**', async (route) => {
    const url = route.request().url();

    // Allow Clerk auth requests to go through (using stored auth)
    if (url.includes('clerk.')) {
      return route.continue();
    }

    // Mock other API responses
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: {} }),
    });
  });

  // Mock GraphQL requests with minimal valid responses
  await page.route('**/graphql', async (route) => {
    const postData = route.request().postDataJSON();
    const operationName = postData?.operationName || '';

    // Provide minimal mock data based on operation
    let mockData = { data: {} };

    if (operationName.includes('Environment')) {
      mockData = {
        data: {
          environment: {
            id: 'test-env',
            name: 'Test Environment',
            slug: 'test-env',
          },
        },
      };
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockData),
    });
  });
}

/**
 * Add a small delay to avoid rate limits between test runs
 */
export async function addRateLimitDelay(page: Page, ms: number = 500) {
  await page.waitForTimeout(ms);
}
