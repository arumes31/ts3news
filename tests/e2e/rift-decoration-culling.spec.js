const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path');
test('offscreen steam and sparkles cull while camera-edge decorations survive',async({page})=>{
 await page.route('**/static/rift_renderer.js*',route=>route.fulfill({contentType:'application/javascript',body:fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift_renderer.js'),'utf8')}));
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.evaluate(run=>{
  run.id='decoration-cull';run.paused=true;run.enemies=[];run.events=[];run.room_objective=null;run.room=0;run.level.rooms[0].camera_lead=350;
  const positions=[-200,-10,400,950,1200,1590,1850];
  run.level.rooms[0].steam_vents=positions.map(x=>({x,y:330,w:40,h:50}));
  run.drops=positions.map((x,i)=>({id:i+1,x,y:410,gold:1}));
  Object.assign(window.RiftDisplay,{lootMotion:false,lootSparkle:true,motionIntensity:1});window.RiftRenderer.reduced=false;
  window.decorationRun=run;window.sparkleDraws=new Set();window.steamDraws=new Set();
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),draw=ctx.drawImage,ellipse=ctx.ellipse;
  ctx.drawImage=function(image,...args){if(image.src?.includes('rift_effects.png')&&args.length===8&&args[6]===34)window.sparkleDraws.add(args[4]);return draw.call(this,image,...args);};
  ctx.ellipse=function(x,y,...args){if(this.fillStyle==='#ced8c8')window.steamDraws.add(Math.floor((x+1000)/40));return ellipse.call(this,x,y,...args);};
 },run);
 for(const camera of [0,640]){
  await page.evaluate(camera=>{window.decorationRun.player.x=camera===0?160:1400;window.RiftRenderer.snapshot(structuredClone(window.decorationRun),true);window.sparkleDraws.clear();window.steamDraws.clear();},camera);
  await expect.poll(()=>page.evaluate(()=>window.sparkleDraws.size)).toBeGreaterThan(0);
  expect(await page.evaluate(()=>[...window.sparkleDraws].sort((a,b)=>a-b))).toEqual((camera===0?[-10,400,950]:[950,1200,1590]).map(x=>x-camera-17));
  expect(await page.evaluate(()=>window.steamDraws.size)).toBe(3);
 }
});
