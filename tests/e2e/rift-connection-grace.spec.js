const {test,expect}=require('@playwright/test');
test('interrupted active connection grants visible temporary protection',async({page})=>{
 const errors=[];page.on('pageerror',error=>errors.push(error.stack||error.message));
 await page.goto('/abyss/rift?scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();let {run}=await(await page.request.get('/api/abyss/rift')).json();
 const send=async kind=>{const response=await page.request.post('/api/abyss/rift',{data:{kind,run_id:run.id,revision:run.revision+1,request_id:crypto.randomUUID(),input:{}}});expect(response.ok()).toBe(true);run=(await response.json()).run;};
 await send('resume');await page.waitForTimeout(2100);await send('resume');expect(run.skill_timers.connection_grace).toBe(1.2);
 await page.evaluate(async run=>{await RiftRenderer.prepareRun(run);RiftHUD.update(run,false);RiftRenderer.snapshot(run,true);},run);
 await page.waitForTimeout(300);expect(errors).toEqual([]);
 expect(await page.evaluate(run=>RiftHUD.detectPlayerAreaEffects(run).name,run)).toBe('Connection protection');await expect(page.locator('#rift-area-effects')).toHaveText('Connection recovered: protected (1.2s)');await expect(page.locator('#rift-area-effects')).toHaveAttribute('data-effect-state','evading');
 await send('step');expect(run.skill_timers.connection_grace).toBeGreaterThan(0);expect(run.skill_timers.connection_grace).toBeLessThan(1.2);
 await send('pause');const frozen=run.skill_timers.connection_grace;await page.waitForTimeout(2100);await send('resume');expect(run.skill_timers.connection_grace).toBe(frozen);
});
