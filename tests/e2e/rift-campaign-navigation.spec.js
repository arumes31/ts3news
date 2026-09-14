const {test,expect}=require('@playwright/test');

test('mission keyboard browsing follows filtered cards without changing selection until activation',async({page})=>{
  await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-region').selectOption('2');
  const cards=page.locator('#rift-levels [data-level]:visible');await expect(cards).toHaveCount(10);
  await cards.first().focus();await page.keyboard.press('ArrowRight');await expect(cards.nth(1)).toBeFocused();
  await expect(page.locator('[data-level="1"]')).toHaveAttribute('aria-current','true');
  await page.keyboard.press('End');await expect(cards.last()).toBeFocused();
  await page.keyboard.press('Home');await expect(cards.first()).toBeFocused();
  const nextRow=await cards.evaluateAll(elements=>{
    const first=elements[0].getBoundingClientRect();return elements.findIndex(element=>element.getBoundingClientRect().top>first.top+1);
  });
  expect(nextRow).toBeGreaterThan(0);
  await page.keyboard.press('ArrowDown');await expect(cards.nth(nextRow)).toBeFocused();
  await page.keyboard.press('ArrowUp');await expect(cards.first()).toBeFocused();
  await page.keyboard.press('Enter');await expect(page.locator('#rift-start')).toHaveText('Enter mission 21');
  await page.locator('#rift-start').click();await page.keyboard.press('Escape');
  await expect(page.locator('[data-level="22"]')).toBeDisabled();
  expect((await(await page.request.get('/api/abyss/rift')).json()).run.level.id).toBe(21);
});

test('bookmark names identify the selected mission and saved state across reloads',async({page})=>{
  await page.goto('/abyss/rift?mission=87');await expect(page.locator('#rift-start')).toHaveText('Enter mission 87');
  const level=(await(await page.request.get('/api/abyss/rift')).json()).levels.find(level=>level.id===87);
  const bookmark=page.locator('#rift-favorite');
  await expect(bookmark).toHaveAccessibleName('Favorite mission 87: '+level.name);
  await bookmark.click();await expect(bookmark).toHaveAccessibleName('Remove favorite 87: '+level.name);
  await expect(bookmark).toHaveAttribute('aria-pressed','true');
  await page.reload();await expect(bookmark).toHaveAccessibleName('Remove favorite 87: '+level.name);
  await page.setViewportSize({width:390,height:844});
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
  await bookmark.click();await expect(bookmark).toHaveAttribute('aria-pressed','false');
});
