const { test, expect } = require('@playwright/test');

test.describe('Keep master mute independent from saved channel volumes (Proposal 0163)', () => {
  test.beforeEach(async ({ page }) => {
    // Clear any previous audio local storage before each test run (guarded against page reload)
    await page.addInitScript(() => {
      if (!sessionStorage.getItem('audioInitClean')) {
        ['muted', 'effects', 'ambience', 'music', 'voice', 'interface', 'interfaceMuted', 'mono', 'steadyAmbience'].forEach(key => {
          localStorage.removeItem('riftAudio:' + key);
        });
        sessionStorage.setItem('audioInitClean', '1');
      }
    });
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('master mute toggle does not alter channel volume sliders or stored levels', async ({ page }) => {
    const settings = page.locator('.rift-settings');
    await settings.locator('> summary').click();

    const soundBtn = page.locator('#rift-sound');
    const masterMuteCheckbox = page.locator('#rift-master-muted');
    const effectsSlider = page.locator('#rift-effects-volume');
    const ambienceSlider = page.locator('#rift-ambience-volume');
    const musicSlider = page.locator('#rift-music-volume');
    const voiceSlider = page.locator('#rift-voice-volume');
    const interfaceSlider = page.locator('#rift-interface-volume');

    // Initial state: unmuted by default with standard channel levels
    await expect(soundBtn).toHaveText('Sound on');
    await expect(soundBtn).toHaveAttribute('aria-pressed', 'false');
    await expect(masterMuteCheckbox).not.toBeChecked();
    await expect(effectsSlider).toHaveValue('65');
    await expect(ambienceSlider).toHaveValue('35');
    await expect(musicSlider).toHaveValue('35');
    await expect(voiceSlider).toHaveValue('65');
    await expect(interfaceSlider).toHaveValue('65');

    // Mute master via the settings checkbox
    await masterMuteCheckbox.check();

    // Top bar sound button must sync immediately via riftaudiochange
    await expect(soundBtn).toHaveText('Sound off');
    await expect(soundBtn).toHaveAttribute('aria-pressed', 'true');

    // Channel sliders must remain completely unaffected
    await expect(effectsSlider).toHaveValue('65');
    await expect(ambienceSlider).toHaveValue('35');
    await expect(musicSlider).toHaveValue('35');
    await expect(voiceSlider).toHaveValue('65');
    await expect(interfaceSlider).toHaveValue('65');

    // Summary reflects both master mute and individual mix percentages
    const summary = settings.locator('> summary');
    await expect(summary).toContainText('Sound muted');
    await expect(summary).toContainText('Effects 65%');
    await expect(summary).toContainText('Ambience 35%');

    // Unmute via top bar sound button
    await soundBtn.click();
    await expect(soundBtn).toHaveText('Sound on');
    await expect(soundBtn).toHaveAttribute('aria-pressed', 'false');
    await expect(masterMuteCheckbox).not.toBeChecked();

    // Channel sliders still preserved
    await expect(effectsSlider).toHaveValue('65');
    await expect(musicSlider).toHaveValue('35');
  });

  test('editing channel volumes while master is muted persists without unmuting master', async ({ page }) => {
    const settings = page.locator('.rift-settings');
    await settings.locator('> summary').click();

    const soundBtn = page.locator('#rift-sound');
    const masterMuteCheckbox = page.locator('#rift-master-muted');
    const musicSlider = page.locator('#rift-music-volume');
    const voiceSlider = page.locator('#rift-voice-volume');

    // Mute master
    await soundBtn.click();
    await expect(soundBtn).toHaveText('Sound off');
    await expect(masterMuteCheckbox).toBeChecked();

    // Change channel levels while muted
    await musicSlider.fill('18');
    await voiceSlider.fill('82');

    // Master must remain muted
    await expect(soundBtn).toHaveText('Sound off');
    await expect(masterMuteCheckbox).toBeChecked();

    // Reload page to test persistence
    await page.reload();
    await page.locator('.rift-settings > summary').click();

    // After reload, master remains muted and modified channel levels are restored
    await expect(page.locator('#rift-sound')).toHaveText('Sound off');
    await expect(page.locator('#rift-master-muted')).toBeChecked();
    await expect(page.locator('#rift-music-volume')).toHaveValue('18');
    await expect(page.locator('#rift-voice-volume')).toHaveValue('82');

    // Unmute master: channel levels remain at custom values
    await page.locator('#rift-master-muted').uncheck();
    await expect(page.locator('#rift-sound')).toHaveText('Sound on');
    await expect(page.locator('#rift-music-volume')).toHaveValue('18');
    await expect(page.locator('#rift-voice-volume')).toHaveValue('82');
  });

  test('zeroing individual channel sliders does not trigger master mute', async ({ page }) => {
    const settings = page.locator('.rift-settings');
    await settings.locator('> summary').click();

    const soundBtn = page.locator('#rift-sound');
    const masterMuteCheckbox = page.locator('#rift-master-muted');

    // Set all channels to 0
    for (const key of ['effects', 'ambience', 'music', 'voice', 'interface']) {
      await page.locator(`#rift-${key}-volume`).fill('0');
    }

    // Master sound must remain enabled
    await expect(soundBtn).toHaveText('Sound on');
    await expect(soundBtn).toHaveAttribute('aria-pressed', 'false');
    await expect(masterMuteCheckbox).not.toBeChecked();
    await expect(settings.locator('> summary')).toContainText('Sound enabled');
  });

  test('audio mix reset restores default channel volumes without changing master mute state', async ({ page }) => {
    const settings = page.locator('.rift-settings');
    await settings.locator('> summary').click();

    const soundBtn = page.locator('#rift-sound');
    const masterMuteCheckbox = page.locator('#rift-master-muted');
    const resetBtn = page.locator('#rift-reset-audio');
    const statusMsg = page.locator('#rift-audio-preview-status');

    // Mute master and change music slider
    await soundBtn.click();
    await expect(soundBtn).toHaveText('Sound off');
    await page.locator('#rift-music-volume').fill('10');

    // Click reset mix
    await resetBtn.click();

    // Master mute is unchanged
    await expect(soundBtn).toHaveText('Sound off');
    await expect(masterMuteCheckbox).toBeChecked();
    await expect(page.locator('#rift-music-volume')).toHaveValue('35');
    await expect(statusMsg).toHaveText('Channel levels restored. Master mute is unchanged.');
  });
});
