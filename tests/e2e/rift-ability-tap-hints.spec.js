const {test,expect}=require('@playwright/test');
test.use({hasTouch:true,isMobile:true,viewport:{width:390,height:844}});
for(const state of ['mana','cooldown'])test('tapped ability shows '+state+' hint beside its label',async({page})=>{
 let blocked=true;const requested=[];
 await page.route('**/api/abyss/rift*',async route=>{
  const body=route.request().postDataJSON();if(body?.kind==='step'&&body.input.skill)requested.push(body.input.skill);
  const response=await route.fetch(),data=await response.json();
  if(data.run){const skill=data.run.build.skills[0];data.run.player.mana=blocked&&state==='mana'?0:100;data.run.skill_timers[skill.id]=blocked&&state==='cooldown'?8:0;}
  await route.fulfill({response,json:data});
 });
 await page.goto('/abyss/rift?practice=skills');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').tap();await expect(page.locator('#rift-overlay')).toBeHidden();
 const button=page.locator('#rift-skills button').first(),hint=button.locator('.rift-ability-rejection');
 await button.scrollIntoViewIfNeeded();const box=await button.boundingBox();await page.touchscreen.tap(box.x+box.width/2,box.y+box.height/2);await expect(hint).toBeVisible();await expect(hint).toContainText(state==='mana'?'more mana needed':'seconds cooldown');
 await expect(button).toHaveAccessibleName(state==='mana'?/more mana needed/:/seconds cooldown/);
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await expect(button).toHaveCSS('opacity','1');
 await button.screenshot({path:test.info().outputPath('tap-hint.png')});
 await expect(hint).toBeHidden({timeout:3500});
 const again=await button.boundingBox();await page.touchscreen.tap(again.x+again.width/2,again.y+again.height/2);await expect(hint).toBeVisible();
 blocked=false;await expect(hint).toBeHidden();expect(requested).toEqual([]);
 await page.locator('#rift-pause').tap();await expect(hint).toBeHidden();
});
