const {test,expect}=require('@playwright/test');
test.use({hasTouch:true,isMobile:true,viewport:{width:320,height:568}});
for(const mode of ['', '?practice=skills'])test('touch descriptions are readable and pause combat '+(mode||'expedition'),async({page})=>{
 await page.goto('/abyss/rift'+mode);await expect(page.locator('#rift-start')).toBeEnabled();
 await page.locator('#rift-start').tap();await expect(page.locator('#rift-overlay')).toBeHidden();
 const open=page.getByRole('button',{name:'Skill descriptions',exact:true});
 await expect(open).toBeVisible();expect((await open.boundingBox()).height).toBeGreaterThanOrEqual(44);
 await open.tap();await expect(page.locator('#rift-overlay')).toBeVisible();
 await expect(page.locator('#rift-skill-glossary')).toHaveAttribute('open','');
 await expect(page.locator('#rift-skill-glossary > summary')).toBeFocused();
 const data=await(await page.request.get('/api/abyss/rift'+mode)).json();
 expect(data.run.paused).toBe(true);
 const build=data.run.build;
 for(const skill of [...build.skills,...build.signatures,build.ultimate].filter(Boolean)){
  const entry=page.locator('#rift-glossary-entries article[data-skill="'+skill.id+'"]');
  await entry.locator('.rift-skill-description').scrollIntoViewIfNeeded();
  await expect(entry.locator('.rift-skill-description')).toBeVisible();
  await expect(entry).toContainText(skill.cost+' MP');
  expect(await entry.locator('.rift-skill-description').evaluate(node=>parseFloat(getComputedStyle(node).fontSize))).toBeGreaterThanOrEqual(14);
 }
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await page.locator('#rift-start').tap();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.locator('#rift-pause').tap();
});
