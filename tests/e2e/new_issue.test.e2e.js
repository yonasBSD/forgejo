// @ts-check
import {test, expect} from '@playwright/test';
import {login_user, load_logged_in_context} from './utils_e2e.js';

test.beforeAll(async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user2');
});

async function login({browser}, workerInfo) {
  const context = await load_logged_in_context(browser, workerInfo, 'user2');
  return await context.newPage();
}

test('Create issue with attachments', async ({browser}, workerInfo) => {
  const page = await login({browser}, workerInfo);
  if (workerInfo.project.name !== 'firefox') {
     await page.context().grantPermissions(['clipboard-read', 'clipboard-write']);
  }
  const response = await page.goto('/user2/repo1/issues/new');
  await expect(response?.status()).toBe(200);

  await page.locator('#issue_title').fill('A new issue!');
  await page.locator('.markdown-text-editor').fill('I think I found a bug, please see the following screenshot.\n');

  const fileChooserPromise = page.waitForEvent('filechooser');
  await page.locator('.dz-button').click();
  const fileChooser = await fileChooserPromise;
  await fileChooser.setFiles({
    name: 'file.svg',
    mimeType: 'image/svg+xml',
    buffer: Buffer.from('<svg width="300" height="200" xmlns="http://www.w3.org/2000/svg"><rect width="100%" height="100%" fill="red" /><circle cx="150" cy="100" r="80" fill="green" /><text x="150" y="125" font-size="60" text-anchor="middle" fill="white">Forgejo</text></svg>'),
  });

  await page.getByText('Copy link').click();
  // eslint-disable-next-line no-undef
  const handle = await page.evaluateHandle(() => navigator.clipboard.readText());
  const clipboardContent = await handle.jsonValue();
  await expect(clipboardContent).toContain('![file.svg](/attachments/');
  const attachmentURL = clipboardContent.substring(clipboardContent.indexOf('](/') + 2, clipboardContent.length - 1);

  await page.locator('.markdown-text-editor').focus();
  await page.keyboard.press('Control+KeyV');
  await expect(await page.locator('.markdown-text-editor').inputValue()).toBe(`I think I found a bug, please see the following screenshot.\n${clipboardContent}`);

  // Ensure it can be loaded on the preview tab.
  await page.getByText('Preview').click();
  await page.waitForLoadState('networkidle');
  await expect(page.locator(`img[src$="/user2/repo1${attachmentURL}"]`)).toBeVisible();

  await page.getByText('Create issue').click();
  await page.waitForLoadState('domcontentloaded');

  await expect(page.locator(`img[src$="/user2/repo1${attachmentURL}"]`)).toBeVisible();
  await expect(page.locator('#issue-title-display')).toContainText('A new issue!');
  await expect(page.locator('.comment-body')).toContainText('I think I found a bug, please see the following screenshot.');
});
