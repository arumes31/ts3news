const {test,expect}=require('@playwright/test');

test('mission scroll survives campaign closure, overview browsing and reload with saved filters',async({page})=>{
  await page.setViewportSize({width:390,height:844});
  await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-region').selectOption('8');
  const grid=page.locator('#rift-levels');
  await grid.evaluate(element=>{element.scrollTop=300;});
  await expect.poll(()=>grid.evaluate(element=>element.scrollTop)).toBe(300);
  // Wait for the browser scroll event before hiding the scroll container.
  await page.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));
  await page.locator('#rift-campaign > summary').click();
  await expect(page.locator('#rift-campaign')).not.toHaveAttribute('open','');
  await page.locator('#rift-campaign > summary').click();
  await expect.poll(()=>grid.evaluate(element=>element.scrollTop)).toBe(300);
  await page.locator('#rift-overview-toggle').click();await expect(grid).toBeHidden();
  await page.locator('#rift-overview-toggle').click();
  await expect.poll(()=>grid.evaluate(element=>element.scrollTop)).toBe(300);
  await page.reload();await expect(page.locator('#rift-region')).toHaveValue('8');
  await expect.poll(()=>grid.evaluate(element=>element.scrollTop)).toBe(300);
});

test('invalid saved mission scroll is ignored and oversized positions are bounded by the grid',async({page})=>{
  await page.addInitScript(()=>localStorage.setItem('riftCampaignView',JSON.stringify({scrollTop:'300'})));
  await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
  await expect.poll(()=>page.locator('#rift-levels').evaluate(element=>element.scrollTop)).toBe(0);
  await page.evaluate(()=>localStorage.setItem('riftCampaignView',JSON.stringify({scrollTop:1e12})));
  // A fresh navigation without the initial script uses a separate page in the same context.
  const restored=await page.context().newPage();await restored.goto('/abyss/rift');
  await expect(restored.locator('#rift-start')).toBeEnabled();
  await expect.poll(()=>restored.locator('#rift-levels').evaluate(element=>element.scrollTop===element.scrollHeight-element.clientHeight)).toBe(true);
  await restored.close();
});

test('region overview expands the selected region and preserves combined filters when browsing',async({page})=>{
  await page.goto('/abyss/rift?mission=87');await expect(page.locator('#rift-start')).toHaveText('Enter mission 87');
  await page.locator('#rift-mission-search').fill('impossible-mission');
  await page.locator('#rift-difficulty').selectOption('Mythic');
  await expect(page.locator('#rift-selected-mission')).toContainText('Selected mission: 87');
  await page.locator('#rift-overview-toggle').click();
  const overview=page.getByRole('navigation',{name:'Campaign regions'});
  await expect(overview.getByRole('region')).toHaveCount(10);
  await expect(page.locator('#rift-levels')).toBeHidden();
  await expect(page.locator('[data-overview-region="8"] details')).toHaveAttribute('open','');
  await expect(page.locator('[data-overview-region="8"] p')).toContainText('Selected mission: 87');
  await page.locator('[data-overview-region="2"] summary').click();
  await page.locator('[data-overview-region="2"] button').click();
  await expect(overview).toBeHidden();await expect(page.locator('#rift-region')).toHaveValue('2');
  await expect(page.locator('#rift-mission-search')).toHaveValue('impossible-mission');
  await expect(page.locator('#rift-difficulty')).toHaveValue('Mythic');
  await expect(page.locator('#rift-filter-count')).toHaveText('0 of 100 missions');
  await expect(page.locator('#rift-selected-mission')).toContainText('Selected mission: 87');
  await page.locator('#rift-clear-filters').click();await expect(page.locator('#rift-selected-mission')).toBeHidden();
  await page.locator('[data-level="21"]').click();await page.locator('#rift-overview-toggle').click();
  await expect(page.locator('[data-overview-region="2"] details')).toHaveAttribute('open','');
  await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});

test('active expedition remains visible and its region opens despite a different incoming mission link',async({page})=>{
  await page.goto('/abyss/rift?mission=21');await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');
  await page.goto('/abyss/rift?mission=87');await page.locator('#rift-overview-toggle').click();
  await expect(page.locator('#rift-selected-mission')).toContainText('Active expedition: 21');
  await expect(page.locator('[data-overview-region="2"] details')).toHaveAttribute('open','');
  expect((await(await page.request.get('/api/abyss/rift')).json()).run.level.id).toBe(21);
});

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
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');
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
