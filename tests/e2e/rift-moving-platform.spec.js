const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test(`moving platform keyboard journey reduced=${reduced}`,async({page},info)=>{
 test.setTimeout(90000);
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.emulateMedia({reducedMotion:reduced?'reduce':'no-preference'});
 await page.goto('/abyss/rift?practice=moving_platform');
 await expect(page.locator('#rift-practice-title')).toHaveText('Moving platform challenge');
 await expect(page.locator('#rift-practice-instructions')).toContainText('250');
 await expect(page.locator('#rift-pickup-training')).toBeHidden();
 const read=async()=>{const response=await page.request.get('/api/abyss/rift?practice=moving_platform');expect(response.ok()).toBe(true);return(await response.json()).run;};
 await page.evaluate(()=>{const original=RiftAudio.play;window.ferryCues=[];RiftAudio.play=function(kind,...args){const result=original.call(this,kind,...args);if(kind.startsWith('platform_'))ferryCues.push({kind,result});return result;};});
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.keyboard.down('d');
 try{await expect.poll(async()=>(await read()).player.elevation,{timeout:15000,intervals:[40]}).toBe(16);}finally{await page.keyboard.up('d');}
 const aboard=await read();
 await expect.poll(async()=>(await read()).practice.platform_ride||0,{timeout:10000,intervals:[100]}).toBe(250);
 const response=await(await page.request.get('/api/abyss/rift?practice=moving_platform')).json();
 expect(await page.evaluate(data=>{RiftProtocol.validate(data,'GET');return [null,-1,251,NaN,Infinity,{},'1'].every(value=>{const bad=structuredClone(data);bad.run.practice.platform_ride=value;try{RiftProtocol.validate(bad,'GET');return false;}catch{return true;}});},response)).toBe(true);
 await expect.poll(()=>page.evaluate(()=>ferryCues.filter(c=>c.result===true).map(c=>c.kind))).toEqual(['platform_board','platform_ready']);
 const ridden=await read();expect(ridden.player.x).toBeGreaterThan(aboard.player.x+100);
 await expect(page.locator('#rift-practice-progress')).toContainText('Reach the exit');
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const paused=await read();await page.waitForTimeout(300);expect((await read()).practice.arena.platforms[0].x).toBe(paused.practice.arena.platforms[0].x);
 await page.locator('#rift-viewport').screenshot({path:info.outputPath('ferry-desktop.png'),style:'#rift-overlay{visibility:hidden!important}'});
 await page.reload();await expect(page.locator('#rift-start')).toHaveText('Resume drill');
 const recovered=await read();expect(recovered.practice.platform_ride).toBe(250);expect(recovered.practice.arena.platforms).toEqual(paused.practice.arena.platforms);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await page.locator('#rift-viewport').screenshot({path:info.outputPath('ferry-mobile.png'),style:'#rift-overlay{visibility:hidden!important}'});
 await page.locator('#rift-start').click();await page.keyboard.down('d');
 try{await expect.poll(async()=>(await read()).status,{timeout:15000,intervals:[100]}).toBe('complete');}finally{await page.keyboard.up('d');}
 const complete=await read();expect(complete.gold).toBe(0);expect(complete.banked_gold).toBe(0);expect(complete.completed_levels||[]).toEqual([]);
 await page.locator('#rift-practice-reset').click();await expect.poll(async()=>(await read()).practice.platform_ride||0).toBe(0);
 expect(errors).toEqual([]);
});
