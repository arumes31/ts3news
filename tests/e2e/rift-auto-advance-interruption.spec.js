const {test,expect}=require('@playwright/test');
for(const reason of ['connection','hidden'])test('auto-advance stops after '+reason+' and restarts only on resume',async({page})=>{
 let fail=reason==='connection';const advances=[];
 await page.route('**/api/abyss/rift*',async route=>{const body=route.request().postDataJSON();if(body?.kind==='advance'){advances.push(body);if(fail){await route.abort();return;}}await route.continue();});
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const before=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.locator('#rift-start').click();await expect(page.locator('#rift-transition')).toContainText('Next:');
 if(reason==='hidden'){
  await page.evaluate(()=>{Object.defineProperty(document,'hidden',{configurable:true,value:true});document.dispatchEvent(new Event('visibilitychange'));});
  await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
 }else await expect(page.locator('#rift-start')).toHaveText('Recover expedition');
 const count=advances.length;expect(count).toBe(reason==='hidden'?0:1);
 // Longer than the default countdown: stale callbacks must not bank or advance.
 await page.waitForTimeout(1600);expect(advances).toHaveLength(count);
 expect((await(await page.request.get('/api/abyss/rift')).json()).run.room).toBe(before.room);
 if(reason==='hidden'){
  await page.evaluate(()=>{Object.defineProperty(document,'hidden',{configurable:true,value:false});document.dispatchEvent(new Event('visibilitychange'));});
  await page.waitForTimeout(1500);expect(advances).toHaveLength(count);
 }else{fail=false;await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toHaveText('Resume expedition');}
 await page.locator('#rift-start').click();await expect(page.locator('#rift-transition')).toContainText('Next:');
 await page.waitForTimeout(400);expect(advances).toHaveLength(count);
 await expect.poll(()=>advances.length).toBe(count+1);
 await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.room).toBe(before.room+1);
 await page.locator('#rift-pause').click();
});
