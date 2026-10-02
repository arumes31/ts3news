const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path');
test('adaptive background particles reduce on sustained slow frames and recover',async({page})=>{
 await page.addInitScript(()=>{const request=window.requestAnimationFrame;window.requestAnimationFrame=function(callback){if(callback.name==='render'){window.renderProbe=callback;return 123456789;}return request.call(this,callback);};});
 for(const file of ['rift_renderer.js','rift_display.js'])await page.route('**/static/'+file+'*',route=>route.fulfill({contentType:'application/javascript',body:fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets',file),'utf8')}));
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();await page.waitForFunction(()=>window.renderProbe);
 await page.locator('.rift-settings > summary').click();
 expect(await page.locator('#rift-adaptive-particles').count()).toBe(1);
 await page.locator('#rift-adaptive-particles').check();
 expect(await page.evaluate(()=>JSON.parse(localStorage.getItem('riftDisplay')).adaptiveParticles)).toBe(true);
 const counts=await page.evaluate(()=>{
  Object.assign(window.RiftDisplay,{fps:60,particles:true,particleIntensity:1,motionIntensity:1});window.RiftRenderer.reduced=false;
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillRect;let particles=0,now=performance.now()+1000;
  ctx.fillRect=function(x,y,w,h){if(w===2&&h===2&&['#a9ce8c','#ffd98a'].includes(this.fillStyle))particles++;return fill.call(this,x,y,w,h);};
  const frame=ms=>{particles=0;window.renderProbe(now+=ms);return particles;};
  try{
   const full=frame(20);for(let i=0;i<12;i++)frame(60);const slow=frame(60);
   for(let i=0;i<360;i++)frame(20);const recovered=frame(20);
   for(let i=0;i<12;i++)frame(60);window.RiftDisplay.adaptiveParticles=false;const disabled=frame(20);
   window.RiftRenderer.reduced=true;const reduced=frame(20);
   return {full,slow,recovered,disabled,reduced};
  }finally{ctx.fillRect=fill;}
 });
 expect(counts).toEqual({full:22,slow:6,recovered:22,disabled:22,reduced:0});
});
