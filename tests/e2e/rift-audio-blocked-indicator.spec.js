const { test, expect } = require('@playwright/test');

test.describe('Show whether the browser has blocked audio (Proposal 0198)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await page.evaluate(() => {
      window.RiftAudio.set('muted', false);
      window.RiftAudio.blocked = false;
    });
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('default active state shows unblocked audio controls and clean settings summary', async ({ page }) => {
    const soundBtn = page.locator('#rift-sound');
    await expect(soundBtn).toHaveText('Sound on');
    await expect(soundBtn).not.toHaveAttribute('data-audio-blocked', 'true');
    expect(await soundBtn.getAttribute('title') || '').toBe('');

    const details = page.locator('.rift-settings');
    await details.evaluate(node => { node.open = true; });

    const blockedNotice = page.locator('#rift-audio-blocked-notice');
    await expect(blockedNotice).toBeHidden();

    const summary = page.locator('#rift-settings-summary');
    await expect(summary).toContainText('Sound enabled');
  });

  test('when browser blocks audio, sound button shows Sound blocked and displays status notices', async ({ page }) => {
    const soundBtn = page.locator('#rift-sound');

    // Simulate browser autoplay blocking
    await page.evaluate(() => {
      window.RiftAudio.blocked = true;
    });

    await expect(soundBtn).toHaveText('Sound blocked');
    await expect(soundBtn).toHaveAttribute('data-audio-blocked', 'true');
    await expect(soundBtn).toHaveAttribute('title', 'Audio is blocked by the browser. Click to allow sound.');

    const details = page.locator('.rift-settings');
    await details.evaluate(node => { node.open = true; });

    const blockedNotice = page.locator('#rift-audio-blocked-notice');
    await expect(blockedNotice).toBeVisible();
    await expect(blockedNotice).toHaveAttribute('role', 'status');
    await expect(blockedNotice).toContainText('Audio is currently blocked by your browser’s autoplay policy');

    const summary = page.locator('#rift-settings-summary');
    await expect(summary).toContainText('Sound blocked by browser');
  });

  test('muting sound takes precedence over blocked display', async ({ page }) => {
    const soundBtn = page.locator('#rift-sound');

    await page.evaluate(() => {
      window.RiftAudio.blocked = true;
      window.RiftAudio.set('muted', true);
    });

    await expect(soundBtn).toHaveText('Sound off');
    await expect(soundBtn).not.toHaveAttribute('data-audio-blocked', 'true');

    const details = page.locator('.rift-settings');
    await details.evaluate(node => { node.open = true; });

    const blockedNotice = page.locator('#rift-audio-blocked-notice');
    await expect(blockedNotice).toBeHidden();

    const summary = page.locator('#rift-settings-summary');
    await expect(summary).toContainText('Sound muted');
  });

  test('clicking sound button or unlocking audio clears the blocked state', async ({ page }) => {
    const soundBtn = page.locator('#rift-sound');

    // Set blocked initially
    await page.evaluate(() => {
      window.RiftAudio.blocked = true;
      window.RiftAudio.set('muted', false);
    });
    await expect(soundBtn).toHaveText('Sound blocked');

    // Clicking sound button performs gesture unlock and toggles mute
    await soundBtn.click();

    // Sound was toggled to muted
    await expect(soundBtn).toHaveText('Sound off');

    // Clicking sound button again unmutes with running context
    await soundBtn.click();
    await expect(soundBtn).toHaveText('Sound on');
    await expect(soundBtn).not.toHaveAttribute('data-audio-blocked', 'true');

    const details = page.locator('.rift-settings');
    await details.evaluate(node => { node.open = true; });

    const blockedNotice = page.locator('#rift-audio-blocked-notice');
    await expect(blockedNotice).toBeHidden();

    const summary = page.locator('#rift-settings-summary');
    await expect(summary).toContainText('Sound enabled');
  });
});
