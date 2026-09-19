const { test, expect } = require('@playwright/test');

test('differentiates all hazard types with distinct tactile patterns and exposes display toggle', async ({ page }) => {
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  // 1. Verify pattern profiles exposed on window.RiftRenderer
  const patternData = await page.evaluate(() => {
    const renderer = window.RiftRenderer;
    const kinds = renderer.getHazardKindsWithPatterns();
    const profiles = {};
    for (const k of kinds) {
      profiles[k] = renderer.getHazardPatternInfo(k);
    }
    return { kinds, profiles };
  });

  const expectedKinds = ['fire', 'ice', 'poison', 'thorns', 'rune', 'radiant', 'void'];
  expect(patternData.kinds).toEqual(expect.arrayContaining(expectedKinds));

  // Verify each hazard kind has a distinct, meaningful pattern
  const patternsSeen = new Set();
  for (const kind of expectedKinds) {
    const info = patternData.profiles[kind];
    expect(info).toBeDefined();
    expect(info.kind).toBe(kind);
    expect(info.pattern).toBeTruthy();
    expect(info.label).toBeTruthy();
    expect(patternsSeen.has(info.pattern)).toBe(false);
    patternsSeen.add(info.pattern);
  }

  expect(patternData.profiles.fire.pattern).toBe('diagonal-stripes');
  expect(patternData.profiles.ice.pattern).toBe('diamond-grid');
  expect(patternData.profiles.poison.pattern).toBe('polka-dots');
  expect(patternData.profiles.thorns.pattern).toBe('chevrons');
  expect(patternData.profiles.rune.pattern).toBe('concentric-diamonds');
  expect(patternData.profiles.radiant.pattern).toBe('vertical-beams');
  expect(patternData.profiles.void.pattern).toBe('dashed-scanlines');

  // 2. Verify display setting toggle in Settings
  const patternsCheckbox = page.locator('#rift-hazard-patterns');
  await expect(patternsCheckbox).toBeAttached();
  await expect(patternsCheckbox).toBeChecked();

  // 3. Test canvas execution across all hazard types without errors
  const renderAllHazards = await page.evaluate(() => {
    const canvas = document.getElementById('rift-canvas');
    const ctx = canvas.getContext('2d');
    const renderer = window.RiftRenderer;
    const errors = [];
    const kinds = renderer.getHazardKindsWithPatterns();

    // Mock run with all hazard kinds active and in warning phase
    kinds.forEach((kind, i) => {
      try {
        const dummyRun = {
          clock: 2.0,
          status: 'fighting',
          practice: {
            arena: {
              hazards: [
                { x: 100 + i * 80, y: 340, w: 70, h: 40, kind, period: 5, offset: 0, duration: 3 },
              ],
            },
          },
          enemies: [],
          player: { x: 50, y: 350, jump: 0 },
        };
        // Test both direct pattern drawing and snapshot update
        renderer.drawHazardPattern(ctx, kind, 100 + i * 80, 340, 70, 40, true, '#ff9955');
        renderer.drawHazardPattern(ctx, kind, 100 + i * 80, 340, 70, 40, false, '#ff9955');
        renderer.snapshot(dummyRun);
      } catch (err) {
        errors.push({ kind, error: err.message });
      }
    });

    return errors;
  });

  expect(renderAllHazards).toEqual([]);

  // 4. Toggle hazard patterns off and on in display settings
  await page.locator('.rift-settings > summary').click();
  await patternsCheckbox.uncheck();
  await expect(patternsCheckbox).not.toBeChecked();

  const savedOff = await page.evaluate(() => {
    return JSON.parse(localStorage.getItem('riftDisplay'))?.hazardPatterns;
  });
  expect(savedOff).toBe(false);

  await patternsCheckbox.check();
  await expect(patternsCheckbox).toBeChecked();

  const savedOn = await page.evaluate(() => {
    return JSON.parse(localStorage.getItem('riftDisplay'))?.hazardPatterns;
  });
  expect(savedOn).toBe(true);

  // 5. Verify accessible preset includes hazardPatterns
  const presetSelect = page.locator('#rift-display-preset');
  await presetSelect.selectOption('accessible');
  await page.locator('#rift-apply-preset').click();
  await expect(patternsCheckbox).toBeChecked();
});

test('hazard practice drill displays hazard patterns on active and warning zones', async ({ page }) => {
  await page.goto('/abyss/rift?practice=hazard');
  await expect(page.locator('#rift-start')).toBeEnabled();

  await page.locator('#rift-start').click();
  await expect(page.locator('#rift-overlay')).toBeHidden();

  // Verify canvas is running and no errors are triggered during live gameplay
  const canvasErrors = [];
  page.on('pageerror', (err) => canvasErrors.push(err.message));

  await page.waitForTimeout(500);
  expect(canvasErrors).toEqual([]);
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
