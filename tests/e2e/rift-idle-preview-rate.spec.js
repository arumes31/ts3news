const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path');
for(const combat of [false,true])test(`lower-power frame budget for ${combat?'combat':'idle preview'}`,async({page})=>{
 await page.addInitScript(()=>{
  const request=window.requestAnimationFrame;
  window.requestAnimationFrame=function(callback){
   if(callback.name==='render'){window.renderProbe=callback;return 123456789;}
   return request.call(this,callback);
  };
 });
 await page.route('**/static/rift_renderer.js*',route=>route.fulfill({contentType:'application/javascript',body:fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift_renderer.js'),'utf8')}));
 await page.goto('/abyss/rift'+(combat?'?scenario=checkpoint':''));
 await expect(page.locator('#rift-start')).toBeEnabled();
 await page.waitForFunction(()=>typeof window.renderProbe==='function');
 const counts=await page.evaluate(()=>{
  let now=performance.now()+1000;
  return [60,30].map(fps=>{
   window.RiftDisplay.fps=fps;
   window.renderProbe(now);
   const before=window.RiftRenderer.frameCount;
   for(let i=0;i<120;i++){now+=1000/60;window.renderProbe(now);}
   return window.RiftRenderer.frameCount-before;
  });
 });
 expect(counts).toEqual([120,combat?60:30]);
});
