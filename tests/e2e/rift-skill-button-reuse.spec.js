const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path');
for(const empty of [false,true])test('unchanged skill controls are reused: '+(empty?'empty':'populated'),async({page})=>{
 await page.route('**/static/rift.js*',route=>route.fulfill({contentType:'application/javascript',body:fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift.js'),'utf8')}));
 if(empty)await page.route('**/api/abyss/rift*',async route=>{
  const response=await route.fetch(),data=await response.json();
  if(data.build)data.build.skills=[];if(data.run)data.run.build.skills=[];
  await route.fulfill({response,json:data});
 });
 await page.addInitScript(()=>{
  const replace=Element.prototype.replaceChildren;window.skillRebuilds=0;
  Element.prototype.replaceChildren=function(...nodes){if(this.id==='rift-skills'||this.id==='rift-signatures')window.skillRebuilds++;return replace.apply(this,nodes);};
 });
 await page.goto('/abyss/rift?scenario=checkpoint');
 await expect(page.locator('#rift-start')).toBeEnabled();
 const result=await page.evaluate(()=>{
  const before=[...document.querySelectorAll('#rift-skills button,#rift-signatures button')];
  window.skillRebuilds=0;
  for(let i=0;i<12;i++)window.dispatchEvent(new Event('riftbindingschange'));
  const after=[...document.querySelectorAll('#rift-skills button,#rift-signatures button')];
  return {rebuilds:window.skillRebuilds,same:before.length===after.length&&before.every((node,i)=>node===after[i]),skills:document.querySelector('#rift-skills').childElementCount};
 });
 expect(result.rebuilds).toBe(0);expect(result.same).toBe(true);
 if(empty)expect(result.skills).toBe(0);else {
  expect(result.skills).toBeGreaterThan(0);
  const repaired=await page.evaluate(()=>{
   document.querySelector('#rift-skills button').remove();window.skillRebuilds=0;
   window.dispatchEvent(new Event('riftbindingschange'));
   window.dispatchEvent(new Event('riftbindingschange'));
   return {rebuilds:window.skillRebuilds,count:document.querySelector('#rift-skills').childElementCount};
  });
  expect(repaired).toEqual({rebuilds:1,count:result.skills});
 }
});
