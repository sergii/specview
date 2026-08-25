const { test, expect } = require('@playwright/test');

test('federation page preserves peer freshness, source attribution, and per-Host control plane', async ({ page }) => {
  await page.goto('/federation');

  await expect(page.getByRole('heading', { name: 'Federation' })).toBeVisible();

  const local = page.locator('[data-host-source="local"]');
  await expect(local).toContainText('e2e-laptop');
  await expect(local).toHaveAttribute('data-host-control-plane', 'available');
  await expect(local.locator('.host-control-plane')).toHaveAttribute('data-active-sessions', '1');
  await expect(local.locator('.host-control-plane')).toHaveAttribute('data-failed-evidence', '0');
  await expect(local.locator('.host-control-plane')).toHaveAttribute('data-blocked-acceptance', '0');
  await expect(local.locator('.host-control-plane')).toHaveAttribute('data-attention-count', '0');

  const unreachable = page.locator('[data-freshness="unreachable"]');
  await expect(unreachable).toContainText('e2e-devbox');
  await expect(unreachable).toContainText('snapshot available');
  await expect(unreachable).toContainText('fixture transport unavailable');
  await expect(unreachable).toHaveAttribute('data-host-control-plane', 'available');
  await expect(unreachable.locator('.host-control-plane')).toHaveAttribute('data-active-sessions', '2');
  await expect(unreachable.locator('.host-control-plane')).toHaveAttribute('data-failed-evidence', '1');
  await expect(unreachable.locator('.host-control-plane')).toHaveAttribute('data-blocked-acceptance', '1');
  await expect(unreachable.locator('.host-control-plane')).toHaveAttribute('data-attention-count', '1');
  await expect(unreachable).toHaveAttribute('href', '/federation/host?host=host%3A550e8400-e29b-41d4-a716-446655440001');

  const neverRetrieved = page.locator('[data-freshness="never_retrieved"]');
  await expect(neverRetrieved).toContainText('newbox');
  await expect(neverRetrieved).toContainText('no snapshot yet');
  await expect(neverRetrieved).toHaveAttribute('data-host-control-plane', 'unavailable');
  await expect(neverRetrieved).toContainText('control plane unavailable');

  const group = page.locator('[data-group-id="group:e2e-specview"]');
  await expect(group).toContainText('sergii/specview');
  await expect(group.locator('[data-host-id="host:550e8400-e29b-41d4-a716-446655440000"]')).toContainText('/specview-e2e/repository');
  await expect(group.locator('[data-host-id="host:550e8400-e29b-41d4-a716-446655440001"]')).toContainText('/srv/repos/sergii/specview');

  await unreachable.click();
  const hostDetail = page.locator('[data-federation-host="host:550e8400-e29b-41d4-a716-446655440001"]');
  await expect(hostDetail).toBeVisible();
  await expect(hostDetail).toHaveAttribute('data-host-source', 'peer');
  await expect(hostDetail).toHaveAttribute('data-freshness', 'unreachable');
  await expect(hostDetail).toHaveAttribute('data-host-control-plane', 'available');
  await expect(page.getByRole('heading', { name: 'e2e-devbox' })).toBeVisible();
  await expect(hostDetail).toContainText('fixture transport unavailable');
  await expect(hostDetail.locator('[data-facet="execution"] .metric', { hasText: 'active sessions' }).locator('strong')).toHaveText('2');
  await expect(hostDetail.locator('[data-facet="evidence"] .metric', { hasText: 'failed/error' }).locator('strong')).toHaveText('1');
  await expect(hostDetail.locator('[data-facet="acceptance"] .metric', { hasText: 'blocked' }).locator('strong')).toHaveText('1');
  await expect(hostDetail.getByText('Needs attention')).toBeVisible();

  const remoteRepository = hostDetail.locator('[data-instance-id]').filter({ hasText: '/srv/repos/sergii/specview' });
  await expect(remoteRepository).toBeVisible();
  await remoteRepository.click();
  await expect(page.locator('[data-federation-instance]')).toHaveAttribute('data-host-source', 'peer');
  await expect(page.getByText('Remote last-known repository snapshot')).toBeVisible();
});
