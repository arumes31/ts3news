const {test,expect}=require('@playwright/test');

test('confirmed attempt history survives reload and replay',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint&room=final');await page.locator('#rift-auto').uncheck();await page.locator('#rift-start').click();await page.locator('#rift-next').click();
 await page.locator('#rift-campaign > summary').click();await page.locator('#rift-attempt-history > summary').click();await expect(page.locator('#rift-attempt-list li')).toHaveCount(1);await expect(page.locator('#rift-attempt-list')).toContainText('Mission 1 · Cleared');await expect(page.locator('#rift-attempt-comparison')).toContainText('No previous recorded attempt');
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await page.locator('#rift-attempt-history > summary').click();await expect(page.locator('#rift-attempt-list li')).toHaveCount(1);
 await page.locator('#rift-replay').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');expect((await(await page.request.get('/api/abyss/rift')).json()).run.attempt_history).toHaveLength(1);
});

test('attempt comparison uses previous matching mission and identifies different outcomes',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const validation=await page.evaluate(async()=>{const data=await(await fetch('/api/abyss/rift')).json(),base={mission:1,outcome:'completed',at_ms:100000,class:'vanguard',seconds:20,hp:80,max_hp:100,hits:2,difficulty:'Wayfarer',definition:'level-v1:'+'a'.repeat(64)};const results=[[base],[{...base,hits:-1}],[{...base,outcome:'unknown'}],Array(51).fill(base),undefined].map(log=>{data.run.attempt_history=log;try{window.RiftProtocol.validate(data,'GET');return true;}catch(_){return false;}});data.run.attempt_history=[base,{...base,mission:2,seconds:1},{...base,outcome:'defeated',seconds:15,hp:0,hits:4}];window.RiftRecords.update(data.run);return results;});
 expect(validation).toEqual([true,false,false,false,true]);await page.locator('#rift-attempt-history > summary').click();await expect(page.locator('#rift-attempt-list li')).toHaveCount(3);await expect(page.locator('#rift-attempt-list li').first()).toContainText('Defeated');await expect(page.locator('#rift-attempt-comparison')).toContainText('Cleared → Defeated');await expect(page.locator('#rift-attempt-comparison')).toContainText('combat time -5.0s; end HP -80.0; damaging hits +2.0');
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
