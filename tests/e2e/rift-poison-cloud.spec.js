const {test,expect}=require('@playwright/test');
const fs=require('node:fs');
for(const reduced of [false,true])test('poison cloud dissipates after damage stops, reduced='+reduced,async({page})=>{
 await page.emulateMedia({reducedMotion:reduced?'reduce':'no-preference'});await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();const {run}=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(()=>{window.cloudMarks=[];const ellipse=CanvasRenderingContext2D.prototype.ellipse;CanvasRenderingContext2D.prototype.ellipse=function(...args){if(this.fillStyle==='#b8d778'){if(cloudMarks[0]?.frame!==RiftRenderer.frameCount)cloudMarks=[];cloudMarks.push({alpha:this.globalAlpha,args,frame:RiftRenderer.frameCount});}return ellipse.apply(this,args);};});
 const show=clock=>page.evaluate(({run,clock})=>{run.clock=clock;run.paused=true;run.status='fighting';run.player.x=160;run.enemies=[];run.events=[];run.level.rooms[run.room].hazards=[{kind:'poison',x:420,y:370,w:180,h:60,period:7,offset:0,duration:1,jumpable:true}];cloudMarks=[];RiftRenderer.snapshot(run,true);},{run,clock});
 await show(1.5);await expect.poll(()=>page.evaluate(()=>cloudMarks.length)).toBe(7);const active=await page.evaluate(()=>cloudMarks[0].alpha);
 await show(2.3);await expect.poll(()=>page.evaluate(()=>cloudMarks.length)).toBe(7);const early=await page.evaluate(()=>cloudMarks[0].alpha);expect(early).toBeLessThan(active);
 const data=await page.locator('#rift-canvas').evaluate(c=>c.toDataURL('image/png'));fs.writeFileSync('test-results/poison-cloud'+(reduced?'-reduced':'')+'.png',Buffer.from(data.split(',')[1],'base64'));
 await page.waitForTimeout(150);expect(await page.evaluate(()=>cloudMarks[0].alpha)).toBe(early);
 await show(2.5);await expect.poll(()=>page.evaluate(()=>cloudMarks[0]?.alpha)).toBeLessThan(early);
 for(const clock of [0,2.7]){await show(clock);await page.waitForTimeout(100);expect(await page.evaluate(()=>cloudMarks.length)).toBe(0);}
});
