const {test,expect}=require('@playwright/test');
const fs=require('node:fs');
for(const reduced of [false,true])test('Elementalist reaction setup and confirmed impact, reduced='+reduced,async({page})=>{
 await page.emulateMedia({reducedMotion:reduced?'reduce':'no-preference'});await page.goto('/abyss/rift?subclass=elementalist&scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();const {run}=await(await page.request.get('/api/abyss/rift')).json();const node=page.locator('#rift-elemental-reaction');await expect(node).toContainText('Reaction setup');
 await page.evaluate(run=>{const target=run.enemies.find(e=>e.hp>0);target.x=420;target.y=405;run.enemies=[target];run.marked=target.id;run.resource=2;run.paused=true;run.events=[];run.player.x=160;RiftHUD.update(run,false);RiftRenderer.snapshot(run,true);window.reactionLabels=0;const fill=CanvasRenderingContext2D.prototype.fillText;CanvasRenderingContext2D.prototype.fillText=function(text,...args){if(text==='REACTION ×1.2')reactionLabels++;return fill.call(this,text,...args);};run.counter++;run.events=[{id:run.counter,kind:'elemental_reaction',x:420,y:335,value:20}];RiftRenderer.snapshot(run,false);},run);
 await expect(node).toContainText('Reaction armed');await expect.poll(()=>page.evaluate(()=>reactionLabels)).toBeGreaterThan(0);
 const png=await page.locator('#rift-canvas').evaluate(c=>c.toDataURL('image/png'));fs.writeFileSync('test-results/elemental-reaction'+(reduced?'-reduced':'')+'.png',Buffer.from(png.split(',')[1],'base64'));
 await page.evaluate(run=>{run.build.class='marksman';RiftHUD.update(run,false);},run);await expect(node).toBeHidden();
});
test('reaction sound stops cleanly on pause',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();expect(await page.evaluate(async()=>{await RiftAudio.setActive(true,0);const before=RiftAudio.voices;RiftAudio.play('elemental_reaction',0);const added=RiftAudio.voices-before;await RiftAudio.setActive(false);return {added,left:RiftAudio.voices};})).toEqual({added:2,left:0});
});
