const {test,expect}=require('@playwright/test');
for(const [kind,color] of Object.entries({fire:'#b86a40',ice:'#91cbd8',poison:'#92ad54',void:'#9170b7',radiant:'#d8ca85',rune:'#aa92c8'})) {
 test('bounded floor decal: '+kind,async({page})=>{
  await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
  const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
  await page.evaluate(({run,kind,color})=>{
   window.decalAlpha=[];const ctx=document.getElementById('rift-canvas').getContext('2d'),stroke=ctx.stroke.bind(ctx);
   ctx.stroke=(...args)=>{if(ctx.strokeStyle===color)window.decalAlpha.push(ctx.globalAlpha);return stroke(...args);};
   run.paused=false;run.counter++;run.events=[{id:run.counter,kind,x:run.player.x,y:run.player.y-30,value:12}];window.RiftRenderer.snapshot(run,false);
  },{run,kind,color});
  await expect.poll(()=>page.evaluate(()=>window.decalAlpha.length)).toBeGreaterThan(0);
  const initial=await page.evaluate(()=>window.decalAlpha[0]);await page.waitForTimeout(500);
  expect(await page.evaluate(()=>window.decalAlpha.at(-1))).toBeLessThan(initial);
  await page.waitForTimeout(1700);const count=await page.evaluate(()=>window.decalAlpha.length);await page.waitForTimeout(100);
  expect(await page.evaluate(()=>window.decalAlpha.length)).toBe(count);
 });
}

for(const mode of ['cast','replay','reduced','still'])test('decal suppression: '+mode,async({page})=>{
 await page.emulateMedia({reducedMotion:mode==='reduced'?'reduce':'no-preference'});
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.evaluate(({run,mode})=>{
  window.decalCount=0;const ctx=document.getElementById('rift-canvas').getContext('2d'),stroke=ctx.stroke.bind(ctx);
  ctx.stroke=(...args)=>{if(ctx.strokeStyle==='#b86a40')window.decalCount++;return stroke(...args);};
  if(mode==='still')window.RiftDisplay.motionIntensity=0;
  run.paused=false;run.counter++;run.events=[{id:run.counter,kind:'fire',x:run.player.x,y:run.player.y-30,value:mode==='cast'?0:12}];window.RiftRenderer.snapshot(run,mode==='replay');
 },{run,mode});
 await page.waitForTimeout(250);expect(await page.evaluate(()=>window.decalCount)).toBe(0);
});
