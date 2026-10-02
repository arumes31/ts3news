const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test(`boss channel cue and interruption reduced=${reduced}`,async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(async({run,bestiary,reduced})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=reduced;RiftDisplay.enemyIntent=true;RiftDisplay.cameraSmooth=false;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText,ellipse=ctx.ellipse;window.channelLabels=[];window.channelShelterArcs=[];window.shelterLabelBounds=[];
  ctx.ellipse=function(...args){if(args[2]===200&&args[3]===90)channelShelterArcs.push(args);return ellipse.apply(this,args);};
  ctx.fillText=function(text,...args){if(/OUTER BAND|ENEMIES ACTIVE/.test(text))shelterLabelBounds.push({text,x:args[0],y:args[1],width:this.measureText(text).width,font:this.font});if(/CHANNEL|TIME PULSE|Channeling|OUTER BAND|ENEMIES ACTIVE/.test(text))channelLabels.push(text);return fill.call(this,text,...args);};
  run.status='fighting';run.paused=true;run.events=[];run.player.x=440;
  run.enemies=[{...bestiary.find(e=>e.lane_slams),id:'chronos',x:650,y:410,hp:1000,max_hp:1000,attacks:3,windup:2,target_x:440,target_y:410,attack_name:'Time Pulse',pose:'windup'}];
  window.channelRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{run:data.run,bestiary:data.bestiary,reduced});
 await expect.poll(()=>page.evaluate(()=>channelLabels.includes('CHANNEL 2.0s · HIT BOSS TO INTERRUPT')&&channelLabels.includes('TIME PULSE · JUMP OR MOVE')&&channelLabels.includes('INTENT · Channeling pulse')&&channelLabels.includes('OUTER BAND · HAZARD SHELTER')&&channelLabels.includes('ENEMIES ACTIVE · AVOID OTHER WARNINGS')&&channelShelterArcs.length>0)).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('channel-warning.png')});
 await page.evaluate(()=>{Object.assign(channelRun.enemies[0],{windup:0,attack_name:'',attacks:4,pose:'hit'});channelRun.counter++;channelRun.events=[{id:channelRun.counter,kind:'boss_channel_interrupt',x:650,y:410,value:0}];channelLabels=[];RiftRenderer.snapshot(channelRun,true);});
 await expect.poll(()=>page.evaluate(()=>channelLabels.includes('CHANNEL INTERRUPTED · WEAK POINT'))).toBe(true);
 expect(await page.evaluate(()=>channelLabels.some(t=>t.startsWith('CHANNEL 2.0')||t.startsWith('TIME PULSE')||t.startsWith('OUTER BAND')))).toBe(false);
 await page.evaluate(()=>{channelRun.events=[];channelRun.enemies[0].hp=0;channelRun.skill_timers['hazard-channel-chronos']=.45;channelLabels=[];channelShelterArcs=[];RiftRenderer.snapshot(channelRun,true);});
 await expect.poll(()=>page.evaluate(()=>channelLabels.includes('OUTER BAND · HAZARD SHELTER')&&channelShelterArcs.length>0)).toBe(true);
 await page.setViewportSize({width:390,height:900});
 await page.evaluate(()=>{shelterLabelBounds=[];RiftRenderer.snapshot(channelRun,true);});
 await expect.poll(()=>page.evaluate(()=>shelterLabelBounds.length)).toBeGreaterThanOrEqual(2);
 const mobile=await page.evaluate(()=>({scale:document.querySelector('#rift-canvas').clientWidth/960,labels:shelterLabelBounds.slice(-2)}));
 for(const label of mobile.labels){expect(parseFloat(label.font.match(/[\d.]+px/)[0])*mobile.scale).toBeGreaterThanOrEqual(8.9);expect(label.x-label.width/2).toBeGreaterThanOrEqual(0);expect(label.x+label.width/2).toBeLessThanOrEqual(960);expect(label.y).toBeLessThanOrEqual(528);}
 expect((mobile.labels[1].y-mobile.labels[0].y)*mobile.scale).toBeGreaterThanOrEqual(13.9);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('channel-impact-mobile.png')});
 await page.evaluate(()=>{channelRun.skill_timers['hazard-channel-chronos']=0;channelLabels=[];channelShelterArcs=[];RiftRenderer.snapshot(channelRun,true);});await page.waitForTimeout(120);expect(await page.evaluate(()=>channelShelterArcs.length)).toBe(0);
 await page.evaluate(()=>{channelRun.skill_timers['hazard-channel-chronos']=.45;});
 await page.evaluate(()=>{RiftDisplay.cleanScreenshot=true;channelLabels=[];channelShelterArcs=[];RiftRenderer.snapshot(channelRun,true);});await page.waitForTimeout(120);expect(await page.evaluate(()=>[channelLabels.length,channelShelterArcs.length])).toEqual([0,0]);expect(errors).toEqual([]);
});
