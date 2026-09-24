const {test,expect}=require('@playwright/test');
test('boss practice explains observed telegraphs, recovery and normal-speed projectiles',async({page})=>{
 await page.goto('/abyss/rift?practice=boss');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-practice-slow').check();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');const saved=(await(await page.request.get('/api/abyss/rift?practice=boss')).json()).run;
 const lesson=page.locator('#rift-boss-practice-lesson');
 const show=async(attacks,windup,cooldown,completed=false)=>page.evaluate(({run,attacks,windup,cooldown,completed})=>{run.practice.completed=completed;const boss=run.enemies[0];boss.art_key='training-boss';boss.attacks=attacks;boss.windup=windup;boss.cooldown=cooldown;boss.attack_name='';RiftHUD.update(run,false);},{run:saved,attacks,windup,cooldown,completed});
 await show(0,.5,0);await expect(lesson).toContainText('ground-attack warning');await expect(lesson).toContainText('Jumping too early');await expect(lesson).toContainText('projectile travel and recovery keep their normal speed');
 await show(1,.5,0);await expect(lesson).toContainText('projectile warning');await expect(lesson).toContainText('face incoming shots and guard');
 await show(1,0,1);await expect(lesson).toContainText('Projectiles already in flight');await show(1,0,0);await expect(lesson).toContainText('Identify a ground attack');await show(1,0,0,true);await expect(lesson).toContainText('Boss defeated');
 await page.setViewportSize({width:390,height:1600});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await page.evaluate(run=>{run.practice.mode='guard';RiftHUD.update(run,false);},saved);await expect(lesson).toBeHidden();expect((await(await page.request.get('/api/abyss/rift?practice=boss')).json()).run).toEqual(saved);
});
