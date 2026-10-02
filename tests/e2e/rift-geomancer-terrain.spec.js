const {test,expect}=require('@playwright/test');
const fs=require('node:fs');
for(const reduced of [false,true])test('Geomancer terrain pulse uses confirmed impact, reduced='+reduced,async({page})=>{
 await page.emulateMedia({reducedMotion:reduced?'reduce':'no-preference'});await page.goto('/abyss/rift?subclass=geomancer&scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();const {run}=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(run=>{run.player.x=160;run.enemies=[];run.paused=true;run.events=[];RiftRenderer.snapshot(run,true);window.earthPulse=[];const ellipse=CanvasRenderingContext2D.prototype.ellipse;CanvasRenderingContext2D.prototype.ellipse=function(...args){if(this.strokeStyle==='#e7c18b')earthPulse=args;return ellipse.apply(this,args);};run.counter++;run.events=[{id:run.counter,kind:'geomancer_terrain',x:420,y:410,value:0}];RiftRenderer.snapshot(run,false);},run);
 await expect.poll(()=>page.evaluate(()=>earthPulse.slice(0,2))).toEqual([420,410]);if(reduced)expect(await page.evaluate(()=>earthPulse[2])).toBe(28);
 const png=await page.locator('#rift-canvas').evaluate(c=>c.toDataURL('image/png'));fs.writeFileSync('test-results/geomancer-terrain'+(reduced?'-reduced':'')+'.png',Buffer.from(png.split(',')[1],'base64'));
});
test('Geomancer terrain cue has a bounded sound and pause cleanup',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();expect(await page.evaluate(async()=>{await RiftAudio.setActive(true,0);const before=RiftAudio.voices;RiftAudio.play('geomancer_terrain',0);const added=RiftAudio.voices-before;await RiftAudio.setActive(false);return {added,left:RiftAudio.voices};})).toEqual({added:2,left:0});
});
