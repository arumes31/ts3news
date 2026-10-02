const { test, expect } = require('@playwright/test');

test('displays non-color health thresholds with notches, patterns, symbols, and accessible text', async ({ page }) => {
  let playerHpRatio = 1.0;

  await page.route('**/api/abyss/rift', async (route) => {
    const response = await route.fetch();
    const json = await response.json();
    if (json.run && json.run.player) {
      json.run.player.hp = json.run.player.max_hp * playerHpRatio;
    }
    await route.fulfill({ response, json });
  });

  await page.goto('/abyss/rift?subclass=geomancer');
  await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-start').click();
  await expect(page.locator('#rift-overlay')).toBeHidden();
  await expect(page.locator('#rift-enemy-count')).toContainText('remaining');

  const hpBadge = page.locator('#rift-hp-threshold');
  const hpMeter = page.locator('#rift-vitals .hp');

  // 1. Initial healthy state (> 50%)
  await expect(hpBadge).toBeVisible();
  await expect(hpBadge).toHaveText('✓ Healthy');
  await expect(hpBadge).toHaveAttribute('data-threshold', 'healthy');
  await expect(hpMeter).toHaveAttribute('data-threshold', 'healthy');
  await expect(hpMeter).toHaveAttribute('aria-valuetext', /Healthy/);

  // Helper unit tests for threshold boundaries
  const thresholdChecks = await page.evaluate(() => {
    const hud = window.RiftHUD;
    return {
      healthy: hud.getHealthThreshold(80, 100),
      woundedBoundary: hud.getHealthThreshold(50, 100),
      wounded: hud.getHealthThreshold(35, 100),
      criticalBoundary: hud.getHealthThreshold(25, 100),
      critical: hud.getHealthThreshold(10, 100),
      zero: hud.getHealthThreshold(0, 100),
    };
  });

  expect(thresholdChecks.healthy).toEqual({ state: 'healthy', label: 'Healthy', symbol: '✓' });
  expect(thresholdChecks.woundedBoundary).toEqual({ state: 'wounded', label: 'Wounded', symbol: '◆' });
  expect(thresholdChecks.wounded).toEqual({ state: 'wounded', label: 'Wounded', symbol: '◆' });
  expect(thresholdChecks.criticalBoundary).toEqual({ state: 'critical', label: 'Critical', symbol: '⚠' });
  expect(thresholdChecks.critical).toEqual({ state: 'critical', label: 'Critical', symbol: '⚠' });
  expect(thresholdChecks.zero).toEqual({ state: 'critical', label: 'Critical', symbol: '⚠' });

  // 2. Transition to Wounded state (35% HP)
  playerHpRatio = 0.35;
  await expect(hpBadge).toHaveText('◆ Wounded');
  await expect(hpBadge).toHaveAttribute('data-threshold', 'wounded');
  await expect(hpMeter).toHaveAttribute('data-threshold', 'wounded');
  await expect(hpMeter).toHaveAttribute('aria-valuetext', /Wounded/);

  // Verify pattern fill is applied for Wounded state
  const woundedFillBg = await page.locator('#rift-hp-fill').evaluate((el) => {
    return window.getComputedStyle(el).backgroundImage;
  });
  expect(woundedFillBg).toContain('repeating-linear-gradient');

  // 3. Transition to Critical state (15% HP)
  playerHpRatio = 0.15;
  await expect(hpBadge).toHaveText('⚠ Critical');
  await expect(hpBadge).toHaveAttribute('data-threshold', 'critical');
  await expect(hpMeter).toHaveAttribute('data-threshold', 'critical');
  await expect(hpMeter).toHaveAttribute('aria-valuetext', /Critical/);

  // Verify dashed outline for Critical state
  const criticalOutline = await hpMeter.evaluate((el) => {
    const style = window.getComputedStyle(el);
    return {
      style: style.outlineStyle,
      color: style.outlineColor,
    };
  });
  expect(criticalOutline.style).toBe('dashed');

  const criticalFillBg = await page.locator('#rift-hp-fill').evaluate((el) => {
    return window.getComputedStyle(el).backgroundImage;
  });
  expect(criticalFillBg).toContain('repeating-linear-gradient');
});

test('displays non-color health thresholds on boss meter during encounters', async ({ page }) => {
  let bossHpRatio = 1.0;

  await page.route('**/api/abyss/rift', async (route) => {
    const response = await route.fetch();
    const json = await response.json();
    if (json.run && json.run.enemies) {
      const boss = json.run.enemies.find(e => e.kind === 'boss');
      if (boss) {
        boss.hp = boss.max_hp * bossHpRatio;
      }
    }
    await route.fulfill({ response, json });
  });

  await page.goto('/abyss/rift?scenario=boss-windup');
  await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-start').click();

  const bossSection = page.locator('#rift-boss');
  await expect(bossSection).toBeVisible();

  const bossBadge = page.locator('#rift-boss-threshold');
  const bossMeter = page.locator('#rift-boss .hp');

  // 1. Initial boss state (healthy)
  await expect(bossBadge).toBeVisible();
  await expect(bossBadge).toHaveText('✓ Healthy');
  await expect(bossBadge).toHaveAttribute('data-threshold', 'healthy');
  await expect(bossMeter).toHaveAttribute('data-threshold', 'healthy');
  await expect(bossMeter).toHaveAttribute('aria-valuetext', /Healthy/);

  // 2. Boss at wounded HP (40%)
  bossHpRatio = 0.40;
  await expect(bossBadge).toHaveText('◆ Wounded');
  await expect(bossBadge).toHaveAttribute('data-threshold', 'wounded');
  await expect(bossMeter).toHaveAttribute('data-threshold', 'wounded');
  await expect(bossMeter).toHaveAttribute('aria-valuetext', /Wounded/);

  // 3. Boss at critical HP (10%)
  bossHpRatio = 0.10;
  await expect(bossBadge).toHaveText('⚠ Critical');
  await expect(bossBadge).toHaveAttribute('data-threshold', 'critical');
  await expect(bossMeter).toHaveAttribute('data-threshold', 'critical');
  await expect(bossMeter).toHaveAttribute('aria-valuetext', /Critical/);

  // Clean screenshot hides boss vitals
  const screenshotBtn = page.locator('#rift-screenshot-toggle');
  await screenshotBtn.click();
  await expect(bossSection).not.toBeVisible();
});

test('mobile layout at 390px does not cause horizontal overflow with health threshold badges', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/abyss/rift?subclass=geomancer');
  await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-start').click();
  await expect(page.locator('#rift-overlay')).toBeHidden();

  const hpBadge = page.locator('#rift-hp-threshold');
  await expect(hpBadge).toBeVisible();

  const isOverflowing = await page.evaluate(() => {
    return document.documentElement.scrollWidth > window.innerWidth;
  });
  expect(isOverflowing).toBe(false);
});
