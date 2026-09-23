const {test,expect}=require('@playwright/test');
const fs=require('node:fs');
for(const reduced of [false,true])test('disabled hazards become static scenery, reduced='+reduced,async({page})=>{
 await page.emulateMedia({reducedMotion:reduced?'reduce':'no-preference'});
 await page.goto('/abyss/rift?scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.evaluate(()=>{window.inertMarks=[];const stroke=CanvasRenderingContext2D.prototype.stroke,rect=CanvasRenderingContext2D.prototype.strokeRect;CanvasRenderingContext2D.prototype.stroke=function(...args){if(this.strokeStyle==='#53645a')inertMarks.push({kind:'residue',alpha:this.globalAlpha,dash:this.getLineDash()});return stroke.apply(this,args);};CanvasRenderingContext2D.prototype.strokeRect=function(...args){if(this.strokeStyle==='#75a796')inertMarks.push({kind:'warning-box'});return rect.apply(this,args);};});
 for(const status of ['fighting','cleared']){
  await page.evaluate(({run,status})=>{run.paused=true;run.status=status;run.player.x=160;run.enemies=[];run.events=[];run.drops=[];run.level.rooms[run.room].hazards=['fire','ice','poison','thorns','rune','radiant','void'].map((kind,i)=>({kind,x:300+(i%4)*140,y:340+Math.floor(i/4)*80,w:110,h:40,period:7,offset:0,duration:1,jumpable:true,disabled:status==='fighting'}));inertMarks=[];RiftRenderer.snapshot(run,true);},{run,status});
  await expect.poll(()=>page.evaluate(()=>inertMarks.filter(m=>m.kind==='residue').length)).toBeGreaterThanOrEqual(7);
  expect(await page.evaluate(()=>inertMarks.some(m=>m.kind==='warning-box'))).toBe(false);
  expect(await page.evaluate(()=>inertMarks.filter(m=>m.kind==='residue').every(m=>m.dash.length===0))).toBe(true);
 }
 const png=await page.locator('#rift-canvas').evaluate(c=>c.toDataURL('image/png'));fs.writeFileSync('test-results/inert-hazards'+(reduced?'-reduced':'')+'.png',Buffer.from(png.split(',')[1],'base64'));
});
