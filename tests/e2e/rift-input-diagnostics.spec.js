const {test,expect}=require('@playwright/test');

test('input confirmation overlay includes delayed response and excludes idle steps',async({page})=>{
 let delay=true;
 await page.route('**/api/abyss/rift*',async route=>{
  const body=route.request().postDataJSON();
  if(delay&&body?.kind==='step'&&body.input.x){delay=false;await new Promise(resolve=>setTimeout(resolve,350));}
  await route.continue();
 });
 await page.goto('/abyss/rift?practice=movement&riftInputDebug=1');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.waitForTimeout(250);
 expect(await page.evaluate(()=>window.RiftInputDiagnostics?.count)).toBe(0);
 await page.keyboard.down('d');
 await expect.poll(()=>page.evaluate(()=>window.RiftInputDiagnostics?.count)).toBe(1);
 await page.keyboard.up('d');
 const sample=await page.evaluate(()=>window.RiftInputDiagnostics.samples[0]);
 expect(sample.total).toBeGreaterThanOrEqual(350);expect(sample.request).toBeGreaterThanOrEqual(350);expect(sample.queue).toBeGreaterThanOrEqual(0);
 expect(sample.actions).toEqual(['right']);expect(sample.total).toBeGreaterThanOrEqual(sample.queue+sample.request);
 for(const phase of ['headers','decode','validation','apply'])expect(sample[phase]).toBeGreaterThanOrEqual(0);
 expect(sample.headers).toBeGreaterThanOrEqual(350);
 expect(sample.headers+sample.decode+sample.validation).toBeCloseTo(sample.request,5);
 expect(sample.queue+sample.request+sample.apply).toBeCloseTo(sample.total,5);
 await expect(page.locator('#rift-input-diagnostics')).toContainText('Input confirmation');
 await page.waitForTimeout(250);expect(await page.evaluate(()=>window.RiftInputDiagnostics.count)).toBe(1);
});

test('input diagnostics are absent by default',async({page})=>{
 await page.goto('/abyss/rift?practice=movement');await expect(page.locator('#rift-start')).toBeEnabled();
 expect(await page.evaluate(()=>window.RiftInputDiagnostics)).toBeUndefined();await expect(page.locator('#rift-input-diagnostics')).toHaveCount(0);
});


test('input history stays bounded and failed requests do not confirm',async({page})=>{
 let fail=false;
 await page.route('**/api/abyss/rift*',async route=>{
  const body=route.request().postDataJSON();
  if(fail&&body?.kind==='step'&&body.input.x){await route.fulfill({status:503,body:'unavailable'});return;}
  await route.continue();
 });
 await page.goto('/abyss/rift?practice=movement&riftInputDebug=1');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 for(let i=1;i<=125;i++){
  await page.keyboard.press(i%2?'d':'a');
  await expect.poll(()=>page.evaluate(()=>window.RiftInputDiagnostics.count),{intervals:[20,40,80]}).toBe(i);
 }
 const diagnostics=await page.evaluate(()=>window.RiftInputDiagnostics);
 expect(diagnostics.samples).toHaveLength(120);
 expect(diagnostics.samples[0].actions).toEqual(['left']);expect(diagnostics.samples[119].actions).toEqual(['right']);
 fail=true;await page.keyboard.press('a');await expect(page.locator('#rift-start')).toHaveText('Recover expedition');
 expect(await page.evaluate(()=>window.RiftInputDiagnostics.count)).toBe(125);
});

test('dodge timing waits for the response carrying the keyboard action',async({page})=>{
 await page.route('**/api/abyss/rift*',async route=>{
  const body=route.request().postDataJSON();
  if(body?.kind==='step'&&body.input.dodge)await new Promise(resolve=>setTimeout(resolve,350));
  await route.continue();
 });
 await page.goto('/abyss/rift?practice=touch&riftInputDebug=1');
 await expect(page.locator('#rift-start')).toBeEnabled();
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.locator('#rift-canvas').focus();
 const response=page.waitForResponse(r=>{
  const body=r.request().postDataJSON();
  return new URL(r.url()).pathname==='/api/abyss/rift'&&body?.kind==='step'&&body.input.dodge;
 });
 await page.keyboard.press('c');
 const accepted=await response;expect(accepted.ok()).toBe(true);
 const body=accepted.request().postDataJSON(),data=await accepted.json();
 expect(data.run.id).toBe(body.run_id);expect(data.run.revision).toBe(body.revision);
 expect(data.run.skill_timers.dodge_cooldown).toBeGreaterThan(0);
 await expect.poll(()=>page.evaluate(()=>window.RiftInputDiagnostics.count)).toBe(1);
 const sample=await page.evaluate(()=>window.RiftInputDiagnostics.samples[0]);
 expect(sample.actions).toEqual(['dodge']);
 expect(sample.request).toBeGreaterThanOrEqual(350);
 expect(sample.total).toBeGreaterThanOrEqual(sample.queue+sample.request);
});
