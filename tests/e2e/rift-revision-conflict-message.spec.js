const {test,expect}=require('@playwright/test');
test('a real revision conflict explains stale state and recovers without replaying input',async({page})=>{
 await page.goto('/abyss/rift?scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();let conflictStatus=0,rejectedRevision=0,posts=0;
 await page.route('**/api/abyss/rift',async route=>{
  if(route.request().method()==='POST'){
   posts++;const body=route.request().postDataJSON();
   if(body.kind==='step'&&!rejectedRevision){rejectedRevision=body.revision+1;const response=await route.fetch({postData:JSON.stringify({...body,revision:rejectedRevision})});conflictStatus=response.status();await route.fulfill({response});return;}
  }
  await route.continue();
 });
 await page.locator('#rift-start').click();await expect.poll(()=>conflictStatus).toBe(409);
 await expect(page.locator('#rift-overlay-kicker')).toHaveText('SAVE CONFLICT');await expect(page.locator('#rift-overlay-title')).toHaveText('Expedition changed.');await expect(page.locator('#rift-overlay-copy')).toContainText('latest confirmed state');await expect(page.locator('#rift-sign-in')).toBeHidden();
 const saved=await(await page.request.get('/api/abyss/rift')).json();expect(saved.run.revision).toBe(rejectedRevision-2);const stopped=posts;
 await page.waitForTimeout(400);expect(posts).toBe(stopped);await page.getByRole('button',{name:'Reload saved expedition',exact:true}).click();await expect(page.locator('#rift-start')).toHaveText('Resume expedition');await page.waitForTimeout(400);expect(posts).toBe(stopped);
 const recovered=await(await page.request.get('/api/abyss/rift')).json();expect(recovered.run.id).toBe(saved.run.id);expect(recovered.run.revision).toBe(saved.run.revision);expect(recovered.run.banked_items).toEqual(saved.run.banked_items);expect(recovered.run.banked_gold).toBe(saved.run.banked_gold);
 await page.locator('#rift-start').click();await expect.poll(()=>posts).toBeGreaterThan(stopped);await page.keyboard.press('Escape');
});

test('a newer snapshot returned for stale input requires explicit recovery',async({page})=>{
 await page.goto('/abyss/rift?scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();let advanced=null,intercepted=false,posts=0;
 await page.route('**/api/abyss/rift',async route=>{
  if(route.request().method()==='POST'){
   posts++;const body=route.request().postDataJSON();
   if(body.kind==='step'&&!intercepted){
    intercepted=true;let run=(await(await page.request.get('/api/abyss/rift')).json()).run;
    for(let i=0;i<2;i++){const response=await page.request.post('/api/abyss/rift',{data:{kind:'step',run_id:run.id,revision:run.revision+1,request_id:crypto.randomUUID(),input:{}}});expect(response.ok()).toBe(true);run=(await response.json()).run;}
    advanced=run;
   }
  }
  await route.continue();
 });
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay-kicker')).toHaveText('SAVE CONFLICT');expect(advanced).not.toBeNull();const stopped=posts;
 await page.getByRole('button',{name:'Reload saved expedition',exact:true}).click();await expect(page.locator('#rift-start')).toHaveText('Resume expedition');await page.waitForTimeout(400);expect(posts).toBe(stopped);
 const saved=(await(await page.request.get('/api/abyss/rift')).json()).run;expect(saved.revision).toBe(advanced.revision);expect(saved.player.x).toBe(advanced.player.x);expect(saved.banked_items).toEqual(advanced.banked_items);
});
