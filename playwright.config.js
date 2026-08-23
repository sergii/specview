const { defineConfig } = require('@playwright/test');

module.exports = defineConfig({
  testDir: './tests/e2e',
  timeout: 30_000,
  expect: {
    timeout: 5_000,
  },
  use: {
    baseURL: 'http://127.0.0.1:7332',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  reporter: process.env.CI ? [['line']] : [['list']],
  webServer: {
    command: 'go run ./testsupport/e2e-server',
    url: 'http://127.0.0.1:7332/healthz',
    timeout: 120_000,
    reuseExistingServer: !process.env.CI,
  },
});
