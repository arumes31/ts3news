const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path');
for(const enabled of [false,true])test('frame diagnostics '+(enabled?'measure a bounded history':'stay opt-in'),async({page})=>{
 await page.addInitScript(()=>{const request=window.requestAnimationFrame;window.requestAnimationFrame=function(callback){if(callback.name==='render'){window.renderProbe=callback;return 123456789;}return request.call(this,callback);};});
 await page.route('**/static/rift_renderer.js*',route=>route.fulfill({contentType:'application/javascript',body:fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift_renderer.js'),'utf8')}));
 await page.goto('/abyss/rift?scenario=checkpoint'+(enabled?'&riftFrameDebug=1':''));await expect(page.locator('#rift-start')).toBeEnabled();await page.waitForFunction(()=>window.renderProbe);
 expect(await page.locator('#rift-frame-diagnostics').count()).toBe(enabled?1:0);
 if(!enabled)return;
 const result=await page.evaluate(()=>{
  const renderer=window.RiftRenderer;let now=performance.now()+1000;window.RiftDisplay.fps=60;
  for(let i=0;i<241;i++)window.renderProbe(now+=20);
  const samples=renderer.frameDiagnostics.samples;
  return {count:renderer.frameDiagnostics.count,length:samples.length,intervals:samples.map(s=>s.interval),costs:samples.map(s=>s.render)};
 });
 expect(result.count).toBe(240);expect(result.length).toBe(120);
 for(const interval of result.intervals)expect(interval).toBeCloseTo(20,5);
 for(const cost of result.costs){expect(Number.isFinite(cost)).toBe(true);expect(cost).toBeGreaterThanOrEqual(0);}
 await expect(page.locator('#rift-frame-diagnostics')).toContainText('Interval');
 await expect(page.locator('#rift-frame-diagnostics')).toContainText('Render');
 await expect(page.locator('#rift-frame-diagnostics')).toContainText('p95');
});
