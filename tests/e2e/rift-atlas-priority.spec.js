const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path');
test('critical atlas priority and async decoding are set before requests start',async({page})=>{
 await page.route('**/static/rift_renderer.js*',route=>route.fulfill({contentType:'application/javascript',body:fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift_renderer.js'),'utf8')}));
 await page.addInitScript(()=>{
  const src=Object.getOwnPropertyDescriptor(HTMLImageElement.prototype,'src'),decode=HTMLImageElement.prototype.decode,requests=new WeakMap();
  window.atlasPrioritySamples=[];
  Object.defineProperty(HTMLImageElement.prototype,'src',{...src,set(value){requests.set(this,{url:value,priority:this.fetchPriority,decoding:this.decoding});return src.set.call(this,value);}});
  HTMLImageElement.prototype.decode=function(){window.atlasPrioritySamples.push(requests.get(this));return decode.call(this);};
 });
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 const samples=await page.evaluate(()=>window.atlasPrioritySamples);
 expect(samples).toHaveLength(5);
 for(const sample of samples){expect(sample.priority,sample.url).toBe('high');expect(sample.decoding,sample.url).toBe('async');}
 expect(samples.some(sample=>sample.url.includes('rift_prop_0.png'))).toBe(true);
});
