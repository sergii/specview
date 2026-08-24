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

test('repository navigation scopes execution history and can return to all history', async ({ page }) => {
  await page.goto('/history');

  const live = page.locator('[data-session-id="e2e-live-codex"]');
  const repositoryLink = live.getByRole('link');
  const repositoryHref = await repositoryLink.getAttribute('href');
  expect(repositoryHref).toMatch(/^\/project\?id=repo-/);
  await repositoryLink.click();
  await expect(page).toHaveURL(new RegExp(repositoryHref.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));

  const hostViews = page.getByRole('navigation', { name: 'Host views' });
  const historyLink = hostViews.getByRole('link', { name: 'History' });
  await expect(historyLink).toHaveAttribute('href', /\/history\?repository=repo-/);
  await historyLink.click();

  await expect(page.locator('body')).toHaveAttribute('data-history-repository', /repo-/);
  await expect(page.getByRole('heading', { name: /Execution history for/ })).toBeVisible();
  await expect(page.locator('[data-session-id="e2e-live-codex"]')).toBeVisible();
  await expect(page.locator('[data-session-id="e2e-ended-claude"]')).toBeVisible();
  await expect(hostViews.getByRole('link', { name: 'History' })).toHaveAttribute('aria-current', 'page');

  await page.getByRole('link', { name: 'All history' }).click();
  await expect(page).toHaveURL(/\/history$/);
  await expect(page.locator('body')).not.toHaveAttribute('data-history-repository', /.+/);
});
