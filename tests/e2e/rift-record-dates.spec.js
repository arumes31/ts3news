const {test,expect}=require('@playwright/test');

for(const [locale,timezoneId] of [['en-US','America/New_York'],['de-AT','Europe/Vienna']])test.describe(locale,()=>{
 test.use({locale,timezoneId});
 test('record dates use local formatting and survive reload',async({page})=>{
  await page.goto('/abyss/rift?scenario=checkpoint&room=final');await page.locator('#rift-auto').uncheck();await page.locator('#rift-start').click();await page.locator('#rift-next').click();
  const run=(await(await page.request.get('/api/abyss/rift')).json()).run,h=run.mission_history[1];expect(h.best_finish_hp_at_ms).toBe(run.last_ms);expect(h.fewest_hits_at_ms).toBe(run.last_ms);
  const date=await page.evaluate(stamp=>new Date(stamp).toLocaleString(),h.best_finish_hp_at_ms);await expect(page.locator('#rift-history-1')).toContainText('('+date+')');
  await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-history-1')).toContainText('('+date+')');
  await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 });
});
