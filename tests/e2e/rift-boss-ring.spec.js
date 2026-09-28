const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test(`void ring marks exact annulus and open escape gap reduced=${reduced}`,async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json(),boss=data.bestiary.find(e=>e.ring_attack);expect(boss).toBeTruthy();
 await page.evaluate(async({run,boss,reduced})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=reduced;RiftDisplay.enemyIntent=true;RiftDisplay.cameraSmooth=false;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillText,ellipse=ctx.ellipse;
  window.ringLabels=[];window.ringArcs=[];window.oldSlam=[];window.ringTextBoxes={};
  ctx.fillText=function(text,...args){if(/VOID RING|OPEN GAP|Preparing ring/.test(text))ringLabels.push(text);if(/OPEN GAP|Preparing ring/.test(text)){const m=this.measureText(text);ringTextBoxes[text]={left:args[0]-m.width/2-5,right:args[0]+m.width/2+5,top:args[1]-(m.actualBoundingBoxAscent||10)-3,bottom:args[1]+(m.actualBoundingBoxDescent||2)+3};}return fill.call(this,text,...args);};
  ctx.ellipse=function(...args){if(args[2]===200||args[2]===100)ringArcs.push(args);if(args[2]===125&&args[3]===62)oldSlam.push(args);return ellipse.apply(this,args);};
  run.status='fighting';run.paused=true;run.events=[];run.player.x=440;
  run.enemies=[{...boss,id:'ring-boss',x:650,y:410,hp:1000,max_hp:1000,attacks:2,windup:2,target_x:650,target_y:410,ring_gap:0,attack_name:'Void Ring',pose:'windup'}];
  window.ringRun=run;document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{run:data.run,boss,reduced});
 for(const gap of [0,1]){
  await page.evaluate(gap=>{const e=ringRun.enemies[0];e.ring_gap=gap;e.y=e.target_y=gap?350:410;ringLabels=[];ringArcs=[];oldSlam=[];RiftRenderer.snapshot(ringRun,true);},gap);
  await expect.poll(()=>page.evaluate(()=>ringLabels.includes('VOID RING · 2.0s · CENTER, GAP OR OUTSIDE')&&ringLabels.includes('OPEN GAP · RING SAFE')&&ringLabels.includes('INTENT · Preparing ring')&&ringArcs.length>=2)).toBe(true);
  const arcs=await page.evaluate(()=>ringArcs.slice(-2)),start=(gap?Math.PI/2:-Math.PI/2)+Math.PI/4;
  expect(arcs[0][2]).toBe(200);expect(arcs[0][3]).toBe(90);expect(arcs[0][5]).toBeCloseTo(start,8);expect(arcs[0][6]-arcs[0][5]).toBeCloseTo(1.5*Math.PI,8);
  expect(arcs[1][2]).toBe(100);expect(arcs[1][3]).toBe(45);expect(arcs[1][7]).toBe(true);expect(await page.evaluate(()=>oldSlam.length)).toBe(0);
  const boxes=await page.evaluate(()=>[ringTextBoxes['OPEN GAP · RING SAFE'],ringTextBoxes['INTENT · Preparing ring']]);
  expect(boxes[0].right<=boxes[1].left||boxes[0].left>=boxes[1].right||boxes[0].bottom<=boxes[1].top||boxes[0].top>=boxes[1].bottom).toBe(true);
  await page.locator('#rift-canvas').screenshot({path:info.outputPath('ring-gap-'+gap+'.png')});
 }
 await page.evaluate(()=>{Object.assign(ringRun.enemies[0],{windup:0,attack_name:'',pose:'idle'});ringLabels=[];ringArcs=[];RiftRenderer.snapshot(ringRun,true);});await page.waitForTimeout(120);expect(await page.evaluate(()=>[ringLabels.length,ringArcs.length])).toEqual([0,0]);
 await page.evaluate(()=>{Object.assign(ringRun.enemies[0],{windup:2,attack_name:'Void Ring'});RiftDisplay.cleanScreenshot=true;ringLabels=[];ringArcs=[];RiftRenderer.snapshot(ringRun,true);});await page.waitForTimeout(120);expect(await page.evaluate(()=>[ringLabels.length,ringArcs.length])).toEqual([0,0]);expect(errors).toEqual([]);
});
