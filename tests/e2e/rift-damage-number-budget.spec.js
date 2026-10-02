const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path');
for(const reduced of [false,true])test('damage number budget keeps newest hits and healing: reduced '+reduced,async({page})=>{
 await page.route('**/static/rift_renderer.js*',route=>route.fulfill({contentType:'application/javascript',body:fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift_renderer.js'),'utf8')}));
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.evaluate(({run,reduced})=>{
  window.RiftAudio.set('muted',true);Object.assign(window.RiftDisplay,{damageNumbers:true,optionalCombatText:true,cleanScreenshot:false});
  const renderer=window.RiftRenderer;renderer.reduced=reduced;
  run.id='damage-budget';run.paused=true;run.status='fighting';run.enemies=[];run.counter=40;
  run.events=Array.from({length:40},(_,i)=>({id:i+1,kind:i<36?'hit':'heal',value:10000+i,x:400,y:400}));
  const context=document.querySelector('#rift-canvas').getContext('2d'),fill=context.fillText;
  window.budgetLabels=new Set();context.fillText=function(label,...args){if(/^100[0-9]{2}$|^\+100[0-9]{2} HP$/.test(String(label)))window.budgetLabels.add(String(label));return fill.call(this,label,...args);};
  renderer.snapshot(run,false);
 },{run,reduced});
 await expect.poll(()=>page.evaluate(()=>[...window.budgetLabels].filter(label=>label.startsWith('+')).length)).toBe(4);
 const labels=await page.evaluate(()=>[...window.budgetLabels].sort());
 const expected=[...Array.from({length:16},(_,i)=>String(10020+i)),...Array.from({length:4},(_,i)=>'+'+(10036+i)+' HP')].sort();
 expect(labels).toEqual(expected);
 await page.evaluate(()=>{window.RiftDisplay.damageNumbers=false;window.budgetLabels.clear();});
 await page.waitForTimeout(150);
 expect(await page.evaluate(()=>[...window.budgetLabels].every(label=>label.startsWith('+')))).toBe(true);
});
