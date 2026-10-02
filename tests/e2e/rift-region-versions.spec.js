const {test,expect}=require('@playwright/test');
const fs=require('fs');
test('regional versions remain separate, exportable and validated',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();const saved=await(await page.request.get('/api/abyss/rift')).json();
 const a='region-v1:'+'a'.repeat(64),b='region-v1:'+'b'.repeat(64);
 const versions={0:{[a]:{definition:a,best_seconds:100,at_ms:1700000000000},[b]:{definition:b,best_seconds:200,at_ms:1700000000000},'':{best_seconds:1,at_ms:0}}};
 await page.evaluate(({saved,versions,b})=>{saved.run.region_records={0:versions[0][b]};saved.run.region_versions=versions;RiftProtocol.validate(saved,'GET');RiftRecords.update(saved.run);},{saved,versions,b});
 await page.locator('#rift-region-records > summary').click();const first=page.locator('#rift-region-record-list > li').first();await expect(first).toContainText('200.0s');await expect(first).toContainText('most recently completed mission version');await first.locator('summary').click();await expect(first.locator('details')).toContainText('100.0s');await expect(first.locator('details')).toContainText('Legacy');await expect(first.locator('details')).toContainText('1.0s');
 await page.locator('#rift-attempt-history > summary').click();
 const downloadPromise=page.waitForEvent('download');await page.locator('#rift-export-records').click();const download=await downloadPromise;const exported=JSON.parse(fs.readFileSync(await download.path(),'utf8'));expect(exported.region_versions).toEqual(versions);
 const rejected=await page.evaluate(({saved,versions,a,b})=>{const checks=[{0:{[a]:versions[0][b]}},{10:versions[0]},{0:{bad:versions[0][a]}},{0:{[a]:{...versions[0][a],best_seconds:0}}}];return checks.every(value=>{saved.run.region_versions=value;try{RiftProtocol.validate(saved,'GET');return false;}catch(_){return true;}});},{saved,versions,a,b});expect(rejected).toBe(true);
 await page.setViewportSize({width:390,height:1600});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(saved.run);
});
