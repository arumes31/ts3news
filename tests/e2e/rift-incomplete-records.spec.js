const {test,expect}=require('@playwright/test');
test('incomplete records stay unavailable while genuine zero measurements remain visible',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const saved=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.evaluate(run=>{run.region_records={0:{best_seconds:0},1:{}};run.room_splits=[null,0];run.attempt_history=[{mission:1,difficulty:'Wayfarer',outcome:'exited',seconds:4,hp:40,max_hp:100,hits:2},{mission:1,difficulty:'Wayfarer',outcome:'defeated',hp:0,max_hp:100,hits:0,splits:[null,0]}];RiftRecords.update(run);},saved);
 const rows=page.locator('#rift-attempt-list > li');await expect(rows.first()).toContainText('unavailable combat');await expect(rows.first()).toContainText('End HP 0.0/100.0');await expect(rows.first()).toContainText('Damaging hits 0');await expect(rows.first()).toContainText('Tier 1 unavailable · Tier 2 0.0s · Tier 3 unavailable');
 await expect(page.locator('#rift-attempt-comparison')).toContainText('combat time comparison unavailable; end HP -40.0; damaging hits -2.0');await expect(page.locator('#rift-region-record-list > li').first()).toContainText('No complete regional run recorded');
 await page.evaluate(run=>{run.attempt_history=[{mission:1,outcome:'exited',seconds:0},{mission:1,outcome:'expired',seconds:0}];RiftRecords.update(run);},saved);
 await expect(rows.first()).toContainText('0.0s combat · End HP unavailable · Damaging hits unavailable');await expect(page.locator('#rift-attempt-comparison')).toContainText('unknown difficulty');
 expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(saved);
});
