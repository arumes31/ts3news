const {test,expect}=require('@playwright/test');

test('record export downloads confirmed history and career totals without changing the expedition',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint&room=final');await page.locator('#rift-auto').uncheck();await page.locator('#rift-start').click();await page.locator('#rift-next').click();await page.locator('#rift-campaign > summary').click();await page.locator('#rift-attempt-history > summary').click();
 const before=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await expect(page.locator('#rift-export-records')).toBeEnabled();const pending=page.waitForEvent('download');await page.locator('#rift-export-records').click();const download=await pending;expect(download.suggestedFilename()).toMatch(/^rift-brawl-records-\d{4}-\d{2}-\d{2}\.json$/);
 const stream=await download.createReadStream();const chunks=[];for await(const chunk of stream)chunks.push(chunk);const exported=JSON.parse(Buffer.concat(chunks).toString('utf8'));
 expect(exported.version).toBe(1);expect(exported.mission_history).toEqual(before.mission_history);expect(exported.recent_attempts).toEqual(before.attempt_history);expect(exported.completed_missions).toEqual(before.completed_levels);expect(exported.current_expedition.room_splits).toEqual(before.room_splits);expect(before.banked_gold).toBe(30+(before.banked_objective_gold||0));expect(exported.career.gold).toBe(before.banked_gold+(before.past_expeditions?.gold||0));expect(exported.career.gear).toBe(1);expect(exported).not.toHaveProperty('start_key');expect(exported).not.toHaveProperty('id');expect(Number.isNaN(Date.parse(exported.exported_at))).toBe(false);
 expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(before);await expect(page.locator('#rift-export-status')).toHaveText('Record download prepared.');
});
