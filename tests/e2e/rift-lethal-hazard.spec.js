const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test('lethal hazard warning is conditional and static, reduced='+reduced,async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();expect(data.hazard_hit_damage).toBeGreaterThanOrEqual(0);
 await page.evaluate(async({data,reduced})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=reduced;RiftDisplay.hazardLabels=true;RiftDisplay.cameraSmooth=false;
  const run=RiftProtocol.validate(data,'GET').run;
  run.status='fighting';run.paused=true;run.clock=.6;run.events=[];run.enemies=[];run.player.x=480;run.player.hp=10;run.hazard_hit_damage=10;
  run.level.rooms[run.room].hazards=[{x:540,y:410,w:160,h:45,kind:'falling_rock',period:7,offset:0,duration:.18,jumpable:false}];
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText;
  window.lethalLabels=[];ctx.fillText=function(text,...args){if(text==='! LETHAL IF HIT')lethalLabels.push(text);return fill.call(this,text,...args);};
  window.lethalRun=run;RiftHUD.update(run,false);document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{data,reduced});
 await expect.poll(()=>page.evaluate(()=>lethalLabels.length)).toBeGreaterThan(0);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('lethal-warning.png')});
 for(const kind of ['fire','ice','void','tracking_lightning','moving_poison','rotating_blade','sweeping_flame']){
  await page.evaluate(kind=>{lethalRun.level.rooms[lethalRun.room].hazards[0].kind=kind;lethalLabels=[];RiftRenderer.snapshot(lethalRun,true);},kind);
  await expect.poll(()=>page.evaluate(()=>lethalLabels.length)).toBeGreaterThan(0);
 }
 for(const change of ['survivable','disabled','recovery','cleared','dead','hidden','clean']){
  await page.evaluate(change=>{
   const run=lethalRun,h=run.level.rooms[run.room].hazards[0];run.player.hp=10;run.hazard_hit_damage=10;run.clock=.6;run.status='fighting';h.disabled=false;RiftDisplay.hazardLabels=true;RiftDisplay.cleanScreenshot=false;
   if(change==='survivable')run.hazard_hit_damage=9.999;
   if(change==='disabled')h.disabled=true;
   if(change==='recovery')run.clock=2;
   if(change==='cleared')run.status='cleared';
   if(change==='dead')run.player.hp=0;
   if(change==='hidden')RiftDisplay.hazardLabels=false;
   if(change==='clean')RiftDisplay.cleanScreenshot=true;
   lethalLabels=[];RiftRenderer.snapshot(run,true);
  },change);await page.waitForTimeout(100);expect(await page.evaluate(()=>lethalLabels)).toEqual([]);
 }
 const protocol=await page.evaluate(data=>{
  const accepted=[];for(const value of [-1,'10',Infinity,NaN,1e9]){try{RiftProtocol.validate({...data,hazard_hit_damage:value},'GET');accepted.push(String(value));}catch(_){}}
  const copy=structuredClone(data);copy.run.hazard_hit_damage=999;delete copy.hazard_hit_damage;
  return {accepted,stale:RiftProtocol.validate(copy,'GET').run.hazard_hit_damage};
 },data);
 expect(protocol.accepted).toEqual([]);expect(protocol.stale).toBeUndefined();expect(errors).toEqual([]);
});
