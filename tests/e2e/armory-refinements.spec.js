const { test, expect } = require('@playwright/test');

const quote = {
  ok: true,
  quote: { cost: { gold: 1000 }, token: 'armory-test-quote', confirmation_phrase: 'FORGE IDENTIFY' },
};

test('Armoury labels match and attributes follow the character equipment sheet', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto('/armory-fixture');
  const rhythm = await page.evaluate(() => {
    const font = selector => getComputedStyle(document.querySelector(selector)).font;
    return {
      occupiedFont: font('.gear-cell:not(.empty) .gear-slot'),
      emptyFont: font('.gear-cell.empty .gear-slot'),
      equipmentBottom: document.querySelector('.armory-equipment').getBoundingClientRect().bottom,
      attributesTop: document.querySelector('.armory-telemetry').getBoundingClientRect().top,
    };
  });
  expect.soft(rhythm.occupiedFont).toBe(rhythm.emptyFont);
  expect.soft(rhythm.attributesTop).toBeGreaterThan(rhythm.equipmentBottom);
});

test('empty Armoury slots have visible hover and keyboard feedback without motion', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.goto('/armory-fixture');
  const slot = page.locator('.gear-cell[data-slot="Neck"]');
  const border = await slot.evaluate(node => getComputedStyle(node).borderTopColor);
  await slot.hover();
  expect.soft(await slot.evaluate(node => getComputedStyle(node).borderTopColor)).not.toBe(border);
  expect.soft(await slot.evaluate(node => getComputedStyle(node).transform)).toBe('none');
  await slot.getByRole('link').focus();
  expect(await slot.evaluate(node => parseFloat(getComputedStyle(node).outlineWidth))).toBeGreaterThanOrEqual(2);
});

test('Armoury inspector actions are centered touch targets at every layout size', async ({ page }) => {
  await page.goto('/armory-fixture');
  await page.getByRole('button', { name: 'Inspect Measured Test Blade', exact: true }).click();
  await page.locator('.modal-card').evaluate(node => Promise.all(node.getAnimations().map(animation => animation.finished)));
  for (const width of [320, 390, 768, 1440]) {
    await page.setViewportSize({ width, height: 900 });
    const inspector = page.locator('.item-inspector');
    const controls = await inspector.locator('.modal-actions > *').evaluateAll(nodes => nodes.map(node => ({
      label: node.textContent,
      height: node.getBoundingClientRect().height,
      centered: getComputedStyle(node).alignItems === 'center',
      padding: parseFloat(getComputedStyle(node).paddingInlineStart),
    })));
    for (const control of controls) {
      expect.soft(control.height, `${width}px ${control.label}`).toBeGreaterThanOrEqual(44);
      expect.soft(control.centered, `${width}px ${control.label}`).toBe(true);
      expect.soft(control.padding, `${width}px ${control.label}`).toBeGreaterThanOrEqual(8);
    }
    expect.soft(await inspector.evaluate(node => node.scrollWidth - node.clientWidth)).toBeLessThanOrEqual(1);
  }
});

test('Armoury complete records wrap long names and exact values without clipping', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 844 });
  await page.goto('/armory-fixture');
  const blade = page.getByRole('button', { name: 'Inspect Measured Test Blade', exact: true });
  await blade.evaluate(node => {
    const item = JSON.parse(node.dataset.itemInspect);
    item.name = 'Ancient-Sentinels-Blade-of-the-Unbroken-Constellation'.repeat(3);
    item.cr = 123456789012345;
    item.score = 123456789012345;
    item.stats.forEach(stat => { stat.value = 123456789012345; });
    node.dataset.itemInspect = JSON.stringify(item);
  });
  await blade.click();
  const inspector = page.locator('.item-inspector');
  expect(await inspector.evaluate(node => node.scrollWidth - node.clientWidth)).toBeLessThanOrEqual(1);
  await inspector.getByRole('button', { name: 'Close', exact: true }).click();
  await expect(blade).toBeFocused();
});

test('all Armoury slots remain readable while mobile reaches equipment sooner', async ({ page }) => {
  await page.goto('/armory-fixture');
  for (const width of [320, 390, 768, 1440]) {
    await page.setViewportSize({ width, height: 900 });
    const slots = page.locator('.armory-equipment .gear-cell');
    await expect(slots).toHaveCount(30);
    const layout = await slots.evaluateAll(nodes => {
      const empty = nodes.filter(node => node.classList.contains('empty'));
      const condition = document.querySelector('.gear-durability');
      const value = condition.querySelector('strong');
      return {
        allVisible: nodes.every(node => node.getBoundingClientRect().height > 0),
        emptyHeight: Math.max(...empty.map(node => node.getBoundingClientRect().height)),
        fadedEmpty: empty.some(node => getComputedStyle(node).opacity !== '1'),
        overflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
        conditionOverflow: value.getBoundingClientRect().right - condition.getBoundingClientRect().right,
        equipmentTop: document.getElementById('loadoutTitle').getBoundingClientRect().top + window.scrollY,
      };
    });
    expect(layout.allVisible).toBe(true);
    expect(layout.emptyHeight).toBeLessThan(100);
    expect(layout.fadedEmpty).toBe(false);
    expect(layout.overflow).toBeLessThanOrEqual(1);
    expect(layout.conditionOverflow).toBeLessThanOrEqual(1);
    if (width <= 390) expect(layout.equipmentTop).toBeLessThan(650);
  }
});

test('exact values agree across health, item cards, and the complete record', async ({ page }) => {
  await page.goto('/armory-fixture');
  const health = page.getByRole('progressbar', { name: 'Current health' });
  await expect(health).toContainText('123,456 / 234,567');
  await expect(health).toHaveAttribute('aria-valuetext', '123,456 of 234,567 health');
  const blade = page.getByRole('button', { name: 'Inspect Measured Test Blade', exact: true });
  await expect(blade.locator('.gear-meta')).toContainText('348,828.8');
  await blade.focus();
  await page.keyboard.press('Enter');
  const inspector = page.locator('.item-inspector');
  await expect(inspector).toContainText('37 / 100 durability');
  await expect(inspector).toContainText('Broken in · +1% stats');
  await expect(inspector.locator('.item-inspector-score')).toContainText('348,828.8');
  await expect(inspector.getByRole('link', { name: 'Manage MainHand' })).toHaveAttribute('href', '/inventory?slot=MainHand#inventoryGear');
  await page.keyboard.press('Escape');
  await expect(blade).toBeFocused();

  await page.getByRole('button', { name: 'Numbers: exact', exact: true }).click();
  await expect(health).toContainText('123.5K');
  await expect(health).toHaveAttribute('aria-valuetext', '123,456 of 234,567 health');
  await blade.click();
  await expect(inspector.locator('.item-inspector-score')).toContainText('348.8K');
  await page.keyboard.press('Escape');
  await page.reload();
  await expect(page.getByRole('button', { name: 'Numbers: compact', exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Numbers: compact', exact: true }).click();
  await expect(health).toContainText('123,456 / 234,567');
});

test('broken durability zero remains visible in the item record', async ({ page }) => {
  await page.goto('/armory-fixture');
  const blade = page.getByRole('button', { name: 'Inspect Measured Test Blade', exact: true });
  await blade.evaluate(node => { node.dataset.itemDurability = '0'; });
  await blade.click();
  await expect(page.locator('.item-inspector')).toContainText('Broken · 0 / 100 durability');
});

test('long item names and large exact values fit the narrow layout', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 844 });
  await page.goto('/armory-fixture');
  await page.locator('.gear-cell[data-slot="MainHand"]').evaluate(node => {
    node.querySelector('.gear-name').textContent = 'Ancient-Sentinels-Blade-of-the-Unbroken-Constellation'.repeat(3);
    node.querySelectorAll('[data-item-number]').forEach(value => { value.dataset.itemNumber = '123456789012345'; });
    window.AbyssItemNumbers.render(node);
  });
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(overflow).toBeLessThanOrEqual(1);
  const name = page.locator('.gear-cell[data-slot="MainHand"] .gear-name');
  expect(await name.evaluate(node => node.scrollWidth - node.clientWidth)).toBeLessThanOrEqual(1);
});

test('a valid zero-cost quote keeps the daily free identification available', async ({ page }) => {
  await page.route('**/api/abyss/forge/quote', route => route.fulfill({
    json: { ...quote, quote: { ...quote.quote, cost: { gold: 0 } } },
  }));
  await page.goto('/armory-fixture');
  await page.getByRole('button', { name: 'Review identification cost for Head' }).click();
  const dialog = page.getByRole('dialog');
  await expect(dialog.getByRole('button', { name: 'Use daily free identify' })).toBeVisible();
  await dialog.getByRole('button', { name: 'Keep unidentified' }).click();
  await expect(page.locator('#armoryStatus')).toContainText('No gold was spent');
});

test('an Armoury slot opens Inventory with matching items and a clear way back to all slots', async ({ page }) => {
  await page.goto('/armory-fixture');
  await page.getByRole('link', { name: 'Find equipment for Neck', exact: true }).click();
  await expect(page).toHaveURL(/\/inventory\?slot=Neck#inventoryGear$/);
  await expect(page.locator('#inventorySlot')).toHaveValue('Neck');
  const matching = await page.locator('.inv-card:not([hidden])').evaluateAll(nodes => nodes.every(node => node.dataset.slot === 'Neck'));
  expect(matching).toBe(true);
  await page.locator('#inventorySlot').selectOption('');
  await expect(page.locator('.inv-card[hidden]')).toHaveCount(0);
  await expect(page).toHaveURL(/\/inventory#inventoryGear$/);

  await page.goto('/inventory?slot=NoSuchSlot#inventoryGear');
  await expect(page.locator('#inventorySlotEmpty')).toBeVisible();
  await expect(page.getByRole('link', { name: 'Browse the Shop' })).toHaveAttribute('href', '/shop');
});

test('section and repair links reach the correct workspace without spending', async ({ page }) => {
  await page.goto('/armory-fixture');
  await page.getByRole('navigation', { name: 'Armoury sections' }).getByRole('link', { name: 'Skills', exact: true }).click();
  await expect(page.locator('#skillsTitle')).toBeInViewport();
  await expect(page.getByRole('link', { name: 'Open Skill Web' })).toHaveAttribute('href', '/abyss/tree');
  await page.getByRole('link', { name: 'Review repairs', exact: true }).first().click();
  await expect(page).toHaveURL(/\/abyss#btnForgeRepairAll$/);
  await expect(page.locator('#btnForgeRepairAll')).toBeVisible();
});

for (const failure of ['non-JSON', 'network', 'malformed quote']) {
  test(`identification recovers from ${failure} without a charge or parser message`, async ({ page }) => {
    let attempts = 0;
    let commits = 0;
    await page.route('**/api/abyss/identify', async route => { commits++; await route.abort(); });
    await page.route('**/api/abyss/forge/quote', async route => {
      attempts++;
      if (attempts > 1) return route.fulfill({ json: quote });
      if (failure === 'network') return route.abort('connectionreset');
      if (failure === 'malformed quote') return route.fulfill({ json: { ok: true, quote: { cost: {} } } });
      return route.fulfill({ status: 502, contentType: 'text/html', body: '<h1>Bad gateway</h1>' });
    });
    await page.goto('/armory-fixture');
    const identify = page.getByRole('button', { name: 'Review identification cost for Head' });
    await identify.click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toContainText('No gold was spent');
    await expect(dialog).not.toContainText(/JSON|SyntaxError|Unexpected|Bad gateway/);
    await expect(identify).toBeEnabled();
    await dialog.getByRole('button', { name: 'Check price again' }).click();
    await expect(dialog).toContainText('1,000 gold');
    await dialog.getByRole('button', { name: 'Keep unidentified' }).click();
    await expect(identify).toBeFocused();
    expect(attempts).toBe(2);
    expect(commits).toBe(0);
  });
}
