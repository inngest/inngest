import { defineConfig, devices } from '@playwright/test';
import dotenv from 'dotenv';

// Load test environment variables
dotenv.config({ path: '.env.test.local' });

/**
 * @see https://playwright.dev/docs/test-configuration
 */
export default defineConfig({
  testDir: './tests',
  /* Run tests in files in parallel */
  fullyParallel: false, // Disable to reduce rate limit issues
  /* Fail the build on CI if you accidentally left test.only in the source code. */
  forbidOnly: !!process.env.CI,
  /* Retry on CI only */
  retries: process.env.CI ? 2 : 1,
  /* Limit workers to reduce rate limiting */
  workers: process.env.CI ? 1 : 2,
  /* Reporter to use. See https://playwright.dev/docs/test-reporters */
  reporter: 'html',
  /* Global timeout */
  timeout: 45000, // Increase timeout to 45s
  /* Shared settings for all the projects below. See https://playwright.dev/docs/api/class-testoptions. */
  use: {
    /* Base URL to use in actions like `await page.goto('/')`. */
    baseURL: 'http://localhost:5173',

    /* Collect trace when retrying the failed test. See https://playwright.dev/docs/trace-viewer */
    trace: 'on-first-retry',

    /* Capture screenshot for failed test.*/
    screenshot: 'only-on-failure',
  },

  /* Make environment variables available in tests */
  /* Note: Environment variables are loaded via dotenv.config() above */
  /* and will be available in process.env within test files */

  /* Configure projects for major browsers */
  projects: [
    // Setup project
    { name: 'setup', testMatch: /.*\.setup\.ts/ },

    // Authenticated tests
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        // Use signed-in state for all tests
        storageState: 'playwright/.auth/user.json',
      },
      dependencies: ['setup'],
    },

    {
      name: 'firefox',
      use: {
        ...devices['Desktop Firefox'],
        storageState: 'playwright/.auth/user.json',
      },
      dependencies: ['setup'],
    },

    {
      name: 'webkit',
      use: {
        ...devices['Desktop Safari'],
        storageState: 'playwright/.auth/user.json',
      },
      dependencies: ['setup'],
    },

    // Unauthenticated tests (for testing sign-in flow itself)
    {
      name: 'unauthenticated',
      use: { ...devices['Desktop Chrome'] },
      testMatch: /.*\.unauthenticated\.spec\.ts/,
    },

    /* Test against branded browsers. */
    // {
    //   name: 'Microsoft Edge',
    //   use: { ...devices['Desktop Edge'], channel: 'msedge' },
    // },
    // {
    //   name: 'Google Chrome',
    //   use: { ...devices['Desktop Chrome'], channel: 'chrome' },
    // },
  ],

  /* Run your local dev server before starting the tests */
  // webServer: {
  //   command: 'pnpm dev',
  //   url: 'http://localhost:5173',
  //   reuseExistingServer: !process.env.CI,
  // },
});
