const { test, expect } = require('@playwright/test');

test('execution history shows active and ended logical sessions', async ({ page }) => {
  await page.goto('/history');

  await expect(page.getByRole('heading', { name: 'Execution history' })).toBeVisible();

  const live = page.locator('[data-session-id="e2e-live-codex"]');
  await expect(live).toHaveAttribute('data-session-active', 'true');
  await expect(live).toHaveAttribute('data-identity-kind', 'logical');
  await expect(live).toContainText('Codex · codex');
  await expect(live.locator('a.repo')).toHaveAttribute('href', /\/project\?id=repo-/);
  await expect(live.locator('a.session-detail')).toHaveAttribute('href', /\/history\/session\?repository=repo-.*&session=e2e-live-codex/);

  const ended = page.locator('[data-session-id="e2e-ended-claude"]');
  await expect(ended).toHaveAttribute('data-session-active', 'false');
  await expect(ended).toHaveAttribute('data-identity-kind', 'logical');
  await expect(ended).toContainText('Claude · claude-code');
  await expect(ended).toContainText('ended');
});

test('repository navigation scopes execution history and can return to all history', async ({ page }) => {
  await page.goto('/history');

  const live = page.locator('[data-session-id="e2e-live-codex"]');
  const repositoryLink = live.locator('a.repo');
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

test('execution session detail preserves exact logical session context', async ({ page }) => {
  await page.goto('/history');

  const live = page.locator('[data-session-id="e2e-live-codex"]');
  const detailLink = live.locator('a.session-detail');
  await expect(detailLink).toHaveAttribute('href', /\/history\/session\?repository=repo-.*&session=e2e-live-codex/);
  await detailLink.click();

  await expect(page).toHaveURL(/\/history\/session\?repository=repo-.*&session=e2e-live-codex/);
  await expect(page.locator('body')).toHaveAttribute('data-session-id', 'e2e-live-codex');
  await expect(page.locator('body')).toHaveAttribute('data-session-active', 'true');
  await expect(page.locator('body')).toHaveAttribute('data-identity-kind', 'logical');
  await expect(page.getByRole('heading', { name: 'Codex execution session' })).toBeVisible();

  const detail = page.locator('[data-specview="execution-session-detail"]');
  await expect(detail).toContainText('e2e-live-codex');
  await expect(detail).toContainText('4242, 4243');
  await expect(detail).toContainText('2026-08-23T11:55:00Z');
  await expect(detail).toContainText('2026-08-23T12:00:00Z');

  const hostViews = page.getByRole('navigation', { name: 'Host views' });
  await expect(hostViews.getByRole('link', { name: 'History' })).toHaveAttribute('aria-current', 'page');
  await expect(hostViews.getByRole('link', { name: 'History' })).toHaveAttribute('href', /\/history\?repository=repo-/);
  await expect(hostViews.getByRole('link', { name: 'Acceptance' })).toHaveAttribute('href', /\/project\/acceptance\?id=repo-/);

  await page.getByRole('link', { name: 'Repository history' }).click();
  await expect(page.locator('body')).toHaveAttribute('data-history-repository', /repo-/);
});
