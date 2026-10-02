const {test,expect}=require('@playwright/test');
test('all regional entrance markers follow frozen coordinates',async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(async({run,levels})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=true;RiftDisplay.cameraSmooth=false;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText;window.entryLabels=[];
  ctx.fillText=function(text,...args){if(text==='ENTRY')entryLabels.push(args.slice(0,2));return fill.call(this,text,...args);};
  run.status='fighting';run.paused=true;run.clock=2;run.events=[];run.enemies=[];run.room=0;
  window.entryRun=run;window.entryLevels=levels;document.querySelector('#rift-overlay').hidden=true;
 },data);
 const positions=[];
 for(let region=0;region<10;region++){
  const entry=await page.evaluate(async region=>{entryRun.level=structuredClone(entryLevels[region*10]);const e=entryRun.level.rooms[0].entrance;entryRun.player.x=e.x;entryRun.player.y=e.y;entryLabels=[];await RiftRenderer.prepareRun(entryRun);RiftRenderer.snapshot(entryRun,true);RiftMinimap.update(entryRun);return e;},region);
  await expect.poll(()=>page.evaluate(()=>entryLabels.length>0)).toBe(true);positions.push([entry.x,entry.y]);
  await expect(page.locator('#rift-minimap [data-kind=entry]')).toHaveAttribute('width','4');
 }
 expect(new Set(positions.map(JSON.stringify)).size).toBe(10);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('region-entry.png')});
 await page.evaluate(async ()=>{delete entryRun.level.rooms[0].entrance;entryLabels=[];await RiftRenderer.prepareRun(entryRun);RiftRenderer.snapshot(entryRun,true);RiftMinimap.update(entryRun);});await page.waitForTimeout(120);expect(await page.evaluate(()=>entryLabels)).toEqual([]);await expect(page.locator('#rift-minimap [data-kind=entry]')).toHaveCount(0);expect(errors).toEqual([]);
});
