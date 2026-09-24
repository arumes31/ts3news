const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path');
test('health labels cull off-screen actors but preserve wide edge names',async({page})=>{
 await page.route('**/static/rift_renderer.js*',route=>route.fulfill({contentType:'application/javascript',body:fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift_renderer.js'),'utf8')}));
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 const result=await page.evaluate(run=>{
  run.id='health-cull';run.paused=true;run.enemies=[];run.events=[];run.player.x=160;run.room=0;run.level.rooms[0].camera_lead=350;
  const renderer=window.RiftRenderer;renderer.snapshot(run,true);
  run.room_objective={captives:[{cage_id:'cage-probe',name:'Probe Wide Enemy Display Name',freed:false}]};
  Object.assign(window.RiftDisplay,{healthBars:true,enemyNames:'all',textScale:1.25,cleanScreenshot:false});
  const context=document.querySelector('#rift-canvas').getContext('2d'),rect=context.fillRect,text=context.fillText,bars=[],names=[];
  context.fillRect=function(x,y,w,h){if(((w===48||w===50)&&h===5)||(w===60&&h===6))bars.push(x);return rect.call(this,x,y,w,h);};
  context.fillText=function(label,x,...args){if(String(label).startsWith('Probe'))names.push(x);return text.call(this,label,x,...args);};
  try{
   for(const kind of ['goblin','boss','generator','totem','cage','lantern'])for(const x of [-400,-60,-10,400,970,1400])renderer.renderActor({id:kind==='cage'?'cage-probe':kind==='lantern'?'ward-lantern':kind+x,name:'Probe Wide Enemy Display Name',art_key:'monster:probe',kind,x,y:410,hp:50,max_hp:100,facing:1,pose:'idle',jump:0,phase:1},performance.now());
  }finally{run.room_objective=null;context.fillRect=rect;context.fillText=text;}
  return {bars:bars.sort((a,b)=>a-b),names:names.sort((a,b)=>a-b)};
 },run);
 expect(result.bars).toHaveLength(18);
 expect(result.names).toEqual([-60,-10,400,970].flatMap(x=>Array(5).fill(x)));
});
