const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test('boss enrage countdown transition pause reload and reset, reduced='+reduced,async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.emulateMedia({reducedMotion:reduced?'reduce':'no-preference'});
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const campaign=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.goto('/abyss/rift?practice=boss&scenario=practice-enrage');await expect(page.locator('#rift-start')).toBeEnabled();
 const read=async()=>(await(await page.request.get('/api/abyss/rift?practice=boss')).json()).run;
 const timer=page.locator('#rift-boss-enrage'),choice=page.locator('#rift-practice-enrage');await expect(choice).toBeChecked();
 await expect(timer).toHaveText('Enrage in 2s');const paused=await read();await page.waitForTimeout(700);expect((await read()).clock).toBe(paused.clock);
 await page.evaluate(()=>{window.enrageCues=0;const play=RiftAudio.play;RiftAudio.play=function(kind,...args){if(kind==='practice_enrage')enrageCues++;return play.call(this,kind,...args);};});
 await page.locator('#rift-start').click();await expect(timer).toBeVisible();await page.locator('#rift-viewport').screenshot({path:info.outputPath('countdown.png')});await expect(timer).toHaveText('ENRAGED · Boss damage +50%',{timeout:10000});
 await expect(page.locator('#rift-practice-announcement')).toContainText('Enraged: boss damage increased by 50 percent');
 await expect(timer).toBeVisible();await page.locator('#rift-viewport').screenshot({path:info.outputPath('enraged.png')});
 await page.setViewportSize({width:390,height:844});await expect(timer).toBeVisible();expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await page.locator('#rift-viewport').screenshot({path:info.outputPath('enraged-mobile.png')});
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 expect((await read()).practice.enrage_triggered).toBe(true);expect(await page.evaluate(()=>enrageCues)).toBe(1);
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift?practice=boss'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(choice).toBeChecked();await expect(timer).toHaveText('ENRAGED · Boss damage +50%');
 await page.locator('#rift-practice-reset').click();await expect(timer).toHaveText('Enrage in 30s');expect((await read()).clock).toBe(0);expect((await read()).practice.enrage_triggered||false).toBe(false);
 await choice.uncheck();await page.locator('#rift-practice-reset').click();await expect(timer).toBeHidden();expect((await read()).practice.enrage_seconds||0).toBe(0);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(campaign);expect(errors).toEqual([]);
});

test('timed practice records stay separate and invalid enrage snapshots are rejected',async({page})=>{
 await page.goto('/abyss/rift?practice=boss&scenario=practice-enrage');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift?practice=boss')).json();
 const result=await page.evaluate(data=>{
  const accepted=[];
  for(const value of [-1,1,29,31,'30']){const bad=structuredClone(data);bad.run.practice.enrage_seconds=value;try{RiftProtocol.validate(bad);accepted.push(value);}catch(_){}}
  const early=structuredClone(data);early.run.practice.enrage_triggered=true;try{RiftProtocol.validate(early);accepted.push('early');}catch(_){}
  const run=structuredClone(data.run);run.status='complete';run.practice.completed=true;run.practice.enrage_seconds=0;run.stats.seconds=5;RiftRecords.update(run);
  const ordinary=document.querySelector('#rift-practice-best').textContent;
  run.practice.enrage_seconds=30;run.stats.seconds=7;RiftRecords.update(run);const timed=document.querySelector('#rift-practice-best').textContent;
  run.practice.enrage_seconds=0;run.stats.seconds=9;RiftRecords.update(run);const restored=document.querySelector('#rift-practice-best').textContent;
  return {accepted,ordinary,timed,restored};
 },data);
 expect(result.accepted).toEqual([]);expect(result.ordinary).toContain('5.00s');expect(result.timed).toContain('7.00s');expect(result.restored).toBe(result.ordinary);
});

test('new boss drill opts into timed challenge explicitly',async({page})=>{
 await page.goto('/abyss/rift?practice=boss');await expect(page.locator('#rift-start')).toBeEnabled();
 await expect(page.locator('#rift-practice-enrage')).not.toBeChecked();await page.locator('#rift-practice-enrage').check();await page.locator('#rift-start').click();
 await expect(page.locator('#rift-boss-enrage')).toBeVisible();await expect(page.locator('#rift-boss-enrage')).toContainText('Enrage in');
 const run=(await(await page.request.get('/api/abyss/rift?practice=boss')).json()).run;expect(run.practice.enrage_seconds).toBe(30);expect(run.practice.enrage_triggered||false).toBe(false);
});
