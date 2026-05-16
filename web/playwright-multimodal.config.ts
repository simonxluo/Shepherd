import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  testMatch: 'multimodal-pages.spec.ts',
  fullyParallel: false,
  retries: 0,
  workers: 1,
  outputDir: 'test-results/artifacts',
  reporter: [['list'], ['html', { open: 'never', outputFolder: 'test-results/playwright-report' }]],
  use: {
    baseURL: process.env.BASE_URL || 'http://localhost:9190',
    trace: 'on',
    screenshot: 'on',
    video: 'off',
    actionTimeout: 15000,
    navigationTimeout: 30000,
  },
  projects: [
    {
      name: 'chromium',
      use: {
        channel: 'chrome',
        viewport: { width: 1440, height: 900 },
        launchOptions: {
          args: ['--no-sandbox', '--disable-gpu', '--disable-dev-shm-usage'],
        },
      },
    },
  ],
});
