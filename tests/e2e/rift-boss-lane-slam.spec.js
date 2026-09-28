const {test,expect}=require('@playwright/test');
for(const reduced of [false,true]) test('lane slam reserves clear bands, reduced motion '+reduced,async({page},testInfo)=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();const boss=data.bestiary.find(e=>e.lane_slams);expect(boss).toBeTruthy();
 await page.evaluate(async({run,boss,reduced})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=reduced;RiftDisplay.cameraSmooth=false;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText,stroke=ctx.strokeRect,ellipse=ctx.ellipse;
  window.laneLabels=[];window.laneRects=[];window.laneEllipses=[];
  ctx.fillText=function(text,x,y,...rest){if(/LANE SLAM|CLEAR OF THIS SLAM|HAZARDS PAUSED/.test(text))laneLabels.push({text,y});return fill.call(this,text,x,y,...rest);};
  ctx.strokeRect=function(x,y,w,h){if(w===1530&&Math.abs(h-175/3)<1e-7)laneRects.push({y,h,dash:this.getLineDash(),color:this.strokeStyle});return stroke.call(this,x,y,w,h);};
  ctx.ellipse=function(...args){if(args[2]===125&&args[3]===62)laneEllipses.push(args);return ellipse.apply(this,args);};
  run.status='fighting';run.paused=true;run.events=[];run.player.x=500;
  run.enemies=[{...boss,id:'lane',x:800,y:410,hp:1000,max_hp:1000,attacks:0,slam_lane:1,attack_name:'Lane Slam',windup:1.6,pose:'windup'}];
  window.laneRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{run:data.run,boss,reduced});
 for(const lane of [0,1,2]){
  await page.evaluate(lane=>{laneRun.enemies[0].slam_lane=lane;laneLabels=[];laneRects=[];laneEllipses=[];RiftRenderer.snapshot(laneRun,true);},lane);
  await expect.poll(()=>page.evaluate(()=>laneLabels.some(v=>v.text==='LANE SLAM · 1.6s · MOVE OR JUMP'))).toBe(true);
  const labels=await page.evaluate(()=>laneLabels.slice(-3));expect(labels.filter(v=>v.text==='HAZARDS PAUSED · ENEMIES ACTIVE')).toHaveLength(2);
  expect(labels.find(v=>v.text.startsWith('LANE SLAM')).y).toBeCloseTo(315+(lane+.5)*175/3,6);
  expect(await page.evaluate(lane=>laneRects.some(v=>Math.abs(v.y-(315+lane*175/3))<1e-7&&Math.abs(v.h-175/3)<1e-7&&v.dash.length>0),lane)).toBe(true);
  expect(await page.evaluate(()=>laneEllipses)).toEqual([]);
 }
 await page.locator('#rift-canvas').screenshot({path:testInfo.outputPath('lane-slam.png')});
 await page.evaluate(()=>{laneRun.enemies[0].slam_lane=1;laneRun.skill_timers={'hazard-slam-lane-0':.4};laneLabels=[];RiftRenderer.snapshot(laneRun,true);});
 await expect.poll(()=>page.evaluate(()=>laneLabels.slice(-3).map(v=>v.text))).toEqual(['CLEAR OF THIS SLAM','LANE SLAM · 1.6s · MOVE OR JUMP','HAZARDS PAUSED · ENEMIES ACTIVE']);
 await page.evaluate(()=>{laneRun.enemies.push({...laneRun.enemies[0],id:'ring-conflict',lane_slams:false,ring_attack:true,attack_name:'Void Ring',target_x:650,target_y:410,ring_gap:0});laneLabels=[];RiftRenderer.snapshot(laneRun,true);});
 await expect.poll(()=>page.evaluate(()=>laneLabels.slice(-3).map(v=>v.text))).toEqual(['CLEAR OF THIS SLAM','LANE SLAM · 1.6s · MOVE OR JUMP','CLEAR OF THIS SLAM']);
 await page.evaluate(()=>{RiftDisplay.cleanScreenshot=true;laneLabels=[];laneRects=[];RiftRenderer.snapshot(laneRun,true);});await page.waitForTimeout(150);
 expect(await page.evaluate(()=>laneLabels)).toEqual([]);expect(await page.evaluate(()=>laneRects)).toEqual([]);
});
