const {test,expect}=require('@playwright/test');
test('aim hints explain facing and exact lane boundaries without mislabeling area or self skills',async({page})=>{
 await page.goto('/abyss/rift?scenario=spawn-hazards&subclass=elementalist');await expect(page.locator('#rift-start')).toBeEnabled();const saved=(await(await page.request.get('/api/abyss/rift')).json()).run;
 const signal=page.locator('#rift-skill-range');
 const show=async(target,dx,dy,facing=1)=>page.evaluate(({saved,target,dx,dy,facing})=>{
  const run=structuredClone(saved);run.player.x=500;run.player.y=400;run.player.facing=facing;run.enemies=[{...run.enemies[0],id:'aim-target',name:'Training target',x:500+dx,y:400+dy,hp:100}];
  const skill={id:'aim-preview',name:'Test skill',kind:target==='area'?'slash':target==='self'?'heal':'fire',reference:{target,horizontal:target==='area'?155:35,depth:target==='self'?0:60}};
  RiftHUD.update(run,false);RiftHUD.setRequestedRange(skill);
 },{saved,target,dx,dy,facing});
 await show('projectile',-100,80);await expect(signal).toContainText('behind you: turn left');await expect(signal).toContainText('move down');
 await show('projectile',100,-60,-1);await expect(signal).toContainText('turn right');await expect(signal).toContainText('move up');
 await show('projectile',100,59.9);await expect(signal).toContainText('currently aligned');await expect(signal).not.toContainText('outside the lane');
 await show('area',-100,0);await expect(signal).toContainText('either facing direction');await expect(signal).not.toContainText('behind you');
 await show('area',155,60);await expect(signal).toContainText('outside the lane');await expect(signal).toContainText('too far horizontally');
 await show('self',-100,200);await expect(signal).toContainText('cannot cause a miss');await expect(signal).not.toContainText('outside the lane');
 await show('projectile',-100,80);await page.setViewportSize({width:390,height:1600});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await signal.scrollIntoViewIfNeeded();await page.evaluate(()=>window.scrollBy(0,-240));await signal.screenshot({path:'test-results/skill-aim-mobile.png'});
 await page.evaluate(()=>RiftHUD.setRequestedRange(null));await expect(signal).toHaveText('Skill range: None requested');expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(saved);
});
