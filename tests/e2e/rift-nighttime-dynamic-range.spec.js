const { test, expect } = require('@playwright/test');

test.describe('Provide a nighttime dynamic-range preset (Proposal 0196)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await page.evaluate(() => {
      localStorage.removeItem('riftAudio:nightMode');
      localStorage.removeItem('riftAudio:dynamicRange');
    });
    await page.reload();
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('default dynamic range is standard and adjusts limiter compressor properties', async ({ page }) => {
    const data = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.unlock();
      const limiter = audio.limiter;
      return {
        nightMode: audio.nightMode,
        dynamicRange: audio.dynamicRange,
        threshold: limiter ? limiter.threshold.value : null,
        ratio: limiter ? limiter.ratio.value : null,
        knee: limiter ? limiter.knee.value : null,
      };
    });

    expect(data.nightMode).toBe(false);
    expect(data.dynamicRange).toBe('standard');
    expect(data.threshold).toBeCloseTo(-16, 0);
    expect(data.ratio).toBeCloseTo(8, 0);
    expect(data.knee).toBeCloseTo(30, 0);
  });

  test('enabling nightMode or dynamicRange night clamps threshold and increases ratio', async ({ page }) => {
    const data = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.unlock();
      audio.applyDynamicRangePreset('night');

      const nightValues = {
        nightMode: audio.nightMode,
        dynamicRange: audio.dynamicRange,
        threshold: audio.limiter ? audio.limiter.threshold.value : null,
        ratio: audio.limiter ? audio.limiter.ratio.value : null,
        knee: audio.limiter ? audio.limiter.knee.value : null,
      };

      audio.applyDynamicRangePreset('standard');

      const standardValues = {
        nightMode: audio.nightMode,
        dynamicRange: audio.dynamicRange,
        threshold: audio.limiter ? audio.limiter.threshold.value : null,
        ratio: audio.limiter ? audio.limiter.ratio.value : null,
        knee: audio.limiter ? audio.limiter.knee.value : null,
      };

      return { nightValues, standardValues };
    });

    expect(data.nightValues.nightMode).toBe(true);
    expect(data.nightValues.dynamicRange).toBe('night');
    expect(data.nightValues.threshold).toBeCloseTo(-28, 0);
    expect(data.nightValues.ratio).toBeCloseTo(14, 0);
    expect(data.nightValues.knee).toBeCloseTo(10, 0);

    expect(data.standardValues.nightMode).toBe(false);
    expect(data.standardValues.dynamicRange).toBe('standard');
    expect(data.standardValues.threshold).toBeCloseTo(-16, 0);
    expect(data.standardValues.ratio).toBeCloseTo(8, 0);
  });

  test('UI checkbox #rift-night-mode toggles nightMode and updates settings summary', async ({ page }) => {
    const details = page.locator('.rift-settings');
    await details.evaluate(node => { node.open = true; });

    const nightCheckbox = page.locator('#rift-night-mode');
    await expect(nightCheckbox).toBeVisible();
    await expect(nightCheckbox).not.toBeChecked();

    const summary = page.locator('#rift-settings-summary');
    await expect(summary).toContainText('Standard range');

    // Click checkbox to enable night mode
    await nightCheckbox.check();

    const enabledState = await page.evaluate(() => {
      const audio = window.RiftAudio;
      return {
        nightMode: audio.nightMode,
        dynamicRange: audio.dynamicRange,
        storedNight: localStorage.getItem('riftAudio:nightMode'),
        storedRange: localStorage.getItem('riftAudio:dynamicRange'),
      };
    });

    expect(enabledState.nightMode).toBe(true);
    expect(enabledState.dynamicRange).toBe('night');
    expect(enabledState.storedNight).toBe('true');
    expect(enabledState.storedRange).toBe('"night"');
    await expect(summary).toContainText('Night range');

    // Reload page to verify persistence
    await page.reload();
    await details.evaluate(node => { node.open = true; });
    await expect(nightCheckbox).toBeChecked();
    await expect(summary).toContainText('Night range');
  });

  test('reset audio mix restores standard dynamic range', async ({ page }) => {
    const details = page.locator('.rift-settings');
    await details.evaluate(node => { node.open = true; });

    const nightCheckbox = page.locator('#rift-night-mode');
    await nightCheckbox.check();
    await expect(nightCheckbox).toBeChecked();

    const resetBtn = page.locator('#rift-reset-audio');
    await resetBtn.click();

    await expect(nightCheckbox).not.toBeChecked();
    const state = await page.evaluate(() => {
      return {
        nightMode: window.RiftAudio.nightMode,
        dynamicRange: window.RiftAudio.dynamicRange,
      };
    });
    expect(state.nightMode).toBe(false);
    expect(state.dynamicRange).toBe('standard');
  });
});
