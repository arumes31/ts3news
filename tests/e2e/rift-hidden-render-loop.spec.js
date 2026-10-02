const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path');
test('hidden document suspends the render loop and resumes one loop',async({page})=>{
 await page.addInitScript(()=>{
  const request=window.requestAnimationFrame;
  window.renderRequests=0;
  window.requestAnimationFrame=function(callback){if(callback.name==='render')window.renderRequests++;return request.call(this,callback);};
 });
 await page.route('**/static/rift_renderer.js*',route=>route.fulfill({contentType:'application/javascript',body:fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift_renderer.js'),'utf8')}));
 await page.goto('/abyss/rift?scenario=checkpoint');
 await expect(page.locator('#rift-start')).toBeEnabled();
 await expect.poll(()=>page.evaluate(()=>window.RiftRenderer.frameCount)).toBeGreaterThan(2);
 const before=await page.evaluate(()=>{
  Object.defineProperty(document,'hidden',{configurable:true,value:true});
  document.dispatchEvent(new Event('visibilitychange'));
  return {requests:window.renderRequests,frames:window.RiftRenderer.frameCount};
 });
 await page.waitForTimeout(250);
 expect(await page.evaluate(()=>({requests:window.renderRequests,frames:window.RiftRenderer.frameCount}))).toEqual(before);
 const resumed=await page.evaluate(()=>{
  Object.defineProperty(document,'hidden',{configurable:true,value:false});
  for(let i=0;i<10;i++)document.dispatchEvent(new Event('visibilitychange'));
  return window.renderRequests;
 });
 expect(resumed).toBe(before.requests+1);
 await expect.poll(()=>page.evaluate(()=>window.RiftRenderer.frameCount)).toBeGreaterThan(before.frames);
});
