const { test, expect } = require('@playwright/test');

test('keeps long boss names readable without covering the health meter in desktop layout', async ({ page }) => {
  await page.goto('/abyss/rift?scenario=long-boss-name');
  await expect(page.locator('#rift-start')).toBeEnabled();

  // Resume the paused encounter
  await page.locator('#rift-start').click();

  const bossSection = page.locator('#rift-boss');
  await expect(bossSection).toBeVisible();

  const bossName = page.locator('#rift-boss-name');
  await expect(bossName).toBeVisible();
  await expect(bossName).toContainText('Ancient Mossbound Colossus');

  // Verify title attribute provides the full unabridged name on hover
  await expect(bossName).toHaveAttribute(
    'title',
    'Ancient Mossbound Colossus of the Verdant Abyss and Eternal Primordial Shadows'
  );

  const bossMeter = page.locator('#rift-boss .hp');
  await expect(bossMeter).toBeVisible();

  // Bounding box verification: the name must NEVER cover or overlap the health meter
  const nameBox = await bossName.boundingBox();
  const meterBox = await bossMeter.boundingBox();

  expect(nameBox).not.toBeNull();
  expect(meterBox).not.toBeNull();
  expect(meterBox.width).toBeGreaterThan(60);
  expect(meterBox.height).toBeGreaterThanOrEqual(10);

  // The bottom of the boss name must sit above the top of the health meter
  expect(nameBox.y + nameBox.height).toBeLessThanOrEqual(meterBox.y + 0.5);

  // Clean screenshot mode hides boss HUD
  const screenshotBtn = page.locator('#rift-screenshot-toggle');
  await screenshotBtn.click();
  await expect(screenshotBtn).toHaveAttribute('aria-pressed', 'true');
  await expect(bossSection).not.toBeVisible();

  // Restore HUD
  await screenshotBtn.click();
  await expect(screenshotBtn).toHaveAttribute('aria-pressed', 'false');
  await expect(bossSection).toBeVisible();
  await expect(bossMeter).toBeVisible();
});

test('mobile viewport at 390px keeps long boss names readable without covering health or horizontal overflow', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/abyss/rift?scenario=long-boss-name');
  await expect(page.locator('#rift-start')).toBeEnabled();

  await page.locator('#rift-start').click();

  const bossSection = page.locator('#rift-boss');
  await expect(bossSection).toBeVisible();

  const bossName = page.locator('#rift-boss-name');
  await expect(bossName).toBeVisible();
  await expect(bossName).toHaveAttribute(
    'title',
    'Ancient Mossbound Colossus of the Verdant Abyss and Eternal Primordial Shadows'
  );

  const bossMeter = page.locator('#rift-boss .hp');
  await expect(bossMeter).toBeVisible();

  // Ensure health meter remains visible and non-overlapping on narrow mobile screens
  const nameBox = await bossName.boundingBox();
  const meterBox = await bossMeter.boundingBox();

  expect(nameBox).not.toBeNull();
  expect(meterBox).not.toBeNull();
  expect(meterBox.width).toBeGreaterThan(40);
  expect(meterBox.height).toBeGreaterThanOrEqual(8);
  expect(nameBox.y + nameBox.height).toBeLessThanOrEqual(meterBox.y + 0.5);

  // Verify no horizontal overflow at 390px
  const isOverflowing = await page.evaluate(() => {
    return document.documentElement.scrollWidth > window.innerWidth;
  });
  expect(isOverflowing).toBe(false);
});


for(const width of [320,768])test('unbroken boss name preserves health and phase layout at '+width+'px',async({page})=>{
 await page.setViewportSize({width,height:900});
 await page.goto('/abyss/rift?scenario=long-boss-name');await expect(page.locator('#rift-start')).toBeEnabled();
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 const name='AncientColossus'.repeat(20);
 await page.evaluate(({run,name})=>{document.getElementById('rift-overlay').hidden=true;const boss=run.enemies.find(e=>e.kind==='boss');boss.name=name;boss.phase=3;boss.windup=.5;boss.attack_name='Ground Slam';window.RiftHUD.update(run,true);},{run,name});
 const bossName=page.locator('#rift-boss-name'),meter=page.locator('#rift-boss .hp'),phase=page.locator('#rift-boss-phase');
 await expect(bossName).toHaveAttribute('title',name);await expect(meter).toBeVisible();await expect(phase).toBeVisible();
 const n=await bossName.boundingBox(),m=await meter.boundingBox(),p=await phase.boundingBox();
 expect(m.width).toBeGreaterThan(40);expect(m.height).toBeGreaterThanOrEqual(8);
 expect(n.y+n.height).toBeLessThanOrEqual(m.y+.5);expect(m.y+m.height).toBeLessThanOrEqual(p.y+.5);
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
