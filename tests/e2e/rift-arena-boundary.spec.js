const {test,expect}=require('@playwright/test');
const fs=require('node:fs');
for(const reduced of [false,true])test('arena rim matches walk bounds without debug grid, reduced='+reduced,async({page})=>{
 await page.emulateMedia({reducedMotion:reduced?'reduce':'no-preference'});
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const {run}=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(()=>{window.boundaryRects=[];window.edgeLabels=[];const stroke=CanvasRenderingContext2D.prototype.strokeRect,fill=CanvasRenderingContext2D.prototype.fillText;
 CanvasRenderingContext2D.prototype.strokeRect=function(x,y,w,h){if(w===1530&&h===175){boundaryRects.push({x,y,w,h});if(boundaryRects.length>20)boundaryRects.shift();}return stroke.apply(this,arguments);};
 CanvasRenderingContext2D.prototype.fillText=function(text,...args){if(text==='ARENA EDGE'){edgeLabels.push(text);if(edgeLabels.length>20)edgeLabels.shift();}return fill.call(this,text,...args);};});
 const show=(x,y,clean=false)=>page.evaluate(({run,x,y,clean})=>{run.paused=true;run.player.x=x;run.player.y=y;run.enemies=[];run.events=[];run.drops=[];RiftDisplay.layoutGrid=false;RiftDisplay.cleanScreenshot=clean;boundaryRects=[];edgeLabels=[];RiftRenderer.snapshot(run,true);},{run,x,y,clean});
 await show(160,410);await expect.poll(()=>page.evaluate(()=>boundaryRects.at(-1))).toEqual({x:35,y:315,w:1530,h:175});expect(await page.evaluate(()=>edgeLabels.length)).toBe(0);
 for(const [x,y] of [[35,410],[160,315],[160,490],[1565,410]]){await show(x,y);await expect.poll(()=>page.evaluate(()=>edgeLabels.length)).toBeGreaterThan(0);}
 await expect.poll(()=>page.evaluate(()=>boundaryRects.at(-1).x)).toBe(-605);
 await show(160,490);await expect.poll(()=>page.evaluate(()=>edgeLabels.length)).toBeGreaterThan(0);
 const data=await page.locator('#rift-canvas').evaluate(c=>c.toDataURL('image/png'));fs.writeFileSync('test-results/arena-boundary'+(reduced?'-reduced':'')+'.png',Buffer.from(data.split(',')[1],'base64'));
 await show(160,490,true);await expect.poll(()=>page.evaluate(()=>boundaryRects.length)).toBeGreaterThan(0);expect(await page.evaluate(()=>edgeLabels.length)).toBe(0);
});

test('WASD stops on the marked floor limits',async({page})=>{
 await page.goto('/abyss/rift?scenario=terrain-cover');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 for(const [key,axis,value] of [['KeyW','y',315],['KeyS','y',490],['KeyA','x',35]]){
  await page.keyboard.down(key);try{await expect.poll(async()=>(await saved()).player[axis]).toBe(value);await page.waitForTimeout(100);expect((await saved()).player[axis]).toBe(value);}finally{await page.keyboard.up(key);}
 }
});
