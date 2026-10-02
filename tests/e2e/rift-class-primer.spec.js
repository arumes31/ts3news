const {test,expect}=require('@playwright/test');
for(const subclass of ['vanguard','berserker','bloodblade','marksman','beastmaster','elementalist','chronomancer','oracle','geomancer','voidwalker','runesmith','alchemist'])test('entry primer uses '+subclass+' abilities',async({page})=>{
 await page.goto('/abyss/rift?subclass='+subclass);await expect(page.locator('#rift-start')).toBeEnabled();const {build}=await(await page.request.get('/api/abyss/rift')).json();const primer=page.locator('#rift-class-primer');await expect(primer).toBeVisible();await expect(primer.locator('strong')).toContainText(build.class_name);for(const skill of build.signatures)await expect(primer).toContainText(skill.name);await expect(primer).toContainText('three');
 if(subclass==='vanguard'){
  await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await primer.scrollIntoViewIfNeeded();await primer.screenshot({path:'test-results/class-primer-mobile.png'});
  await page.locator('#rift-start').click();await expect(primer).toBeHidden();await page.keyboard.press('Escape');await expect(primer).toBeVisible();
 }
});

test('primer keeps saved loadout and follows remapped keys',async({page})=>{
 await page.route('**/api/abyss/rift',async route=>{const response=await route.fetch();const data=await response.json();if(data.build)data.build={...data.build,class_name:'Different current class',signatures:[]};await route.fulfill({response,json:data});});
 await page.goto('/abyss/rift?scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();const {run}=await(await page.request.get('/api/abyss/rift')).json();const primer=page.locator('#rift-class-primer');await expect(primer).toContainText(run.build.class_name);await expect(primer).not.toContainText('Different current class');
 await page.locator('#rift-controls-open').click();await page.locator('[data-remap="signature0"]').click();await page.keyboard.press('t');await page.locator('#rift-controls-close').click();await expect(primer).toContainText('(T)');
});
