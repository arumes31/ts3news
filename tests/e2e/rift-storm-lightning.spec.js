const {test,expect}=require('@playwright/test');
for(const mode of ['normal','reduced','still','flash-off','other-region'])test('distant storm lightning: '+mode,async({page})=>{
 await page.emulateMedia({reducedMotion:mode==='reduced'?'reduce':'no-preference'});
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.evaluate(async ({run,mode})=>{
  window.lightning=[];const ctx=document.getElementById('rift-canvas').getContext('2d'),stroke=ctx.stroke.bind(ctx);
  ctx.stroke=(...args)=>{if(ctx.strokeStyle==='#b8c9ec')window.lightning.push(ctx.globalAlpha);return stroke(...args);};
  if(mode==='still')window.RiftDisplay.motionIntensity=0;if(mode==='flash-off')window.RiftDisplay.flashIntensity=0;
  run.level.region=mode==='other-region'?0:3;run.paused=false;run.events=[];await window.RiftRenderer.prepareRun(run);window.RiftRenderer.snapshot(run,false);
 },{run,mode});
 if(mode==='normal'){
  await expect.poll(()=>page.evaluate(()=>window.lightning.length),{timeout:12000}).toBeGreaterThan(0);
  expect(await page.evaluate(()=>window.lightning.every(a=>a>=0&&a<=.28))).toBe(true);
  await page.waitForTimeout(400);const count=await page.evaluate(()=>window.lightning.length);await page.waitForTimeout(500);
  expect(await page.evaluate(()=>window.lightning.length)).toBe(count);
 }else{await page.waitForTimeout(9000);expect(await page.evaluate(()=>window.lightning.length)).toBe(0);}
});
