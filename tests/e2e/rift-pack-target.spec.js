const {test,expect}=require('@playwright/test');
const fs=require('node:fs');
for(const reduced of [false,true])test('Beastmaster command target follows the living mark, reduced='+reduced,async({page})=>{
 await page.emulateMedia({reducedMotion:reduced?'reduce':'no-preference'});await page.goto('/abyss/rift?subclass=beastmaster&scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();const {run}=await(await page.request.get('/api/abyss/rift')).json();const node=page.locator('#rift-pack-target');
 await page.evaluate(()=>{window.packLabels=[];const fill=CanvasRenderingContext2D.prototype.fillText;CanvasRenderingContext2D.prototype.fillText=function(text,x,y,...args){if(text==='PACK TARGET'){if(packLabels[0]?.frame!==RiftRenderer.frameCount)packLabels=[];packLabels.push({x,y,frame:RiftRenderer.frameCount});}return fill.call(this,text,x,y,...args);};});
 const show=(marked,hp=100)=>page.evaluate(({run,marked,hp})=>{const target=run.enemies.find(e=>e.hp>0);target.hp=hp;target.name='Command target';target.x=420;target.y=405;run.enemies=[target];run.marked=marked?target.id:'';run.player.x=160;run.paused=true;run.events=[];packLabels=[];RiftHUD.update(run,false);RiftRenderer.snapshot(run,true);},{run,marked,hp});
 await show(false);await expect(node).toContainText('none');await show(true);await expect(node).toContainText('Command target');await expect(node).toContainText('first enemy in that lane');await expect.poll(()=>page.evaluate(()=>packLabels.length)).toBe(1);
 const png=await page.locator('#rift-canvas').evaluate(c=>c.toDataURL('image/png'));fs.writeFileSync('test-results/pack-target'+(reduced?'-reduced':'')+'.png',Buffer.from(png.split(',')[1],'base64'));
 await show(true,0);await expect(node).toContainText('none');await page.waitForTimeout(100);expect(await page.evaluate(()=>packLabels.length)).toBe(0);
});
