const { test, expect } = require('@playwright/test');

test('shared host navigation makes top-level views discoverable', async ({ page }) => {
  await page.goto('/');

  let navigation = page.getByRole('navigation', { name: 'Host views' });
  await expect(navigation.getByRole('link', { name: 'Host' })).toHaveAttribute('aria-current', 'page');

  await navigation.getByRole('link', { name: 'History' }).click();
  await expect(page).toHaveURL(/\/history$/);
  await expect(page.getByRole('heading', { name: 'Execution history' })).toBeVisible();
  navigation = page.getByRole('navigation', { name: 'Host views' });
  await expect(navigation.getByRole('link', { name: 'History' })).toHaveAttribute('aria-current', 'page');

  await navigation.getByRole('link', { name: 'Federation' }).click();
  await expect(page).toHaveURL(/\/federation$/);
  await expect(page.getByRole('heading', { name: 'Federation' })).toBeVisible();
  navigation = page.getByRole('navigation', { name: 'Host views' });
  await expect(navigation.getByRole('link', { name: 'Federation' })).toHaveAttribute('aria-current', 'page');

  await navigation.getByRole('link', { name: 'Host' }).click();
  await expect(page).toHaveURL(/\/$/);
  navigation = page.getByRole('navigation', { name: 'Host views' });
  await expect(navigation.getByRole('link', { name: 'Host' })).toHaveAttribute('aria-current', 'page');
});
