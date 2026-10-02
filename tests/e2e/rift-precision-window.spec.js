const {test,expect}=require('@playwright/test');
const fs=require('node:fs');
for(const reduced of [false,true])test('Marksman precision exposes target, charges and readiness, reduced='+reduced,async({page})=>{
 await page.emulateMedia({reducedMotion:reduced?'reduce':'no-preference'});await page.goto('/abyss/rift?subclass=marksman&scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();const {run}=await(await page.request.get('/api/abyss/rift')).json();const node=page.locator('#rift-precision-state');
 await page.evaluate(()=>{window.precisionFrames=0;const stroke=CanvasRenderingContext2D.prototype.stroke;CanvasRenderingContext2D.prototype.stroke=function(...args){if(this.strokeStyle==='#ffe39b')precisionFrames++;return stroke.apply(this,args);};});
 const show=(resource,marked=true,hp=100)=>page.evaluate(({run,resource,marked,hp})=>{const target=run.enemies.find(e=>e.hp>0);target.hp=hp;target.x=420;target.y=405;run.enemies=[target];run.marked=marked?target.id:'';run.resource=resource;run.player.x=160;run.paused=true;run.events=[];window.precisionFrames=0;RiftHUD.update(run,false);RiftRenderer.snapshot(run,true);},{run,resource,marked,hp});
 await show(0,false);await expect(node).toContainText('land a builder hit');
 await show(0);await expect(node).toContainText('Build charges');await page.waitForTimeout(100);expect(await page.evaluate(()=>precisionFrames)).toBe(0);
 await show(2);await expect(node).toContainText('Precision armed on');await expect(node).toContainText('+60 percentage points');await expect(node).toContainText('Paused');await expect.poll(()=>page.evaluate(()=>precisionFrames)).toBeGreaterThan(0);
 const png=await page.locator('#rift-canvas').evaluate(c=>c.toDataURL('image/png'));fs.writeFileSync('test-results/precision'+(reduced?'-reduced':'')+'.png',Buffer.from(png.split(',')[1],'base64'));
 await show(2,true,0);await expect(node).toContainText('land a builder hit');
});
