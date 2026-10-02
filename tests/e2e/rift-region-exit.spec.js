const {test,expect}=require('@playwright/test');
test('all regional exit markers follow frozen coordinates',async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(async({run,levels})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=true;RiftDisplay.cameraSmooth=false;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText;window.exitLabels=[];
  ctx.fillText=function(text,...args){if(String(text).startsWith('EXIT'))exitLabels.push(args.slice(0,2));return fill.call(this,text,...args);};
  run.status='fighting';run.paused=true;run.clock=2;run.events=[];run.enemies=[];run.room=0;
  window.exitRun=run;window.exitLevels=levels;document.querySelector('#rift-overlay').hidden=true;
 },data);
 const positions=[];
 for(let region=0;region<10;region++){
  const exit=await page.evaluate(async region=>{exitRun.level=structuredClone(exitLevels[region*10]);const e=exitRun.level.rooms[0].exit;exitRun.player.x=e.x;exitRun.player.y=e.y;exitLabels=[];await RiftRenderer.prepareRun(exitRun);RiftRenderer.snapshot(exitRun,true);RiftMinimap.update(exitRun);return e;},region);
  await expect.poll(()=>page.evaluate(()=>exitLabels.length>0)).toBe(true);positions.push([exit.x,exit.y]);
  await expect(page.locator('#rift-minimap [data-kind=exit]')).toHaveAttribute('x',String(Math.round((exit.x-10)*2)/10));
  await expect(page.locator('#rift-minimap [data-kind=exit]')).toHaveAttribute('width','4');
 }
 expect(new Set(positions.map(JSON.stringify)).size).toBe(10);
 await page.evaluate(async ()=>{exitRun.status='cleared';exitLabels=[];await RiftRenderer.prepareRun(exitRun);RiftRenderer.snapshot(exitRun,true);});
 await expect.poll(()=>page.evaluate(()=>exitLabels.length>0)).toBe(true);
 await page.waitForTimeout(150);await page.locator('#rift-canvas').screenshot({path:info.outputPath('region-exit.png')});
 await page.evaluate(async ()=>{delete exitRun.level.rooms[0].exit;exitLabels=[];await RiftRenderer.prepareRun(exitRun);RiftRenderer.snapshot(exitRun,true);RiftMinimap.update(exitRun);});await page.waitForTimeout(120);expect(await page.evaluate(()=>exitLabels)).toEqual([]);await expect(page.locator('#rift-minimap [data-kind=exit]')).toHaveCount(0);expect(errors).toEqual([]);
});
