const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test('offset bastion flanks across ten regions, reduced='+reduced,async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.emulateMedia({reducedMotion:reduced?'reduce':'no-preference'});
 await page.goto('/abyss/rift?scenario=visual&seed=bastion&level=3&room=2');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json(),saved=data.run.level.rooms[2].high_cover;
 expect(saved).toHaveLength(3);expect(data.run.level.rooms[2].name).toContain('Offset Bastion');
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();
 expect((await(await page.request.get('/api/abyss/rift')).json()).run.level.rooms[2].high_cover).toEqual(saved);
 const scenes=[];
 for(let region=0;region<10;region++){
  const level=data.levels[region*10+2],walls=level.rooms[2].high_cover;
  expect(level.tactic).toContain('upper or lower flank');expect(level.rooms[2].name).toContain('Offset Bastion');
  await page.evaluate(async({run,level,reduced})=>{
   await RiftRenderer.ready;RiftRenderer.reduced=reduced;RiftDisplay.cameraSmooth=false;
   run.level=level;run.room=2;run.room_objective=null;run.status='fighting';run.paused=true;run.clock=0;run.events=[];run.enemies=[];run.player.x=850;run.player.y=410;
   document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);RiftMinimap.update(run);
  },{run:structuredClone(data.run),level,reduced});
  const stones=page.locator('#rift-minimap [data-kind=stone]');await expect(stones).toHaveCount(3);
  expect(await stones.evaluateAll(nodes=>nodes.map(n=>Number(n.getAttribute('x'))))).toEqual(walls.map(w=>Math.round(w.x*2)/10));
  expect(await stones.evaluateAll(nodes=>nodes.map(n=>Number(n.getAttribute('width'))))).toEqual(walls.map(w=>Math.round(w.w*2)/10));
  await page.waitForTimeout(150);
  const image=await page.locator('#rift-canvas').screenshot({path:info.outputPath('bastion-'+level.id+'.png')});
  scenes.push({name:level.region_name,image:image.toString('base64')});
 }
 expect(errors).toEqual([]);
 await page.setViewportSize({width:1020,height:1600});
 await page.setContent('<body style="margin:20px;background:#101a19;color:white;font:16px sans-serif"><h1>Offset bastions — '+(reduced?'reduced motion':'normal motion')+'</h1><main style="display:grid;grid-template-columns:480px 480px;gap:12px">'+scenes.map(s=>'<section><div>'+s.name+'</div><img width="480" src="data:image/png;base64,'+s.image+'"></section>').join('')+'</main></body>');
 await page.screenshot({path:info.outputPath('bastion-regions.png'),fullPage:true});
});
