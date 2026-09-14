const {test,expect}=require('@playwright/test');

test('mission links select the route and previews match all three saved terrain layouts',async({page})=>{
  await page.goto('/abyss/rift?mission=87');await expect(page.locator('#rift-start')).toHaveText('Enter mission 87');
  await expect(page.locator('[data-level="87"]')).toHaveAttribute('aria-current','true');
  const level=(await(await page.request.get('/api/abyss/rift')).json()).levels.find(level=>level.id===87);
  await page.locator('#rift-mission-preview > summary').click();
  const rooms=page.locator('#rift-room-previews article');await expect(rooms).toHaveCount(3);
  for(let i=0;i<3;i++){
    await expect(rooms.nth(i).locator('h4')).toContainText(level.rooms[i].name);
    await expect(rooms.nth(i).locator('[data-terrain="cover"]')).toHaveCount(level.rooms[i].obstacles.length);
    await expect(rooms.nth(i).locator('[data-terrain="hazard"]')).toHaveCount(level.rooms[i].hazards.length);
    expect(await rooms.nth(i).locator('[data-terrain="cover"]').first().getAttribute('x')).toBe(String(level.rooms[i].obstacles[0].x));
  }
  await expect(page.locator('#rift-mission-class')).toContainText('Vanguard');
  await expect(page.locator('#rift-mission-link')).toHaveValue(/\/abyss\/rift\?mission=87$/);
  await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});

test('invalid mission links fall back and active expeditions take priority over incoming links',async({page})=>{
  await page.goto('/abyss/rift?mission=999');await expect(page.locator('#rift-mission-link-status')).toContainText('invalid');await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('[data-level="12"]').click();await page.locator('#rift-start').click();await page.keyboard.press('Escape');
  await page.goto('/abyss/rift?mission=65');await expect(page.locator('[data-level="12"]')).toHaveAttribute('aria-current','true');
  await expect(page.locator('#rift-mission-link-status')).toContainText('saved expedition');
  expect((await(await page.request.get('/api/abyss/rift')).json()).run.level.id).toBe(12);
});

test('route preview preference survives reload and link copy uses only the mission',async({page})=>{
  await page.goto('/abyss/rift?mission=21&subclass=elementalist');await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-preview-default').check();await expect(page.locator('#rift-mission-preview')).toHaveAttribute('open','');
  await page.reload();await expect(page.locator('#rift-mission-preview')).toHaveAttribute('open','');
  await page.evaluate(()=>Object.defineProperty(navigator,'clipboard',{configurable:true,value:{writeText:async value=>window.copiedMission=value}}));
  await page.locator('#rift-copy-mission').click();expect(await page.evaluate(()=>window.copiedMission)).toMatch(/\/abyss\/rift\?mission=21$/);
  await expect(page.locator('#rift-mission-link')).not.toHaveValue(/subclass/);
});
