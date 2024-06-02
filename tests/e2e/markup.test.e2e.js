// @ts-check
import {test, expect} from '@playwright/test';
import {login_user, load_logged_in_context} from './utils_e2e.js';

test.beforeAll(async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user2');
});

test('Test markup with #xyz-mode-only', async ({browser}, workerInfo) => {
  const context = await load_logged_in_context(browser, workerInfo, 'user2');
  /** @type {import('@playwright/test').Page} */
  const page = await context.newPage();

  const response = await page.goto('/user2/repo1/issues/1');
  await expect(response?.status()).toBe(200);
  await page.getByPlaceholder('Leave a comment').fill('test markup with #xyz-mode-only: ![GitHub-Mark-Light](https://user-images.githubusercontent.com/3369400/139447912-e0f43f33-6d9f-45f8-be46-2df5bbc91289.png#gh-dark-mode-only)![GitHub-Mark-Dark](https://user-images.githubusercontent.com/3369400/139448065-39a229ba-4b06-434b-bc67-616e2ed80c8f.png#gh-light-mode-only)');
  await page.locator('form button.ui.primary.button:visible').click();
  await page.waitForLoadState('networkidle');

  const comment = page.locator('.comment-body>.markup', {hasText: 'test markup with #xyz-mode-only:'});
  await expect(comment).toBeVisible();
  await expect(comment.locator('[src$="#gh-light-mode-only"]')).toBeVisible();
  await expect(comment.locator('[src$="#gh-dark-mode-only"]')).not.toBeVisible();
});
