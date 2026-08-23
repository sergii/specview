const { test, expect } = require('@playwright/test');

test('host to work item preserves accepted evidence semantics', async ({ page }, testInfo) => {
  await page.goto('/');

  const repository = page.locator('a[href^="/project?id="]').first();
  await expect(repository).toBeVisible();
  await expect(repository).toContainText('repository');
  await repository.click();

  const workItem = page.locator('a[href^="/project/spec?"]').filter({ hasText: 'H17 Acceptance Policy' }).first();
  await expect(workItem).toBeVisible();
  await workItem.click();

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
    path: testInfo.outputPath('acceptance-detail.png'),
    fullPage: true,
  });
});
