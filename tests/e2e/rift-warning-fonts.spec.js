const { test, expect } = require('@playwright/test');

test.describe('Keep important warnings readable with custom system fonts (Proposal 0154)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('warning elements meet minimum font size and responsive line-height requirements', async ({ page }) => {
    const atRisk = page.locator('.rift-at-risk');
    await expect(atRisk).toBeVisible();

    const atRiskStyles = await atRisk.evaluate((el) => {
      const cs = window.getComputedStyle(el);
      return {
        fontSize: parseFloat(cs.fontSize),
        lineHeight: parseFloat(cs.lineHeight),
        overflowWrap: cs.overflowWrap,
      };
    });
    // Font size should be at least 10px (not tiny 8px micro-text)
    expect(atRiskStyles.fontSize).toBeGreaterThanOrEqual(10);
    // Line height should allow comfortable vertical breathing room (>= 14px for >= 10px text)
    expect(atRiskStyles.lineHeight).toBeGreaterThanOrEqual(14);
    expect(['anywhere', 'break-word'].includes(atRiskStyles.overflowWrap)).toBe(true);

    const hpThreshold = page.locator('#rift-vitals .rift-hp-threshold');
    const hpStyles = await hpThreshold.evaluate((el) => {
      const cs = window.getComputedStyle(el);
      return {
        fontSize: parseFloat(cs.fontSize),
        lineHeight: parseFloat(cs.lineHeight),
        overflowWrap: cs.overflowWrap,
      };
    });
    expect(hpStyles.fontSize).toBeGreaterThanOrEqual(10);
    expect(hpStyles.lineHeight).toBeGreaterThanOrEqual(14);
  });

  test('important warnings remain fully visible and non-clipped when custom system fonts and dyslexia spacing are applied', async ({ page }) => {
    // Inject a custom system font with enhanced spacing (simulating Atkinson Hyperlegible / OpenDyslexic user stylesheets)
    await page.addStyleTag({
      content: `
        #rift-app, #rift-app * {
          font-family: 'OpenDyslexic', 'Comic Sans MS', system-ui, sans-serif !important;
          letter-spacing: 0.12em !important;
          word-spacing: 0.16em !important;
        }
      `,
    });

    // 1. Verify expedition bag "AT RISK" warning badge
    const atRisk = page.locator('.rift-at-risk');
    await expect(atRisk).toBeVisible();
    await expect(atRisk).toHaveText(/AT RISK/i);
    const atRiskBox = await atRisk.boundingBox();
    expect(atRiskBox).not.toBeNull();
    expect(atRiskBox.width).toBeGreaterThan(30);
    expect(atRiskBox.height).toBeGreaterThan(14);

    // 2. Verify health threshold warning badge
    await page.evaluate(() => {
      document.getElementById('rift-vitals').hidden = false;
      document.getElementById('rift-overlay').hidden = true;
    });
    const hpThreshold = page.locator('#rift-vitals .rift-hp-threshold');
    await hpThreshold.evaluate((el) => {
      el.dataset.threshold = 'critical';
      el.textContent = '⚠ Critical';
    });
    await expect(hpThreshold).toBeVisible();
    await expect(hpThreshold).toHaveText(/Critical/);
    const hpBox = await hpThreshold.boundingBox();
    expect(hpBox.width).toBeGreaterThan(40);
    expect(hpBox.height).toBeGreaterThan(14);

    // 3. Verify paused warning badge
    const pausedBadge = page.locator('#rift-paused-badge');
    await pausedBadge.evaluate((el) => {
      el.hidden = false;
      el.textContent = 'Paused (Tactical)';
    });
    await expect(pausedBadge).toBeVisible();
    const pausedBox = await pausedBadge.boundingBox();
    expect(pausedBox.width).toBeGreaterThan(40);
    expect(pausedBox.height).toBeGreaterThan(14);

    // 4. Verify defeat guidance warning panel
    const defeatGuide = page.locator('#rift-defeat-guide');
    await defeatGuide.evaluate((el) => { el.hidden = false; });
    await expect(defeatGuide).toBeVisible();
    const defeatTitle = defeatGuide.locator('#rift-defeat-guide-title');
    await expect(defeatTitle).toBeVisible();
    const defeatBox = await defeatGuide.boundingBox();
    expect(defeatBox.height).toBeGreaterThan(60);

    // 5. Verify checkpoint guidance warning
    const checkpointGuide = page.locator('#rift-checkpoint-guide');
    await checkpointGuide.evaluate((el) => {
      el.hidden = false;
      el.querySelector('#rift-checkpoint-guide-copy').textContent = 'Room cleared · Bank rewards now or risk them in the next room.';
    });
    await expect(checkpointGuide).toBeVisible();
    const checkpointBox = await checkpointGuide.boundingBox();
    expect(checkpointBox.height).toBeGreaterThan(30);

    // 6. Verify keybind conflict warning in controls dialog
    await page.locator('#rift-controls-open').click();
    const dialog = page.locator('#rift-controls-dialog');
    await expect(dialog).toBeVisible();
    const bindingStatus = page.locator('#rift-binding-status');
    await bindingStatus.evaluate((el) => {
      el.textContent = 'Warning: Key conflict detected. Escape cancels remapping.';
    });
    await expect(bindingStatus).toBeVisible();
    const statusBox = await bindingStatus.boundingBox();
    expect(statusBox.height).toBeGreaterThan(20);

    // 7. Verify document does not suffer horizontal scroll overflow with expanded custom font metrics
    const overflows = await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 2);
    expect(overflows).toBe(true);
  });

  test('warnings remain readable on narrow viewports with enlarged system fonts', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.addStyleTag({
      content: `
        #rift-app, #rift-app * {
          font-family: 'OpenDyslexic', 'Comic Sans MS', sans-serif !important;
          font-size: 110% !important;
          line-height: 1.5 !important;
        }
      `,
    });

    const atRisk = page.locator('.rift-at-risk');
    await expect(atRisk).toBeVisible();
    const box = await atRisk.boundingBox();
    expect(box.width).toBeGreaterThan(30);
    expect(box.height).toBeGreaterThan(14);

    const overflows = await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 2);
    expect(overflows).toBe(true);
  });
});
