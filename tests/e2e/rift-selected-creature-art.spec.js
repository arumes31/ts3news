const {test,expect}=require('@playwright/test');

test('idle preview does not download shared creature sheets',async({page})=>{
 const paths=[];page.on('request',r=>{if(/abyss_combat_(?:roles|creatures|bestiary|bosses)_v2\.png/.test(r.url()))paths.push(new URL(r.url()).pathname);});
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 expect(paths).toEqual([]);
});

test('encounter preparation includes current actors, every planned room and saved waves',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 const paths=[];page.on('request',r=>{if(/rift_creature_.*\.png/.test(r.url()))paths.push(new URL(r.url()).pathname);});
 await page.evaluate(async()=>{
  const actor=name=>({name,kind:'goblin',art_key:'bounds-probe:'+name});
  const run={level:{id:1,region:0},enemies:[actor('dragon')],encounter_plan:[[actor('rat')],[],[actor('ranger')]],room_objective:{waves:[[actor('wizard')],[actor('rat')]]}};
  await Promise.all([RiftRenderer.prepareRun(run),RiftRenderer.prepareRun(run)]);
 });
 expect(paths.sort()).toEqual(['dragon','ranger','rat','wizard'].map(rig=>'/static/rift_creature_'+rig+'.png').sort());
});

test('saved creature preparation holds readiness and can recover after a failed row',async({page})=>{
 let fail=true,requested=0;
 await page.route('**/api/abyss/rift',async route=>{
  if(route.request().method()!=='GET')return route.continue();
  const response=await route.fetch(),data=await response.json();
  data.run.enemies=[{...data.run.enemies[0],name:'dragon',kind:'boss',art_key:'bounds-probe:dragon'}];
  await route.fulfill({response,json:data});
 });
 await page.route('**/static/rift_creature_dragon.png*',route=>{requested++;return fail?route.abort():route.continue();});
 await page.goto('/abyss/rift?scenario=visual');
 await expect.poll(()=>requested).toBeGreaterThan(0);
 await expect(page.locator('#rift-overlay')).toBeVisible();
 fail=false;
 await page.evaluate(()=>RiftRenderer.prepareRun({level:{id:1,region:0},enemies:[{name:'dragon',kind:'boss',art_key:'bounds-probe:dragon'}]}));
 expect(requested).toBe(2);
});


test('fresh combat waits visibly for creature artwork before polling',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 let release,requested=0,steps=0;const gate=new Promise(resolve=>release=resolve);
 await page.route('**/api/abyss/rift',async route=>{
  if(route.request().method()!=='POST')return route.continue();
  const body=route.request().postDataJSON();if(body.kind==='step')steps++;
  if(body.kind!=='start')return route.continue();
  const response=await route.fetch(),data=await response.json();
  data.run.enemies[0]={...data.run.enemies[0],name:'dragon',kind:'boss',art_key:'bounds-probe:dragon'};
  await route.fulfill({response,json:data});
 });
 await page.route('**/static/rift_creature_dragon.png*',async route=>{requested++;await gate;await route.continue();});
 try{
  await page.locator('#rift-start').click();await expect.poll(()=>requested).toBe(1);
  await expect(page.locator('#rift-overlay')).toBeVisible();
  await expect(page.locator('#rift-overlay-copy')).toContainText('Preparing your encounter artwork');
  await expect(page.locator('#rift-start')).toBeDisabled();expect(steps).toBe(0);
 }finally{release();}
 await expect(page.locator('#rift-overlay')).toBeHidden();await expect.poll(()=>steps).toBeGreaterThan(0);
 await page.locator('#rift-pause').click();
});


test('boss examples prepare shared artwork even for a rig with local combat animations',async({page})=>{
 await page.goto('/abyss/rift?practice=boss');await expect(page.locator('#rift-start')).toBeEnabled();
 const paths=[];page.on('request',r=>{if(/rift_creature_knight\.png/.test(r.url()))paths.push(new URL(r.url()).pathname);});
 await page.evaluate(()=>{const select=document.getElementById('rift-practice-boss');select.append(new Option('Knight','Knight'));select.value='Knight';select.dispatchEvent(new Event('change'));});
 await page.locator('#rift-boss-jump-example summary').click();await page.locator('#rift-boss-jump-step').click();
 await expect(page.locator('#rift-boss-jump-canvas')).toHaveAttribute('data-stage','warning');
 expect(paths).toEqual(['/static/rift_creature_knight.png']);
});


test('creature preparation retains the decoded image fallback on older browsers',async({page})=>{
 await page.addInitScript(()=>{window.createImageBitmap=undefined;});
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 const paths=[];page.on('request',r=>{if(/rift_creature_dragon\.png/.test(r.url()))paths.push(new URL(r.url()).pathname);});
 await page.evaluate(()=>RiftRenderer.prepareCreatures([{name:'dragon',kind:'boss',art_key:'bounds-probe:dragon'}]));
 expect(paths).toEqual(['/static/rift_creature_dragon.png']);
});
