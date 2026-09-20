const { test, expect } = require('@playwright/test');

test.describe('Provide a streamer-friendly procedural-music preset (Proposal 0197)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await page.evaluate(() => {
      localStorage.removeItem('riftAudio:streamerMusic');
      localStorage.removeItem('riftAudio:musicPreset');
    });
    await page.reload();
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('default preset has streamerMusic off and music bus active at authored volume', async ({ page }) => {
    const data = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.unlock();
      return {
        streamerMusic: audio.streamerMusic,
        musicPreset: audio.musicPreset,
        savedVolume: audio.music,
        channelLevel: audio.channelLevel('music'),
        busGain: audio.musicBus ? audio.musicBus.gain.value : null,
      };
    });

    expect(data.streamerMusic).toBe(false);
    expect(data.musicPreset).toBe('standard');
    expect(data.savedVolume).toBeCloseTo(0.35, 2);
    expect(data.channelLevel).toBeCloseTo(0.35, 2);
    expect(data.busGain).toBeCloseTo(0.35, 2);
  });

  test('applyMusicPreset streamer mutes music channel while retaining saved volume and blocks music preview', async ({ page }) => {
    const data = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.unlock();

      // Enable streamer preset
      audio.applyMusicPreset('streamer');

      const streamerState = {
        streamerMusic: audio.streamerMusic,
        musicPreset: audio.musicPreset,
        savedVolume: audio.music,
        channelLevel: audio.channelLevel('music'),
      };

      const previewBlocked = !(await audio.preview('music'));

      // Restore standard preset
      audio.applyMusicPreset('standard');

      const standardState = {
        streamerMusic: audio.streamerMusic,
        musicPreset: audio.musicPreset,
        savedVolume: audio.music,
        channelLevel: audio.channelLevel('music'),
      };

      return { streamerState, previewBlocked, standardState };
    });

    expect(data.streamerState.streamerMusic).toBe(true);
    expect(data.streamerState.musicPreset).toBe('streamer');
    expect(data.streamerState.savedVolume).toBeCloseTo(0.35, 2);
    expect(data.streamerState.channelLevel).toBe(0);
    expect(data.previewBlocked).toBe(true);

    expect(data.standardState.streamerMusic).toBe(false);
    expect(data.standardState.musicPreset).toBe('standard');
    expect(data.standardState.savedVolume).toBeCloseTo(0.35, 2);
    expect(data.standardState.channelLevel).toBeCloseTo(0.35, 2);
  });

  test('UI checkbox #rift-streamer-music toggles preset and updates settings summary and preview status', async ({ page }) => {
    const details = page.locator('.rift-settings');
    await details.evaluate(node => { node.open = true; });

    const checkbox = page.locator('#rift-streamer-music');
    await expect(checkbox).toBeVisible();
    await expect(checkbox).not.toBeChecked();

    const summary = page.locator('#rift-settings-summary');
    await expect(summary).toContainText('Music 35%');

    // Check streamer music preset
    await checkbox.check();

    const enabled = await page.evaluate(() => {
      const audio = window.RiftAudio;
      return {
        streamerMusic: audio.streamerMusic,
        musicPreset: audio.musicPreset,
        storedStreamer: localStorage.getItem('riftAudio:streamerMusic'),
        storedPreset: localStorage.getItem('riftAudio:musicPreset'),
      };
    });

    expect(enabled.streamerMusic).toBe(true);
    expect(enabled.musicPreset).toBe('streamer');
    expect(enabled.storedStreamer).toBe('true');
    expect(enabled.storedPreset).toBe('"streamer"');
    await expect(summary).toContainText('Music off (streamer preset)');

    // Attempting to preview music displays explanatory message
    const musicPreviewBtn = page.locator('button[data-audio-preview="music"]');
    await musicPreviewBtn.click();
    const status = page.locator('#rift-audio-preview-status');
    await expect(status).toHaveText('Procedural music is muted by streamer preset. Disable streamer preset to preview.');

    // Reload page to verify persistence
    await page.reload();
    await details.evaluate(node => { node.open = true; });
    await expect(checkbox).toBeChecked();
    await expect(summary).toContainText('Music off (streamer preset)');

    // On fresh context unlock after reload with streamer preset active, busGain initializes to 0
    const busGain = await page.evaluate(async () => {
      await window.RiftAudio.unlock();
      return window.RiftAudio.musicBus ? window.RiftAudio.musicBus.gain.value : null;
    });
    expect(busGain).toBe(0);
  });

  test('reset audio mix restores standard procedural music preset', async ({ page }) => {
    const details = page.locator('.rift-settings');
    await details.evaluate(node => { node.open = true; });

    const checkbox = page.locator('#rift-streamer-music');
    await checkbox.check();
    await expect(checkbox).toBeChecked();

    const resetBtn = page.locator('#rift-reset-audio');
    await resetBtn.click();

    await expect(checkbox).not.toBeChecked();
    const state = await page.evaluate(() => {
      return {
        streamerMusic: window.RiftAudio.streamerMusic,
        musicPreset: window.RiftAudio.musicPreset,
        channelLevel: window.RiftAudio.channelLevel('music'),
      };
    });
    expect(state.streamerMusic).toBe(false);
    expect(state.musicPreset).toBe('standard');
    expect(state.channelLevel).toBeCloseTo(0.35, 2);
  });
});
