const { defineConfig, devices } = require('@playwright/test');

module.exports = defineConfig({
  testDir: './tests/e2e',
  timeout: 30_000,
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: process.env.CI ? [['line'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: 'http://127.0.0.1:18082',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    ...devices['Desktop Chrome'],
  },
  webServer: process.env.ABYSS_E2E_EXTERNAL_SERVER ? undefined : {
    command: 'go test -tags=e2e ./internal/bot -run TestAbyssE2EServer -count=1 -v -timeout=15m',
    url: 'http://127.0.0.1:18082/healthz',
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
});
