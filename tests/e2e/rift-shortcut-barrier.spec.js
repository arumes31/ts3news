const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test('shortcut labels and broken route reduced='+reduced,async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(async({run,levels,reduced})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=reduced;RiftDisplay.cameraSmooth=false;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText;window.shortcutLabels=[];
  ctx.fillText=function(text,...args){if(/SHORTCUT/.test(text))shortcutLabels.push(text);return fill.call(this,text,...args);};
  run.status='fighting';run.paused=true;run.clock=2;run.events=[];run.enemies=[];run.level=structuredClone(levels[6]);run.room=1;run.room_objective=null;
  const c=run.level.rooms[1].cover.find(c=>c.shortcut);run.player.x=c.x-50;run.player.y=410;run.player.facing=1;
  window.shortcutRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);RiftMinimap.update(run);
 },{...data,reduced});
 await expect.poll(()=>page.evaluate(()=>shortcutLabels.includes('BREAK FOR SHORTCUT'))).toBe(true);
 await expect(page.locator('#rift-minimap [data-kind=wood]')).toHaveCount(1);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('shortcut-blocked.png')});
 await page.evaluate(()=>{shortcutRun.level.rooms[1].cover[0].hp=0;shortcutLabels=[];RiftRenderer.snapshot(shortcutRun,true);RiftMinimap.update(shortcutRun);});
 await expect.poll(()=>page.evaluate(()=>shortcutLabels.includes('SHORTCUT OPEN'))).toBe(true);
 await expect(page.locator('#rift-minimap [data-kind=wood]')).toHaveCount(0);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('shortcut-open.png')});expect(errors).toEqual([]);
});
