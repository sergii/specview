const { test, expect } = require('@playwright/test');

test('federation page preserves peer freshness and source attribution', async ({ page }) => {
  await page.goto('/federation');

  await expect(page.getByRole('heading', { name: 'Federation' })).toBeVisible();
  await expect(page.locator('[data-host-source="local"]')).toContainText('e2e-laptop');

  const unreachable = page.locator('[data-freshness="unreachable"]');
  await expect(unreachable).toContainText('e2e-devbox');
  await expect(unreachable).toContainText('snapshot available');
  await expect(unreachable).toContainText('fixture transport unavailable');

  const neverRetrieved = page.locator('[data-freshness="never_retrieved"]');
  await expect(neverRetrieved).toContainText('newbox');
  await expect(neverRetrieved).toContainText('no snapshot yet');

  const group = page.locator('[data-group-id="group:e2e-specview"]');
  await expect(group).toContainText('sergii/specview');
  await expect(group.locator('[data-host-id="host:550e8400-e29b-41d4-a716-446655440000"]')).toContainText('/specview-e2e/repository');
  await expect(group.locator('[data-host-id="host:550e8400-e29b-41d4-a716-446655440001"]')).toContainText('/srv/repos/sergii/specview');
});
