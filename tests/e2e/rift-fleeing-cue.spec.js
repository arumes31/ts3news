const {test,expect}=require('@playwright/test');
for(const mode of ['normal','reduced','still','paused','clean','idle','escaped'])test('treasure panic cue: '+mode,async({page},testInfo)=>{
 await page.emulateMedia({reducedMotion:mode==='reduced'?'reduce':'no-preference'});
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.evaluate(async({run,mode})=>{
  await window.RiftRenderer.ready;
  document.getElementById('rift-overlay').hidden=true;
  window.panicLabels=[];
  const ctx=document.getElementById('rift-canvas').getContext('2d'),fillText=ctx.fillText.bind(ctx);
  ctx.fillText=(text,x,y,...rest)=>{if(text==='FLEEING!')window.panicLabels.push({x,y});return fillText(text,x,y,...rest);};
  window.RiftDisplay.motionIntensity=mode==='still'?0:1;
  window.RiftDisplay.cleanScreenshot=mode==='clean';
  run.paused=mode==='paused';run.status='fighting';run.events=[];
  run.enemies=[{id:'panic-runner',kind:'treasure',x:run.player.x+100,y:410,hp:mode==='escaped'?0:100,max_hp:100,facing:1,pose:mode==='escaped'?'escape':'run',fleeing:mode!=='idle',jump:0}];
  window.RiftRenderer.snapshot(run,false);
 },{run,mode});
 if(['clean','idle','escaped'].includes(mode)){await page.waitForTimeout(250);expect(await page.evaluate(()=>window.panicLabels.length)).toBe(0);return;}
 await expect.poll(()=>page.evaluate(()=>window.panicLabels.length)).toBeGreaterThan(2);
 await page.waitForTimeout(250);
 const positions=await page.evaluate(()=>[...new Set(window.panicLabels.map(p=>p.y.toFixed(3)))]);
 if(mode==='normal'){expect(positions.length).toBeGreaterThan(1);await page.locator('#rift-canvas').screenshot({path:testInfo.outputPath('fleeing-cue.png')});}else expect(positions).toHaveLength(1);
});
