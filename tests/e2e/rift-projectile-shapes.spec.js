const { test, expect } = require('@playwright/test');

test('differentiates hostile and friendly projectiles by geometric shape and silhouette', async ({ page }) => {
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  // 1. Verify shape profiles exposed on window.RiftRenderer
  const shapeData = await page.evaluate(() => {
    const renderer = window.RiftRenderer;
    const hostile = renderer.getProjectileShapeInfo(true);
    const friendly = renderer.getProjectileShapeInfo(false);
    return { hostile, friendly };
  });

  expect(shapeData.hostile).toEqual({
    hostility: 'hostile',
    shape: 'barbed-wedge',
    label: 'Barbed wedge with rear spurs',
  });

  expect(shapeData.friendly).toEqual({
    hostility: 'friendly',
    shape: 'diamond-crest',
    label: 'Diamond crest with swept wings',
  });

  expect(shapeData.hostile.shape).not.toEqual(shapeData.friendly.shape);

  // 2. Direct canvas rendering across multiple flight angles without errors
  const drawErrors = await page.evaluate(() => {
    const canvas = document.getElementById('rift-canvas');
    const ctx = canvas.getContext('2d');
    const renderer = window.RiftRenderer;
    const errors = [];

    const testAngles = [
      { vx: 400, vy: 0 },
      { vx: -400, vy: 0 },
      { vx: 300, vy: 200 },
      { vx: -200, vy: -150 },
    ];

    for (const isEnemy of [true, false]) {
      for (const { vx, vy } of testAngles) {
        try {
          renderer.drawProjectileShape(ctx, 400, 300, isEnemy, vx, vy, 'arrow');
          renderer.drawProjectileShape(ctx, 400, 300, isEnemy, vx, vy, 'fire');
        } catch (err) {
          errors.push({ isEnemy, vx, vy, error: err.message });
        }
      }
    }

    return errors;
  });

  expect(drawErrors).toEqual([]);

  // 3. Test snapshot rendering with simultaneous hostile and friendly projectiles
  const snapshotErrors = await page.evaluate(() => {
    const renderer = window.RiftRenderer;
    const errors = [];
    try {
      const dummyRun = {
        clock: 3.5,
        status: 'fighting',
        room: 0,
        player: { x: 200, y: 360, hp: 100, max_hp: 100, mana: 80, facing: 1, jump: 0 },
        enemies: [{ id: 'archer-1', kind: 'archer', x: 600, y: 360, hp: 40, max_hp: 40, facing: -1, art_key: 'archer' }],
        projectiles: [
          { id: 1, x: 280, y: 360, vx: 530, vy: 0, enemy: false, kind: 'fire' },
          { id: 2, x: 520, y: 360, vx: -300, vy: -50, enemy: true, kind: 'arrow' },
          { id: 3, x: 500, y: 340, vx: -300, vy: 50, enemy: true, kind: 'ice' },
        ],
        drops: [],
        events: [],
        skill_timers: {},
      };
      renderer.snapshot(dummyRun);
    } catch (err) {
      errors.push(err.message);
    }
    return errors;
  });

  expect(snapshotErrors).toEqual([]);

  // 4. Verify display setting toggle in Settings
  await page.locator('.rift-settings > summary').click();
  const shapesCheckbox = page.locator('#rift-projectile-shapes');
  await expect(shapesCheckbox).toBeVisible();
  await expect(shapesCheckbox).toBeChecked();

  // Toggle off and verify localStorage persistence
  await shapesCheckbox.uncheck();
  await expect(shapesCheckbox).not.toBeChecked();

  const savedOff = await page.evaluate(() => {
    return JSON.parse(localStorage.getItem('riftDisplay'))?.projectileShapes;
  });
  expect(savedOff).toBe(false);

  // Toggle on again
  await shapesCheckbox.check();
  await expect(shapesCheckbox).toBeChecked();

  const savedOn = await page.evaluate(() => {
    return JSON.parse(localStorage.getItem('riftDisplay'))?.projectileShapes;
  });
  expect(savedOn).toBe(true);

  // 5. Verify accessible preset includes projectileShapes
  const presetSelect = page.locator('#rift-display-preset');
  await presetSelect.selectOption('accessible');
  await page.locator('#rift-apply-preset').click();
  await expect(shapesCheckbox).toBeChecked();
});

test('mobile layout at 390px maintains display settings without horizontal overflow', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  const isOverflowing = await page.evaluate(() => {
    return document.documentElement.scrollWidth > window.innerWidth;
  });
  expect(isOverflowing).toBe(false);
});
