const { test, expect } = require('@playwright/test');

test('federation repository drill-down preserves local and remote authority', async ({ page }) => {
  await page.goto('/federation');

  const local = page.locator('[data-instance-id="instance:e2e-local"]');
  await expect(local).toBeVisible();
  await local.click();

  await expect(page).toHaveURL(/\/federation\/repository\?/);
  await expect(page.getByRole('heading', { name: 'sergii/specview' })).toBeVisible();
  await expect(page.locator('[data-federation-instance="instance:e2e-local"]')).toHaveAttribute('data-host-source', 'local');
  await expect(page.getByRole('link', { name: 'Federation', exact: true })).toHaveAttribute('aria-current', 'page');
  await expect(page.getByRole('link', { name: 'Open live local repository' })).toHaveAttribute('href', '/project?id=repo-e2e');

  await page.getByRole('link', { name: 'Federation', exact: true }).first().click();
  const remote = page.locator('[data-instance-id="instance:e2e-devbox"]');
  await expect(remote).toBeVisible();
  await remote.click();

  const shell = page.locator('[data-federation-instance="instance:e2e-devbox"]');
  await expect(shell).toHaveAttribute('data-host-source', 'peer');
  await expect(shell).toHaveAttribute('data-freshness', 'unreachable');
  await expect(page.getByText(/fixture transport unavailable/)).toBeVisible();
  await expect(page.getByRole('link', { name: 'Open live local repository' })).toHaveCount(0);
  await expect(page.getByText('No active sessions were present in this snapshot.')).toBeVisible();
  await expect(page.getByText('No worktree facts were present in this snapshot.')).toBeVisible();
});
