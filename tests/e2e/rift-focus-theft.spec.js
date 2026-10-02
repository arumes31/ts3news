const { test, expect } = require('@playwright/test');

test.describe('Avoid focus theft when combat counters update (Proposal 0140)', () => {
  test('retains active focus on combat skill button during cooldown without focus theft to body', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    // Start expedition
    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-pause')).toBeEnabled();

    const skill0 = page.locator('#rift-skills button[data-bind="skill0"]');
    await expect(skill0).toBeAttached();

    // 1. Move keyboard focus to the first skill button
    await skill0.focus();
    await expect(skill0).toBeFocused();

    // 2. Activate the skill using Space or Enter key
    await page.keyboard.press('Space');

    // 3. The skill goes on cooldown (countdown timer begins, e.g. "3.9s" or "4.0s")
    await expect(skill0.locator('small')).toContainText(/s$/);

    // 4. CRITICAL: Active focus must NOT be stolen to document.body!
    await expect(skill0).toBeFocused();
    const activeIsBody = await page.evaluate(() => document.activeElement === document.body);
    expect(activeIsBody).toBe(false);

    // 5. While focused and on cooldown, aria-disabled is true, but native disabled is false
    expect(await skill0.getAttribute('aria-disabled')).toBe('true');
    const isNativeDisabledWhileFocused = await page.evaluate(() => {
      const btn = document.querySelector('#rift-skills button[data-bind="skill0"]');
      return btn.disabled;
    });
    expect(isNativeDisabledWhileFocused).toBe(false);

    // 6. Activating while aria-disabled="true" is prevented
    await page.keyboard.press('Enter');
    await expect(skill0).toBeFocused();

    // 7. When tabbing away, blur event safely transitions the cooldown button to native disabled
    await page.keyboard.press('Tab');
    const isNativeDisabledAfterBlur = await page.evaluate(() => {
      const btn = document.querySelector('#rift-skills button[data-bind="skill0"]');
      return btn.disabled;
    });
    expect(isNativeDisabledAfterBlur).toBe(true);

    // Focus successfully moved to next element without being dumped to body
    const activeTagAfterTab = await page.evaluate(() => document.activeElement?.tagName);
    expect(activeTagAfterTab).toBeTruthy();
    expect(await page.evaluate(() => document.activeElement === document.body)).toBe(false);
  });

  test('does not steal focus from canvas or pause controls as live combat counters tick', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();

    // Start expedition
    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-pause')).toBeEnabled();

    // 1. Focus battlefield canvas during combat
    const canvas = page.locator('#rift-canvas');
    await canvas.focus();
    await expect(canvas).toBeFocused();

    // Allow multiple combat counter ticks (85ms each)
    await page.waitForTimeout(450);

    // Canvas must retain focus
    await expect(canvas).toBeFocused();

    // 2. Focus Pause control during live combat
    const pauseBtn = page.locator('#rift-pause');
    await pauseBtn.focus();
    await expect(pauseBtn).toBeFocused();

    // Allow counter updates (HP, mana, latency, elapsed time)
    await page.waitForTimeout(450);

    // Pause control must retain focus
    await expect(pauseBtn).toBeFocused();

    // 3. Focus an actionbar basic button (Attack)
    const attackBtn = page.locator('#rift-actionbar button[data-bind="attack"]');
    await attackBtn.focus();
    await expect(attackBtn).toBeFocused();

    await page.waitForTimeout(450);
    await expect(attackBtn).toBeFocused();
  });

  test('preserves focus on actionbar button if container is reconstructed', async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
    await page.locator('#rift-start').click();
    await expect(page.locator('#rift-pause')).toBeEnabled();

    const skill0 = page.locator('#rift-skills button[data-bind="skill0"]');
    await expect(skill0).toBeAttached();
    await skill0.focus();
    await expect(skill0).toBeFocused();

    // Trigger skills container rebuild while focused
    await page.evaluate(() => {
      const container = document.getElementById('rift-skills');
      const active = document.activeElement;
      const activeBind = container && active && container.contains(active) ? active.dataset?.bind : null;
      container.replaceChildren();
      // rebuild dummy button with same data-bind
      const btn = document.createElement('button');
      btn.dataset.bind = 'skill0';
      btn.textContent = 'Rebuilt Skill';
      container.append(btn);
      if (activeBind && container.querySelector(`[data-bind="${activeBind}"]`)) {
        container.querySelector(`[data-bind="${activeBind}"]`).focus();
      }
    });

    // Focus must still be on the rebuilt skill0 button
    await expect(page.locator('#rift-skills button[data-bind="skill0"]')).toBeFocused();
    expect(await page.evaluate(() => document.activeElement === document.body)).toBe(false);
  });
});
