const {test,expect}=require('@playwright/test');
test('ability and reference icons follow catalog payloads while retaining accessible names',async({page})=>{
 await page.goto('/abyss/rift?scenario=spawn-hazards&subclass=geomancer');await expect(page.locator('#rift-start')).toBeEnabled();const {run,build}=await(await page.request.get('/api/abyss/rift')).json();
 const abilities=[...run.build.skills,...run.build.signatures,...(run.build.ultimate?[run.build.ultimate]:[])];
 const buttons=page.locator('#rift-skills button, #rift-signatures button');await expect(buttons).toHaveCount(abilities.length);
 for(let i=0;i<abilities.length;i++){await expect(buttons.nth(i).locator('.rift-catalog-icon')).toHaveText(abilities[i].icon);await expect(buttons.nth(i).locator('.rift-catalog-icon')).toHaveAttribute('aria-hidden','true');await expect(buttons.nth(i)).toHaveAccessibleName(new RegExp(abilities[i].name));}
 await page.locator('#rift-loadout-preview').click();for(const skill of [...build.skills,...build.signatures,...(build.ultimate?[build.ultimate]:[])])await expect(page.locator('#rift-glossary-entries article').filter({has:page.locator('strong',{hasText:skill.name})}).locator('.rift-catalog-icon')).toHaveText(skill.icon);
 await page.evaluate(run=>{run.build.skills[0].icon='⬇';RiftHUD.update(run,false);},run);await expect(buttons.first().locator('.rift-catalog-icon')).toHaveText('⬇');
 expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(run);
});
