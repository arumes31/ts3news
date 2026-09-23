const {test,expect}=require('@playwright/test');
const fs=require('node:fs');
for(const reduced of [false,true])test((reduced?'reduced motion: ':'')+'hazard labels show activation and safe intervals from the saved combat clock',async({page})=>{
 await page.emulateMedia({reducedMotion:reduced?'reduce':'no-preference'});
 async function capture(name){const data=await page.locator('#rift-canvas').evaluate(canvas=>canvas.toDataURL('image/png'));fs.writeFileSync('test-results/'+name+(reduced?'-reduced':'')+'.png',Buffer.from(data.split(',')[1],'base64'));}
 await page.goto('/abyss/rift?scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.locator('#rift-auto').uncheck();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const data=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(()=>{window.timingLabels=[];const fill=CanvasRenderingContext2D.prototype.fillText;CanvasRenderingContext2D.prototype.fillText=function(text,...args){if(/^([A-Z]+ IN|SAFE ·|JUMP ·|OFF)/.test(String(text)))window.timingLabels.push(text);if(window.timingLabels.length>80)window.timingLabels.splice(0,40);return fill.call(this,text,...args);};});
 async function show(clock,options={}){await page.evaluate(({run,clock,options})=>{run=structuredClone(run);run.clock=clock;run.status=options.status||'fighting';run.paused=true;run.events=[];run.level.rooms[run.room].hazards=[{x:430,y:350,w:150,h:38,kind:options.kind||'fire',period:7,offset:options.offset||0,duration:.8,disabled:!!options.disabled}];RiftDisplay.hazardLabels=options.labels!==false;RiftDisplay.cleanScreenshot=!!options.clean;window.timingLabels=[];RiftRenderer.snapshot(run,true);},{run:data.run,clock,options});}
 for(const [clock,label] of [[0,'FIRE IN 1.2s'],[1.19,'FIRE IN 0.1s'],[1.2,'JUMP · 0.8s'],[1.3,'JUMP · 0.7s'],[2,'SAFE · 6.2s'],[6.9,'SAFE · 1.3s'],[7,'FIRE IN 1.2s']]){
  await show(clock);await expect.poll(()=>page.evaluate(()=>timingLabels.at(-1))).toBe(label);
 }
 await show(0,{offset:2});await expect.poll(()=>page.evaluate(()=>timingLabels.at(-1))).toBe('SAFE · 6.2s');
 await capture('hazard-safe-timer');
 await page.waitForTimeout(250);expect(await page.evaluate(()=>timingLabels.at(-1))).toBe('SAFE · 6.2s');
 await show(0);await expect.poll(()=>page.evaluate(()=>timingLabels.at(-1))).toBe('FIRE IN 1.2s');await capture('hazard-activation-timer');
 for(const kind of ['fire','ice','poison','thorns','rune','radiant','void']){await show(.5,{kind});await expect.poll(()=>page.evaluate(()=>timingLabels.at(-1))).toBe(kind.toUpperCase()+' IN 0.7s');}
 for(const options of [{disabled:true},{status:'cleared'}]){await show(1.3,options);await expect.poll(()=>page.evaluate(()=>timingLabels.at(-1))).toBe('OFF');}
 for(const options of [{labels:false},{clean:true}]){await show(0,options);await page.waitForTimeout(200);expect(await page.evaluate(()=>timingLabels)).toEqual([]);}
});
