const {test,expect}=require('@playwright/test');
test.use({hasTouch:true,isMobile:true});

async function smallTargets(page,scope='#rift-app'){
 return page.locator(scope).evaluate(root=>{
  const visible=node=>{const box=node.getBoundingClientRect(),style=getComputedStyle(node);return box.width>0&&box.height>0&&style.visibility!=='hidden'&&!node.closest('[hidden]');};
  return [...root.querySelectorAll('button,select,input,summary,a[href]:not(.rift-field-guide a)')].filter(visible).flatMap(node=>{
   if(node.type==='hidden')return [];
   const target=['checkbox','radio'].includes(node.type)?node.labels?.[0]||node:node,box=target.getBoundingClientRect();
   return box.width>=47.5&&box.height>=47.5?[]:[{id:node.id||node.getAttribute('aria-label')||node.textContent.trim().slice(0,60),tag:node.tagName,type:node.type,width:Math.round(box.width*10)/10,height:Math.round(box.height*10)/10}];
  });
 });
}
for(const viewport of [{width:390,height:844},{width:844,height:390},{width:1280,height:900}]){
 test('touch targets cover settings, mission selection, combat and controls at '+viewport.width+'x'+viewport.height,async({page},testInfo)=>{
  await page.setViewportSize(viewport);await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
  expect(await page.evaluate(()=>matchMedia('(any-pointer: coarse)').matches)).toBe(true);
  await page.evaluate(()=>document.querySelectorAll('#rift-app details').forEach(node=>node.open=true));
  const ordinary=await smallTargets(page);expect.soft(ordinary).toEqual([]);
  expect.soft(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
  await page.locator('#rift-controls-open').tap();await expect(page.locator('#rift-controls-dialog')).toBeVisible();
  const controls=await smallTargets(page,'#rift-controls-dialog');expect.soft(controls).toEqual([]);
  await page.locator('#rift-controls-close').tap();
  const label=page.locator('label[for="rift-loot-sparkle"]');const previous=await page.locator('#rift-loot-sparkle').isChecked();
  await label.tap({position:{x:8,y:8}});await expect(page.locator('#rift-loot-sparkle')).toBeChecked({checked:!previous});
  await page.locator('.rift-display-settings').first().screenshot({path:testInfo.outputPath('touch-settings.png')});
 });
}


for(const practice of ['boss','hazard','skills']){
 test('touch targets include '+practice+' practice options',async({page})=>{
  await page.setViewportSize({width:390,height:844});await page.goto('/abyss/rift?practice='+practice);await expect(page.locator('#rift-start')).toBeEnabled();
  expect(await smallTargets(page)).toEqual([]);
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 });
}
test('touch targets include checkpoint banking and the reward receipt',async({page})=>{
 await page.setViewportSize({width:390,height:844});await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.locator('#rift-auto').uncheck();await page.locator('#rift-start').tap();await expect(page.locator('#rift-overlay')).toBeHidden();
 expect(await smallTargets(page)).toEqual([]);
 await page.locator('#rift-next').tap();await expect(page.locator('#rift-receipt')).toBeVisible();await page.locator('#rift-pause').tap();
 await page.locator('#rift-receipt > summary').tap();expect(await smallTargets(page)).toEqual([]);
});
