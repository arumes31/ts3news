const {test,expect}=require('@playwright/test');

test('controls queued behind an idle request reach the next serial save',async({page})=>{
 let release;const held=new Promise(resolve=>release=resolve),steps=[];
 await page.route('**/api/abyss/rift*',async route=>{
  const body=route.request().postDataJSON();
  if(body?.kind==='step'){steps.push(body);if(steps.length===1)await held;}
  await route.continue();
 });
 await page.goto('/abyss/rift?practice=touch');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.locator('#rift-canvas').focus();
 await expect.poll(()=>steps.length).toBe(1);
 const before=(await(await page.request.get('/api/abyss/rift?practice=touch')).json()).run;
 await page.keyboard.press('Space');await page.keyboard.press('d');
 await page.waitForTimeout(150);expect(steps).toHaveLength(1);
 const next=page.waitForResponse(r=>{const body=r.request().postDataJSON();return new URL(r.url()).pathname==='/api/abyss/rift'&&body?.kind==='step'&&body.input.jump&&body.input.x>0;});
 release();const response=await next;expect(response.ok()).toBe(true);
 const requested=response.request().postDataJSON(),after=(await response.json()).run;
 expect(steps[1].request_id).toBe(requested.request_id);
 expect(after.id).toBe(before.id);expect(after.revision).toBe(requested.revision);
 expect(after.stats.jumps).toBe(before.stats.jumps+1);expect(after.player.x).toBeGreaterThan(before.player.x);
 await page.keyboard.press('Escape');await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift?practice=touch')).json()).run.paused).toBe(true);
});
