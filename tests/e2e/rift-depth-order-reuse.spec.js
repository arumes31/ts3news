const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path');
test('depth order is reused until crossing or stable tie order changes',async({page})=>{
 await page.addInitScript(()=>{const request=window.requestAnimationFrame;window.requestAnimationFrame=function(callback){if(callback.name==='render'){window.renderProbe=callback;return 123456789;}return request.call(this,callback);};});
 await page.route('**/static/rift_renderer.js*',route=>route.fulfill({contentType:'application/javascript',body:fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift_renderer.js'),'utf8')}));
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();await page.waitForFunction(()=>window.renderProbe);
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 const result=await page.evaluate(async run=>{
  run.id='depth-order';run.paused=true;run.events=[];run.drops=[];run.room_objective=null;run.room=0;run.player.y=400;
  Object.assign(run.level.rooms[0],{obstacles:[],cover:[],high_cover:[]});
  run.enemies=[350,450].map((y,i)=>({id:'probe'+i,name:'Probe'+i,kind:'goblin',art_key:'monster:probe',x:400,y,hp:50,max_hp:100,facing:1,pose:'idle'}));
  await RiftRenderer.prepareRun(run);
  const empty=structuredClone(run);empty.enemies=[];RiftRenderer.snapshot(empty,true);window.renderProbe(performance.now()+500);
  Object.assign(window.RiftDisplay,{enemyNames:'all',fps:60});
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText,sort=Array.prototype.sort;
  let sorts=0,names=[],now=performance.now()+1000;
  ctx.fillText=function(text,...args){if(String(text).startsWith('Probe'))names.push(text);return fill.call(this,text,...args);};
  Array.prototype.sort=function(...args){if(this.length===3&&this.every(v=>typeof v==='number'||v.id==='player'||v.id?.startsWith('probe')))sorts++;return sort.apply(this,args);};
  const sample=()=>{window.RiftRenderer.snapshot(structuredClone(run),true);names=[];window.renderProbe(now+=100);return {sorts,names:[...names]};};
  try{
   const first=sample();for(let i=0;i<20;i++){run.enemies[0].y+=1;sample();}
   const stable={sorts,names:[...names]};
   run.enemies[0].y=470;const crossed=sample();
   run.enemies[0].y=450;const tied=sample();
   return {first,stable,crossed,tied};
  }finally{Array.prototype.sort=sort;ctx.fillText=fill;}
 },run);
 expect(result.first).toEqual({sorts:1,names:['Probe0','Probe1']});
 expect(result.stable).toEqual({sorts:1,names:['Probe0','Probe1']});
 expect(result.crossed).toEqual({sorts:2,names:['Probe1','Probe0']});
 expect(result.tied).toEqual({sorts:3,names:['Probe0','Probe1']});
});
