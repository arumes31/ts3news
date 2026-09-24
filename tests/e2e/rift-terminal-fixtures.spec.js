const {test,expect}=require('@playwright/test');
for(const state of ['complete','banked','defeated','expired'])test('terminal '+state+' fixture renders and survives reload',async({page})=>{
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.goto('/abyss/rift?scenario=terminal-'+state);
 await expect(page.locator('#rift-start')).toBeEnabled();
 const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const run=await read();expect(run.status).toBe(state);
 await expect(page.locator('#rift-overlay')).toBeVisible();
 await expect(page.locator('#rift-overlay-kicker')).toHaveText(state==='expired'?'EXPEDITION EXPIRED':state==='defeated'?'EXPEDITION ENDED':'REWARDS SECURED');
 await expect(page.locator('#rift-room-actions')).toBeHidden();
 if(state==='expired')await expect(page.locator('#rift-result-actions')).toBeHidden();
 else await expect(page.locator('#rift-result-actions')).toBeVisible();
 if(state==='complete')await expect(page.locator('#rift-replay')).toBeVisible();
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));
 await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();
 const restored=await read();expect(restored.id).toBe(run.id);expect(restored.status).toBe(state);expect(restored.banked_gold).toBe(run.banked_gold);
 expect(errors).toEqual([]);
});
