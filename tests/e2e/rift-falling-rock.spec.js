const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test(`falling rock shadow and overhead counterplay reduced=${reduced}`,async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(async({run,reduced})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=reduced;RiftDisplay.hazardLabels=true;RiftDisplay.cameraSmooth=false;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText,ellipse=ctx.ellipse;window.rockLabels=[];window.rockShadows=0;
  ctx.fillText=function(text,...args){if(/ROCK FALL|MOVE OUT|MOVE ·/.test(text))rockLabels.push(text);return fill.call(this,text,...args);};
  ctx.ellipse=function(...args){if(this.strokeStyle==='#f4d6aa'&&Math.abs(args[2]-70.4)<1e-8)rockShadows++;return ellipse.apply(this,args);};
  run.status='fighting';run.paused=true;run.clock=.6;run.events=[];run.player.x=480;run.enemies=[];
  run.level.rooms[run.room].hazards=[{x:540,y:410,w:160,h:45,kind:'falling_rock',period:7,offset:0,duration:.18,jumpable:false}];
  window.rockRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{run:data.run,reduced});
 await expect.poll(()=>page.evaluate(()=>rockLabels.includes('ROCK FALL IN 0.6s · HITS ENEMIES')&&rockLabels.includes('MOVE OUT · JUMP WON’T HELP')&&rockShadows>0)).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('rock-warning.png')});
 await page.evaluate(()=>{rockRun.clock=1.22;rockLabels=[];RiftRenderer.snapshot(rockRun,true);});
 await expect.poll(()=>page.evaluate(()=>rockLabels.includes('MOVE · 0.2s · HITS ENEMIES'))).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('rock-impact.png')});
 await page.evaluate(()=>{rockRun.clock=2;rockLabels=[];rockShadows=0;RiftRenderer.snapshot(rockRun,true);});await page.waitForTimeout(120);expect(await page.evaluate(()=>[rockLabels.length,rockShadows])).toEqual([0,0]);expect(errors).toEqual([]);
});

test('server rock strike defeats a monster and grants one normal drop',async({page})=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto('/abyss/rift?scenario=enemy-rock');await expect(page.locator('#rift-start')).toBeEnabled();
 const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const before=await read();expect(before.paused).toBe(true);expect(before.enemies[0].hp).toBe(1);
 const id=before.enemies[0].id;
 await page.locator('#rift-start').click();
 await expect.poll(async()=>(await read()).enemies.find(e=>e.id===id).hp).toBe(0);
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const after=await read();expect(after.stats.kills).toBe(1);expect(after.drops.filter(d=>d.id===id)).toHaveLength(1);
 expect(after.stats.damage_dealt).toBe(0);expect(after.stats.combo_score).toBe(0);
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();
 const restored=await read();expect(restored.stats.kills).toBe(1);expect(restored.drops.filter(d=>d.id===id)).toHaveLength(1);
 expect(errors).toEqual([]);
});
