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


test('bestiary previews authoritative upcoming boss phase timings',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 const {bestiary}=await(await page.request.get('/api/abyss/rift')).json();
 const boss=bestiary.find(unit=>unit.kind==='boss');
 expect(boss.training.boss_phases).toEqual([{phase:1,at_health_percent:100,windup_seconds:1.15},{phase:2,at_health_percent:50,windup_seconds:.98},{phase:3,at_health_percent:25,windup_seconds:.85}]);
 await page.locator('.rift-bestiary > summary').click();await page.getByRole('button',{name:'Inspect '+boss.name,exact:true}).click();
 const stats=page.locator('#rift-monster-stats');
 for(const phase of boss.training.boss_phases){
  const row=stats.locator('div').filter({has:page.locator('dt',{hasText:'Phase '+phase.phase})});
  await expect(row).toContainText(phase.phase===1?'Opening':phase.at_health_percent+'% HP');
  await expect(row).toContainText(new Intl.NumberFormat('en-US',{maximumFractionDigits:2}).format(phase.windup_seconds)+' s windup');
 }
});
