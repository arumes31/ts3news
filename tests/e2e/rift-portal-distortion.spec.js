const {test,expect}=require('@playwright/test');
test('cleared exits have bounded portal ripples with static accessibility modes',async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(({run,levels})=>{
  run.level=structuredClone(levels[0]);run.room=0;run.room_objective=null;run.enemies=[];run.events=[];run.paused=true;run.player.x=run.level.rooms[0].exit.x;run.player.y=run.level.rooms[0].exit.y;run.status='cleared';
  window.portalRun=run;window.portalCopies=[];window.portalRims=[];
  RiftDisplay.cameraSmooth=false;RiftDisplay.particles=false;RiftDisplay.shakeIntensity=0;RiftRenderer.reduced=false;
  const canvas=document.querySelector('#rift-canvas'),ctx=canvas.getContext('2d'),draw=ctx.drawImage,fill=ctx.fillRect,ellipse=ctx.ellipse;
  ctx.fillRect=function(...a){if(this.fillStyle==='#091914'&&a[2]===960&&a[3]===540){portalCopies=[];portalRims=[];}return fill.apply(this,a);};
  ctx.drawImage=function(img,...a){if(img===canvas)portalCopies.push(a);return draw.call(this,img,...a);};
  ctx.ellipse=function(...a){if(a[2]===32&&a[3]===48)portalRims.push(a.slice(0,4));return ellipse.apply(this,a);};
  document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },data);
 await expect.poll(()=>page.evaluate(()=>portalCopies.length)).toBeGreaterThan(0);
 const state=await page.evaluate(()=>({run:JSON.stringify(portalRun),copies:portalCopies}));
 expect(state.copies.length).toBeLessThanOrEqual(7);
 for(const [sx,sy,w,h,dx,dy,dw,dh] of state.copies){expect(sx).toBeGreaterThanOrEqual(0);expect(sy).toBeGreaterThanOrEqual(0);expect(sx+w).toBeLessThanOrEqual(960);expect(sy+h).toBeLessThanOrEqual(540);expect(Math.abs(dx-sx)).toBeLessThanOrEqual(2);expect(dy).toBe(sy);expect(dw).toBe(w);expect(dh).toBe(h);}
 await page.waitForTimeout(250);expect(await page.evaluate(()=>portalCopies)).toEqual(state.copies);expect(await page.evaluate(()=>JSON.stringify(portalRun))).toBe(state.run);
 const view=await page.evaluate(()=>{portalRun.player.x-=100;RiftRenderer.snapshot(portalRun,true);return RiftRenderer.frameCount;});await expect.poll(()=>page.evaluate(()=>RiftRenderer.frameCount)).toBeGreaterThan(view);
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('rift-portal.png')});
 await page.evaluate(()=>{portalRun.paused=false;RiftRenderer.snapshot(portalRun,true);});
 await expect.poll(()=>page.evaluate(()=>JSON.stringify(portalCopies))).not.toBe(JSON.stringify(state.copies));
 for(const setting of ['reduced','zero']){
  await page.evaluate(setting=>{RiftRenderer.reduced=setting==='reduced';RiftDisplay.motionIntensity=setting==='zero'?0:1;},setting);
  await expect.poll(()=>page.evaluate(()=>portalCopies.length)).toBe(0);expect(await page.evaluate(()=>portalRims.length)).toBeGreaterThan(0);
 }
 await page.evaluate(()=>{RiftDisplay.motionIntensity=1;portalRun.status='fighting';RiftRenderer.snapshot(portalRun,true);});
 await expect.poll(()=>page.evaluate(()=>portalRims.length)).toBe(0);expect(await page.evaluate(()=>portalCopies.length)).toBe(0);
 await page.evaluate(()=>{portalRun.status='cleared';portalRun.player.x=160;RiftRenderer.snapshot(portalRun,true);});
 await expect.poll(()=>page.evaluate(()=>portalCopies.length)).toBe(0);
 const edge=await page.evaluate(()=>{portalRun.level.rooms[0].exit={x:20,y:60};RiftRenderer.snapshot(portalRun,true);return RiftRenderer.frameCount;});
 await expect.poll(()=>page.evaluate(()=>RiftRenderer.frameCount)).toBeGreaterThan(edge);
 const edgeCopies=await page.evaluate(()=>portalCopies);expect(edgeCopies.length).toBeGreaterThan(0);
 for(const [x,y,w,h] of edgeCopies){expect(x).toBeGreaterThanOrEqual(0);expect(y).toBeGreaterThanOrEqual(0);expect(x+w).toBeLessThanOrEqual(960);expect(y+h).toBeLessThanOrEqual(540);}
 await page.evaluate(()=>{portalRun.practice={mode:'movement',arena:{exit:{x:20,y:60}}};RiftRenderer.snapshot(portalRun,true);});
 await expect.poll(()=>page.evaluate(()=>portalRims.length)).toBe(0);expect(await page.evaluate(()=>portalCopies.length)).toBe(0);
 await page.evaluate(()=>{delete portalRun.practice;portalRun.player.x=1500;delete portalRun.level.rooms[0].exit;RiftRenderer.snapshot(portalRun,true);});await page.waitForTimeout(150);
 expect(await page.evaluate(()=>portalRims.length)).toBe(0);expect(errors).toEqual([]);
});
