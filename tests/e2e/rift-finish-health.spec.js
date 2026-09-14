const {test,expect}=require('@playwright/test');

test('completed mission health appears on the campaign card and survives reload',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint&room=final&condition=wounded');await page.locator('#rift-auto').uncheck();await page.locator('#rift-start').click();await page.locator('#rift-next').click();
 await expect(page.locator('#rift-history-1')).toContainText('Most HP at finish 170.0/340.0');await expect(page.locator('#rift-history-1')).toContainText('Recorded subclass clears: Vanguard ×1');await expect(page.locator('#rift-history-1')).toContainText('Fewest damaging hits 0');
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;expect(run.mission_history['1'].fewest_hits).toBe(0);expect(run.mission_history['1'].completed_by_class).toEqual({vanguard:1});expect(run.mission_history['1'].best_finish_hp).toBe(170);expect(run.mission_history['1'].best_finish_max_hp).toBe(340);
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-history-1')).toContainText('Most HP at finish 170.0/340.0');await expect(page.locator('#rift-history-1')).toContainText('Recorded subclass clears: Vanguard ×1');await expect(page.locator('#rift-history-1')).toContainText('Fewest damaging hits 0');
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
