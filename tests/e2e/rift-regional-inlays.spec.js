const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test('regional interaction inlays reduced='+reduced,async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(async({run,levels,reduced})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=reduced;RiftDisplay.cameraSmooth=false;
  run.status='fighting';run.paused=true;run.clock=2;run.events=[];run.enemies=[];run.room=0;
  window.inlayRun=run;window.inlayLevels=levels;document.querySelector('#rift-overlay').hidden=true;
  const sheet=document.createElement('canvas');sheet.id='regional-inlay-sheet';sheet.width=600;sheet.height=500;document.body.append(sheet);
 },{...data,reduced});
 for(let region=0;region<10;region++){
  await page.evaluate(region=>{inlayRun.level=structuredClone(inlayLevels[region*10]);const e=inlayRun.level.rooms[0].entrance;inlayRun.player.x=e.x+100;inlayRun.player.y=e.y;RiftRenderer.snapshot(inlayRun,true);},region);
  await page.waitForTimeout(150);
  await page.evaluate(region=>{
   const canvas=document.querySelector('#rift-canvas'),sheet=document.querySelector('#regional-inlay-sheet'),ctx=sheet.getContext('2d'),e=inlayRun.level.rooms[0].entrance;
   const sx=canvas.width/960,sy=canvas.height/540;
   ctx.drawImage(canvas,(e.x-55)*sx,(e.y-20)*sy,110*sx,55*sy,(region%2)*300,Math.floor(region/2)*100,220,80);
   ctx.fillStyle='#fff';ctx.font='12px monospace';ctx.fillText(inlayRun.level.name,(region%2)*300,Math.floor(region/2)*100+95);
  },region);
 }
 await page.locator('#regional-inlay-sheet').screenshot({path:info.outputPath('regional-inlays.png')});
 expect(errors).toEqual([]);
});
