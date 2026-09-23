const {test,expect}=require('@playwright/test');
const fs=require('fs');
test('each subclass has a distinct stable reduced-motion victory stance',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const results=await page.evaluate(async()=>{
  const base=(await(await fetch('/api/abyss/rift')).json()).run,results=[];window.RiftRenderer.reduced=true;
  for(const name of ['vanguard','berserker','marksman','beastmaster','elementalist','chronomancer','oracle','geomancer','bloodblade','voidwalker','runesmith','alchemist']){
   const run=structuredClone(base);run.status='complete';run.player.pose='victory';run.player.pose_time=3;run.player.guard=false;run.player.jump=0;run.player.x=400;run.player.y=410;run.build.class=name;run.player.kind=name;run.enemies=[];run.events=[];
   window.RiftRenderer.snapshot(run,true);await new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve)));const first={...window.RiftRenderer.lastVictoryPose};await new Promise(resolve=>setTimeout(resolve,70));results.push([first,{...window.RiftRenderer.lastVictoryPose}]);
  }return results;
 });
 expect(new Set(results.map(([r])=>r.stance)).size).toBe(12);expect(new Set(results.map(([r])=>[r.frame,r.angle,r.lift].join(','))).size).toBe(12);
 for(const [first,last] of results){expect(first.reduced).toBe(true);expect(last.lift).toBe(first.lift);expect(last.angle).toBe(first.angle);expect(first.color).toMatch(/^#[0-9a-f]{6}$/);}
 const png=await page.locator('#rift-canvas').evaluate(c=>c.toDataURL('image/png'));fs.writeFileSync('test-results/class-victory-reduced.png',Buffer.from(png.split(',')[1],'base64'));
});
