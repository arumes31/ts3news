const {test,expect}=require('@playwright/test');

test('direct slot swaps preserve skills, save their order and lock during expeditions',async({page})=>{
  await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
  const values=()=>page.locator('#rift-loadout select').evaluateAll(slots=>slots.map(slot=>slot.value));
  const swap=page.getByRole('button',{name:'Swap slots 1 and 2',exact:true});
  await swap.focus();await page.keyboard.press('Enter');expect(await values()).toEqual(['bash','guard','spark']);await expect(swap).toBeFocused();
  await page.getByRole('button',{name:'Swap slots 2 and 3',exact:true}).click();expect(await values()).toEqual(['bash','spark','guard']);
  await page.locator('#rift-loadout-name').fill('Swap test');await page.locator('#rift-save-loadout').click();
  await swap.click();await page.locator('#rift-apply-loadout').click();expect(await values()).toEqual(['bash','spark','guard']);
  await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
  await page.locator('#rift-start').click();await page.keyboard.press('Escape');await expect(swap).toBeDisabled();
  expect((await(await page.request.get('/api/abyss/rift')).json()).run.build.skills.map(skill=>skill.id)).toEqual(['bash','spark','guard']);
});

test('preset transfer rejects invalid skills and imports owned slots for review before application',async({page})=>{
  await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-loadout-presets details > summary').click();
  const values=()=>page.locator('#rift-loadout select').evaluateAll(slots=>slots.map(slot=>slot.value));const original=await values();
  for(const payload of ['bad JSON',JSON.stringify({version:2,name:'Future',skills:[]}),JSON.stringify({version:1,name:'Missing',skills:['not_owned']}),JSON.stringify({version:1,name:'Duplicate',skills:['guard','guard']})]){
    await page.locator('#rift-preset-json').fill(payload);await page.locator('#rift-import-loadout').click();
    await expect(page.locator('#rift-preset-list option')).toHaveCount(1);expect(await values()).toEqual(original);
  }
  const preset={version:1,name:'Reordered',skills:['spark','guard','bash']};
  await page.locator('#rift-preset-json').fill(JSON.stringify(preset));await page.locator('#rift-import-loadout').click();
  await expect(page.locator('#rift-loadout-status')).toContainText('Review the changes');expect(await values()).toEqual(original);
  await expect(page.locator('#rift-loadout-review')).toContainText('Iron Guard → Cinder Bolt');
  await page.locator('#rift-apply-loadout').click();expect(await values()).toEqual(preset.skills);
  await page.locator('#rift-export-loadout').click();expect(JSON.parse(await page.locator('#rift-preset-json').inputValue())).toEqual(preset);
  await page.locator('#rift-start').click();await page.keyboard.press('Escape');await expect(page.locator('#rift-import-loadout')).toBeDisabled();
  expect((await(await page.request.get('/api/abyss/rift')).json()).run.build.skills.map(skill=>skill.id)).toEqual(preset.skills);
});

test('empty optional loadouts explain retained controls and can be saved and applied',async({page})=>{
  await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-empty-loadout')).toBeHidden();
  for(const slot of await page.locator('#rift-loadout select').all())await slot.selectOption('');
  await expect(page.locator('#rift-empty-loadout')).toContainText('Basic attacks, jumping and guarding remain available');
  await page.locator('#rift-loadout-name').fill('Basics');await page.locator('#rift-save-loadout').click();
  await page.locator('#rift-loadout select').first().selectOption('guard');await expect(page.locator('#rift-empty-loadout')).toBeHidden();
  await page.locator('#rift-apply-loadout').click();await expect(page.locator('#rift-empty-loadout')).toBeVisible();
  await page.setViewportSize({width:390,height:844});await page.locator('#rift-loadout-presets details > summary').click();expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});

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
