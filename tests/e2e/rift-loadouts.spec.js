const {test,expect}=require('@playwright/test');

test('named presets review changes before apply, persist, rename and delete without changing current skills',async({page})=>{
  await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
  const values=()=>page.locator('#rift-loadout select').evaluateAll(slots=>slots.map(slot=>slot.value));
  const original=await values();await page.locator('#rift-loadout-name').fill('Guardian hunter');await page.locator('#rift-save-loadout').click();
  await expect(page.locator('#rift-loadout-status')).toHaveText('Preset saved locally.');
  for(const slot of await page.locator('#rift-loadout select').all())await slot.selectOption('');
  await expect(page.locator('#rift-loadout-review')).toContainText('Slot 1: None → Iron Guard');expect(await values()).toEqual(['','','']);
  await page.locator('#rift-apply-loadout').click();expect(await values()).toEqual(original);
  await page.locator('#rift-loadout-name').fill('Boss kit');await page.locator('#rift-rename-loadout').click();
  await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-preset-list').selectOption({label:'Boss kit'});
  await expect(page.locator('#rift-loadout-name')).toHaveValue('Boss kit');
  await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
  await page.locator('#rift-delete-loadout').click();expect(await values()).toEqual(original);await expect(page.locator('#rift-preset-list option')).toHaveCount(1);
  await page.reload();await expect(page.locator('#rift-preset-list option')).toHaveCount(1);
});

test('unavailable and duplicate skills cannot apply and active expeditions lock preset changes',async({page})=>{
  await page.addInitScript(()=>localStorage.setItem('riftLoadoutPresets',JSON.stringify([
    {id:'missing',name:'Old skills',skills:['removed_skill']},{id:'duplicate',name:'Duplicate',skills:['guard','guard']},{id:'owned',name:'Owned',skills:['guard','bash','spark']}
  ])));
  await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-preset-list').selectOption('missing');await expect(page.locator('#rift-loadout-review')).toContainText('Unavailable skill: removed_skill');await expect(page.locator('#rift-apply-loadout')).toBeDisabled();
  await page.locator('#rift-preset-list').selectOption('duplicate');await expect(page.locator('#rift-apply-loadout')).toBeDisabled();
  await page.locator('#rift-preset-list').selectOption('owned');await expect(page.locator('#rift-apply-loadout')).toBeEnabled();
  await page.locator('#rift-start').click();await page.keyboard.press('Escape');
  await expect(page.locator('#rift-apply-loadout')).toBeDisabled();await expect(page.locator('#rift-save-loadout')).toBeDisabled();
  const before=(await(await page.request.get('/api/abyss/rift')).json()).run.build.skills;
  await page.locator('#rift-apply-loadout').evaluate(button=>button.click());
  expect((await(await page.request.get('/api/abyss/rift')).json()).run.build.skills).toEqual(before);
});
