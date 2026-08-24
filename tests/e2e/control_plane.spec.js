const { test, expect } = require('@playwright/test');

test('repository control plane composes Intent, Execution, Evidence, and Acceptance', async ({ page }) => {
  await page.goto('/history');
  await page.locator('[data-session-id="e2e-live-codex"] a.repo').click();

  const control = page.locator('[data-specview="repository-control-plane"]');
  await expect(control).toBeVisible();
  await expect(control).toHaveAttribute('data-intent-total', '1');
  await expect(control).toHaveAttribute('data-execution-active', '1');
  await expect(control).toHaveAttribute('data-evidence-total', '2');
  await expect(control).toHaveAttribute('data-acceptance-configured', 'true');

  const intent = control.locator('[data-control-facet="intent"]');
  await expect(intent).toHaveAttribute('href', '#specification');
  await expect(intent).toContainText('1 work item');
  await expect(intent).toContainText('1 in progress');

  const execution = control.locator('[data-control-facet="execution"]');
  await expect(execution).toHaveAttribute('href', /\/history\?repository=repo-/);
  await expect(execution).toContainText('1 active');
  await expect(execution).toContainText('Latest Codex · active');
  await expect(execution).toContainText('e2e-live-codex');

  const evidence = control.locator('[data-control-facet="evidence"]');
  await expect(evidence).toHaveAttribute('href', /\/project\/evidence\?id=repo-/);
  await expect(evidence).toContainText('2 records');
  await expect(evidence).toContainText('2 passed');
  await expect(evidence).toContainText('lint · e2e-fixture · passed');

  const acceptance = control.locator('[data-control-facet="acceptance"]');
  await expect(acceptance).toHaveAttribute('href', /\/project\/acceptance\?id=repo-/);
  await expect(acceptance).toContainText('1 accepted');
  await expect(acceptance).toContainText('0 waiting · 0 blocked');

  await evidence.click();
  await expect(page).toHaveURL(/\/project\/evidence\?id=repo-/);
  await expect(page.getByRole('heading', { name: /Evidence for/ })).toBeVisible();
});
