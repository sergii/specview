const { test, expect } = require('@playwright/test');

test('execution history shows active and ended logical sessions', async ({ page }) => {
  await page.goto('/history');

  await expect(page.getByRole('heading', { name: 'Execution history' })).toBeVisible();

  const live = page.locator('[data-session-id="e2e-live-codex"]');
  await expect(live).toHaveAttribute('data-session-active', 'true');
  await expect(live).toHaveAttribute('data-identity-kind', 'logical');
  await expect(live).toContainText('Codex · codex');
  await expect(live.getByRole('link')).toHaveAttribute('href', /\/project\?id=repo-/);

  const ended = page.locator('[data-session-id="e2e-ended-claude"]');
  await expect(ended).toHaveAttribute('data-session-active', 'false');
  await expect(ended).toHaveAttribute('data-identity-kind', 'logical');
  await expect(ended).toContainText('Claude · claude-code');
  await expect(ended).toContainText('ended');
});
