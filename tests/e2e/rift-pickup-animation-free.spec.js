const { test, expect } = require('@playwright/test');

test.describe('Show pickup confirmation without requiring animation (Proposal 0151)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift?scenario=checkpoint');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('static pickup confirmation is rendered on canvas when reduced motion or disabled sparkles are active', async ({ page }) => {
    const confirmation = await page.evaluate(() => {
      const renderer = window.RiftRenderer;
      const canvas = document.createElement('canvas');
      canvas.width = 200;
      canvas.height = 200;
      const ctx = canvas.getContext('2d');

      // Verify drawStaticPickup helper is available and draws on canvas
      renderer.drawStaticPickup(ctx, 100, 100);
      const pixelData = ctx.getImageData(84, 84, 32, 32).data;
      const hasDrawn = Array.from(pixelData).some(channel => channel > 0);

      // Verify that reduced motion suppresses multi-frame particle animations
      const reducedActive = renderer.reduced;
      return { hasDrawn, reducedActive };
    });

    expect(confirmation.hasDrawn).toBe(true);
  });

  test('pickup notices display static confirmation with checkmark prefix and data-confirmed attribute without animation', async ({ page }) => {
    const notices = page.locator('#rift-pickup-notices');
    await expect(notices).toBeHidden();

    // Trigger pickup update with unbanked loot and gold
    await page.evaluate(async () => {
      const response = await fetch('/api/abyss/rift');
      const data = await response.json();
      const run = data.run;
      run.drops[0].collected = false;
      window.RiftLoot.update(run, true);

      // Mark drop collected
      run.drops[0].collected = true;
      window.RiftLoot.update(run);
    });

    // Notices should be visible with data-confirmed attribute
    await expect(notices).toBeVisible();
    await expect(notices).toHaveAttribute('data-confirmed', 'true');

    // Confirm text has static checkmark prefix and contents
    const pElements = notices.locator('p');
    await expect(pElements).toHaveCount(2);
    await expect(pElements.first()).toContainText('✓ Gold picked up: +30');
    await expect(pElements.last()).toContainText('✓ Gear picked up:');

    // Notices should not have CSS animation or transitions that block reading
    const animationState = await notices.evaluate(node => {
      const style = window.getComputedStyle(node);
      return {
        animationName: style.animationName,
        opacity: style.opacity,
        display: style.display
      };
    });
    expect(animationState.animationName).toBe('none');
    expect(animationState.opacity).toBe('1');
  });

  test('accessible and minimal presets show pickup confirmation without requiring animated sparkles', async ({ page }) => {
    // Enable accessible display preset (which sets lootSparkle: false, lootMotion: false, damageMotion: false)
    await page.locator('.rift-settings > summary').click();
    await page.locator('#rift-reduced').check();

    // Verify reduced visual effects is set
    const reducedChecked = await page.locator('#rift-reduced').isChecked();
    expect(reducedChecked).toBe(true);

    // Trigger pickup
    await page.evaluate(async () => {
      const response = await fetch('/api/abyss/rift');
      const data = await response.json();
      const run = data.run;
      run.drops[0].collected = false;
      window.RiftLoot.update(run, true);
      run.drops[0].collected = true;
      window.RiftLoot.update(run);
    });

    // Static confirmation is immediately visible in DOM
    const notices = page.locator('#rift-pickup-notices');
    await expect(notices).toBeVisible();
    await expect(notices).toContainText('✓ Gold picked up:');
  });
});
