const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test(`boss channel cue and interruption reduced=${reduced}`,async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(async({run,bestiary,reduced})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=reduced;RiftDisplay.enemyIntent=true;RiftDisplay.cameraSmooth=false;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText;window.channelLabels=[];
  ctx.fillText=function(text,...args){if(/CHANNEL|TIME PULSE|Channeling/.test(text))channelLabels.push(text);return fill.call(this,text,...args);};
  run.status='fighting';run.paused=true;run.events=[];run.player.x=440;
  run.enemies=[{...bestiary.find(e=>e.lane_slams),id:'chronos',x:650,y:410,hp:1000,max_hp:1000,attacks:3,windup:2,target_x:440,target_y:410,attack_name:'Time Pulse',pose:'windup'}];
  window.channelRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{run:data.run,bestiary:data.bestiary,reduced});
 await expect.poll(()=>page.evaluate(()=>channelLabels.includes('CHANNEL 2.0s · HIT BOSS TO INTERRUPT')&&channelLabels.includes('TIME PULSE · JUMP OR MOVE')&&channelLabels.includes('INTENT · Channeling pulse'))).toBe(true);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('channel-warning.png')});
 await page.evaluate(()=>{Object.assign(channelRun.enemies[0],{windup:0,attack_name:'',attacks:4,pose:'hit'});channelRun.counter++;channelRun.events=[{id:channelRun.counter,kind:'boss_channel_interrupt',x:650,y:410,value:0}];channelLabels=[];RiftRenderer.snapshot(channelRun,true);});
 await expect.poll(()=>page.evaluate(()=>channelLabels.includes('CHANNEL INTERRUPTED · WEAK POINT'))).toBe(true);
 expect(await page.evaluate(()=>channelLabels.some(t=>t.startsWith('CHANNEL 2.0')||t.startsWith('TIME PULSE')))).toBe(false);
 await page.evaluate(()=>{RiftDisplay.cleanScreenshot=true;channelLabels=[];RiftRenderer.snapshot(channelRun,true);});await page.waitForTimeout(120);expect(await page.evaluate(()=>channelLabels)).toEqual([]);expect(errors).toEqual([]);
});
