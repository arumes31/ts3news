const {test,expect}=require('@playwright/test');
for(const mode of ['skills','class','ranged'])test('practice range previews preserve '+mode+' drill state',async({page})=>{
 await page.goto('/abyss/rift?practice='+mode);await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');const read=async()=>(await(await page.request.get('/api/abyss/rift?practice='+mode)).json()).run;const saved=await read();
 const skills=[...saved.build.skills,...saved.build.signatures,...(saved.build.ultimate?[saved.build.ultimate]:[])],toggle=page.locator('#rift-range-toggle'),signal=page.locator('#rift-skill-range');
 for(const skill of skills){await toggle.click();await expect(signal).toContainText('Range: '+skill.name);if(['area','projectile'].includes(skill.reference.target))await expect.poll(()=>page.evaluate(()=>RiftRenderer.lastRangePreview?.target)).toBe(skill.reference.target);}
 await toggle.click();await expect(signal).toHaveText('Skill range: None requested');await expect.poll(()=>page.evaluate(()=>RiftRenderer.lastRangePreview)).toBe(null);
 expect(await read()).toEqual(saved);await page.setViewportSize({width:390,height:1600});await toggle.click();await expect(signal).toContainText('Range: '+skills[0].name);expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
