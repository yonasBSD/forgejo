// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// templates/repo/diff/new_review.tmpl
// web_src/js/features/repo-issue.js
// @watch end

import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';
import {screenshot} from './shared/screenshots.ts';

test.use({user: 'user2'});

test('PR Commits: mobile responsive layout checks', async ({page}) => {
  const response = await page.goto('/user2/repo1/pulls/3/commits');
  expect(response?.status()).toBe(200);
  // mobile view
  await page.setViewportSize({width: 375, height: 667});

  // mobile-specific grid positioning
  await expect(page.locator('.commit-timeline .author').first()).toHaveCSS('grid-column-start', '1');
  await expect(page.locator('.commit-timeline .date').first()).toHaveCSS('grid-column-start', '2');
  await expect(page.locator('.commit-timeline .message').first()).toHaveCSS('grid-column-end', 'span 2');
  await expect(page.locator('.commit-timeline .mobile-shabox').first()).toHaveCSS('grid-column-start', '1');
  await expect(page.locator('.commit-timeline details').first()).toHaveCSS('grid-column-start', '2');

  // mobile-specific visibility test
  await expect(page.locator('.mobile-shabox').first()).toBeVisible();
  await expect(page.locator('.commit-group h4').first()).toBeVisible();
  await expect(page.locator('.shabox').first()).toBeHidden();
  await expect(page.locator('.commit-buttons').first()).toBeHidden();

  // horizontal scrolling to check for overflow
  await expect(page.locator('.commit-group-commits').first()).not.toHaveCSS('overflow-x', 'scroll');

  await screenshot(page);
});

test('PR Commits: desktop responsive layout checks', async ({page}) => {
  const response = await page.goto('/user2/repo1/pulls/3/commits');
  expect(response?.status()).toBe(200);
  // desktop view
  await page.setViewportSize({width: 1200, height: 800});

  // date group visibility test
  await expect(page.locator('.commit-group').first()).toBeVisible();
  await expect(page.locator('.commit-group h4').first()).toBeVisible();

  // desktop grid is the default 5‑column template; just assert it’s a grid
  await expect(page.locator('.commit-timeline').first()).toHaveCSS('display', 'grid');

  // desktop-specific visibility test
  await expect(page.locator('.mobile-shabox').first()).toBeHidden();
  await expect(page.locator('.shabox').first()).toBeVisible();
  await expect(page.locator('.commit-buttons').first()).toBeVisible();

  await screenshot(page);
});
