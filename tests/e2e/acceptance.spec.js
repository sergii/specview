const { test, expect } = require('@playwright/test');

test('host to repository acceptance overview preserves accepted evidence semantics', async ({ page }, testInfo) => {
  await page.goto('/');

  const repository = page.locator('a[href^="/project?id="]').first();
  await expect(repository).toBeVisible();
  await expect(repository).toContainText('repository');
  await repository.click();

  const hostViews = page.getByRole('navigation', { name: 'Host views' });
  const acceptanceLink = hostViews.getByRole('link', { name: 'Acceptance', exact: true });
  await expect(acceptanceLink).toHaveAttribute('href', /\/project\/acceptance\?id=repo-/);
  await acceptanceLink.click();

  const overview = page.locator('[data-specview="acceptance-overview"]');
  await expect(overview).toBeVisible();
  await expect(overview).toHaveAttribute('data-acceptance-configured', 'true');
  await expect(overview).toHaveAttribute('data-accepted', '1');
  await expect(overview).toHaveAttribute('data-waiting', '0');
  await expect(overview).toHaveAttribute('data-blocked', '0');
  await expect(overview).toHaveAttribute('data-evidence-count', '2');
  await expect(acceptanceLink).toHaveAttribute('aria-current', 'location');

  const workItem = overview.locator('[data-work-item="H17"]');
  await expect(workItem).toHaveAttribute('data-acceptance-state', 'accepted');
  await expect(workItem).toContainText('2 evidence records');
  await workItem.getByRole('link').click();

  await expect(page.getByRole('heading', { name: 'H17 Acceptance Policy' })).toBeVisible();

  const acceptance = page.locator('[data-specview="acceptance"]');
  await expect(acceptance).toBeVisible();
  await expect(acceptance).toHaveAttribute('data-acceptance-state', 'accepted');
  await expect(acceptance).toContainText('git:abc123');
  await expect(acceptance).toContainText('2 evidence records');

  const unitTests = acceptance.locator('[data-check="unit-tests"]');
  await expect(unitTests).toHaveAttribute('data-check-state', 'passed');
  await expect(unitTests).toContainText('e2e-fixture');

  const lint = acceptance.locator('[data-check="lint"]');
  await expect(lint).toHaveAttribute('data-check-state', 'passed');
  await expect(lint).toContainText('e2e-fixture');

  await page.screenshot({
    path: testInfo.outputPath('acceptance-overview-detail.png'),
    fullPage: true,
  });
});
