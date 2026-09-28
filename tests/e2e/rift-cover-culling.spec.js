const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path');
test('cover culling preserves partial edge sprites as the camera moves',async({page})=>{
 await page.route('**/static/rift_renderer.js*',route=>route.fulfill({contentType:'application/javascript',body:fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift_renderer.js'),'utf8')}));
 await page.addInitScript(()=>{
  const sources=new WeakMap(),draw=CanvasRenderingContext2D.prototype.drawImage;
  CanvasRenderingContext2D.prototype.drawImage=function(image,...args){
   const source=sources.get(image)||(/rift_terrain_cover/.test(image.src||'')?'terrain':/rift_props/.test(image.src||'')?'prop':null);
   if(source){
    if(this.canvas.id==='rift-canvas'){
     if(window.coverOpacity){coverOpacity.push(this.globalAlpha);if(coverOpacity.length>20)coverOpacity.shift();}
     if(window.coverDraws&&args.length===8)coverDraws.add(JSON.stringify([source,args[4],args[7]]));
    }else sources.set(this.canvas,source);
   }
   return draw.call(this,image,...args);
  };
 });
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.evaluate(run=>{
  run.id='cover-cull';run.status='fighting';run.paused=true;run.enemies=[];run.events=[];run.room_objective=null;run.room=0;run.level.rooms[0].camera_lead=350;
  const boxes=[-300,-20,300,940,1200,1580,1900].map(x=>({x,y:390,w:40,h:40}));
  Object.assign(run.level.rooms[0],{obstacles:boxes,high_cover:boxes,cover:boxes.map((box,i)=>({...box,id:'cover'+i,material:'wood',hp:50,max_hp:100}))});
  window.coverRun=run;window.coverDraws=new Set();

 },run);
 for(const camera of [0,640]){
  await page.evaluate(camera=>{window.coverRun.player.x=camera===0?160:1400;window.RiftRenderer.snapshot(structuredClone(window.coverRun),true);window.coverDraws.clear();},camera);
  await expect.poll(()=>page.evaluate(()=>window.coverDraws.size)).toBeGreaterThan(0);
  const expected=[];
  for(const worldX of camera===0?[-20,300,940]:[940,1200,1580]){
   expected.push(JSON.stringify(['prop',worldX-camera-7,78]),JSON.stringify(['prop',worldX-camera-7,140]),JSON.stringify(['terrain',worldX-camera-6,114]));
  }
  expect(await page.evaluate(()=>[...window.coverDraws].sort())).toEqual(expected.sort());
 }
});
