const { test, expect } = require('@playwright/test');

test('host control plane composes repositories into four read-only facets', async ({ page }) => {
  await page.goto('/');

  const control = page.locator('[data-specview="host-control-plane"]');
  await expect(control).toBeVisible();
  await expect(control).toHaveAttribute('data-intent-work-items', '1');
  await expect(control).toHaveAttribute('data-execution-active', '1');
  await expect(control).toHaveAttribute('data-evidence-failed', '0');
  await expect(control).toHaveAttribute('data-acceptance-blocked', '0');
  await expect(control).toHaveAttribute('data-attention-count', '0');

  await expect(control.locator('[data-control-facet="intent"]')).toContainText('1 work items');

  const execution = control.locator('[data-control-facet="execution"]');
  await expect(execution).toHaveAttribute('href', '/history');
  await expect(execution).toContainText('1 active sessions');
  await expect(execution).toContainText('1 active repositories');
  await expect(execution).toContainText('· Codex ·');

  await expect(control.locator('[data-control-facet="evidence"]')).toContainText('2 records');
  await expect(control.locator('[data-control-facet="evidence"]')).toContainText('2 passed · 0 failed · 0 invalid');

  await expect(control.locator('[data-control-facet="acceptance"]')).toContainText('0 blocked · 0 waiting');
  await expect(control.locator('[data-control-facet="acceptance"]')).toContainText('1 accepted · 1 configured repositories');
  await expect(page.locator('[data-specview="host-attention"]')).toContainText('No failed or invalid Evidence');

  await execution.click();
  await expect(page).toHaveURL(/\/history$/);
  await expect(page.getByRole('heading', { name: 'Execution history' })).toBeVisible();
});
