const {test,expect}=require('@playwright/test');
async function watchParticles(page){
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.evaluate(run=>{
  run.level.region=1;run.paused=true;window.RiftRenderer.snapshot(run,true);
  window.particleFrames=[];let count=0,started=false;
  const ctx=document.getElementById('rift-canvas').getContext('2d'),fill=ctx.fillRect.bind(ctx);
  ctx.fillRect=(...args)=>{
   if(ctx.fillStyle==='#091914'&&args[2]===960&&args[3]===540){if(started)window.particleFrames.push(count);started=true;count=0;}
   if(ctx.fillStyle==='#d6aaa0'&&args[2]===2&&args[3]===2)count++;
   return fill(...args);
  };
 },run);
}
async function expectBudget(page,budget){
 await page.evaluate(()=>{window.particleFrames=[];});
 await expect.poll(()=>page.evaluate(()=>window.particleFrames.length)).toBeGreaterThan(3);
 expect(await page.evaluate(()=>window.particleFrames.slice(-3))).toEqual([budget,budget,budget]);
}
test('particle budget changes actual drawing and persists',async({page})=>{
 await page.emulateMedia({reducedMotion:'no-preference'});
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.locator('.rift-settings > summary').click();await watchParticles(page);
 for(const [value,count] of [['1',22],['0.5',11],['0',0],['0.5',11]]){
  await page.locator('#rift-particle-intensity').selectOption(value);await expectBudget(page,count);
 }
 await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();
 await expect(page.locator('#rift-particle-intensity')).toHaveValue('0.5');await watchParticles(page);await expectBudget(page,11);
 await page.locator('.rift-settings > summary').click();await page.locator('#rift-background-particles').uncheck();await expectBudget(page,0);
 await page.locator('#rift-background-particles').check();await expectBudget(page,11);
 await page.locator('#rift-reduced').check();await expectBudget(page,0);
});
