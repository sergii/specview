const { test, expect } = require('@playwright/test');

test('repository Evidence is a first-class read-only view', async ({ page }) => {
  await page.goto('/history');

  const repositoryLink = page.locator('[data-session-id="e2e-live-codex"] a.repo');
  await repositoryLink.click();

  const hostViews = page.getByRole('navigation', { name: 'Host views' });
  const evidenceLink = hostViews.getByRole('link', { name: 'Evidence' });
  await expect(evidenceLink).toHaveAttribute('href', /\/project\/evidence\?id=repo-/);
  await evidenceLink.click();

  await expect(page).toHaveURL(/\/project\/evidence\?id=repo-/);
  await expect(page.getByRole('heading', { name: /Evidence for/ })).toBeVisible();
  await expect(page.locator('body')).toHaveAttribute('data-evidence-repository', /repo-/);
  await expect(hostViews.getByRole('link', { name: 'Evidence' })).toHaveAttribute('aria-current', 'location');

  const overview = page.locator('[data-specview="evidence-overview"]');
  await expect(overview).toHaveAttribute('data-evidence-count', '2');
  await expect(overview).toHaveAttribute('data-passed', '2');

  const unitTests = page.locator('[data-evidence-id="H17-unit-tests-e2e"]');
  await expect(unitTests).toHaveAttribute('data-evidence-result', 'passed');
  await expect(unitTests).toContainText('H17 Acceptance Policy');
  await expect(unitTests).toContainText('unit-tests');
  await expect(unitTests).toContainText('git:abc123');
  await expect(unitTests).toContainText('e2e-fixture · test');

  await unitTests.getByRole('link').click();
  await expect(page).toHaveURL(/\/project\/spec\?id=repo-.*&path=H17\.md/);
  await expect(page.getByRole('heading', { name: 'H17 Acceptance Policy' })).toBeVisible();
  await expect(hostViews.getByRole('link', { name: 'Evidence' })).toHaveAttribute('href', /\/project\/evidence\?id=repo-/);
});
