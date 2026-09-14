const {test,expect}=require('@playwright/test');

test('bestiary training details expose shared windup and interruption rules',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();const {bestiary}=await(await page.request.get('/api/abyss/rift')).json();await page.locator('.rift-bestiary > summary').click();
 for(const boss of [false,true]){
  const unit=bestiary.find(unit=>(unit.kind==='boss')===boss);await page.getByRole('button',{name:'Inspect '+unit.name,exact:true}).click();
  const stats=page.locator('#rift-monster-stats');await expect(stats).toContainText('Attack windup');await expect(stats).toContainText(unit.training.windup_seconds+' s');await expect(stats).toContainText(boss?'No: resists basic-combo and ice interrupts':'Yes: third basic strike or ice');
 }
});

test('undiscovered state explains how to record monsters and future roster entries need no registration',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();await page.locator('.rift-bestiary > summary').click();await page.locator('#rift-monster-seen').check();await expect(page.locator('#rift-monsters-empty')).toContainText('Start a campaign expedition');await page.locator('#rift-monster-clear').click();await expect(page.locator('#rift-catalog-help')).toContainText('new missions use the current catalog');
 const future={...data.bestiary[0],id:'future-roster-entry',name:'Future Abyss Wanderer',art_key:'monster:Future Abyss Wanderer'};
 await page.evaluate(({roster,run})=>window.RiftBestiary.render(roster,run),{roster:[...data.bestiary,future],run:data.run});
 await expect(page.locator('#rift-monsters article:visible')).toHaveCount(data.bestiary.length+1);await page.getByRole('button',{name:'Inspect '+future.name,exact:true}).click();await expect(page.locator('#rift-monster-title')).toHaveText(future.name);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
