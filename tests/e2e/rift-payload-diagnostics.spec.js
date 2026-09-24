const {test,expect}=require('@playwright/test');
const fs=require('node:fs');const path=require('node:path');
// Exercise the current source without rebuilding the large Go fixture binary.
async function currentClient(page){await page.route('**/static/rift.js*',route=>route.fulfill({contentType:'application/javascript',body:fs.readFileSync(path.join(__dirname,'../../internal/bot/webassets/rift.js'),'utf8')}));}

test('payload measurement is opt-in',async({page})=>{
 await currentClient(page);await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 expect(await page.evaluate(()=>window.RiftPayloadDiagnostics)).toBeUndefined();
});
test('payload diagnostics measure UTF-8 and keep a bounded metadata history',async({page})=>{
 await currentClient(page);const responses=[];
 await page.route('**/api/abyss/rift*',async route=>{
  const response=await route.fetch(),data=await response.json();data.diagnostic_probe='魔法🔥';const body=JSON.stringify(data),request=route.request().postDataJSON();
  responses.push({action:request?.kind||'load',responseBytes:Buffer.byteLength(body),runBytes:data.run?Buffer.byteLength(JSON.stringify(data.run)):0});
  await route.fulfill({response,body});
 });
 await page.goto('/abyss/rift?practice=movement&riftPayloadDebug=1');await expect(page.locator('#rift-start')).toBeEnabled();
 const first=await page.evaluate(()=>window.RiftPayloadDiagnostics);expect(first).toBeTruthy();expect(first.count).toBe(1);expect(first.samples[0]).toMatchObject(responses[0]);
 await page.locator('#rift-start').click();
 await expect.poll(()=>page.evaluate(()=>window.RiftPayloadDiagnostics.count),{timeout:15000}).toBeGreaterThan(40);
 await page.keyboard.press('Escape');
 const stats=await page.evaluate(()=>window.RiftPayloadDiagnostics);
 expect(stats.samples).toHaveLength(32);expect(stats.maxResponseBytes).toBeGreaterThanOrEqual(Math.max(...stats.samples.map(s=>s.responseBytes)));
 expect(stats.totalResponseBytes).toBeGreaterThan(stats.maxResponseBytes);
 for(const sample of stats.samples){expect(responses.some(r=>r.action===sample.action&&r.responseBytes===sample.responseBytes&&r.runBytes===sample.runBytes)).toBe(true);expect(Object.keys(sample).sort()).toEqual(['action','method','requestBytes','responseBytes','runBytes']);}
 expect(stats.samples.some(s=>s.action==='step')).toBe(true);
});


test('diagnostic parsing preserves recovery from a truncated response',async({page})=>{
 await currentClient(page);let first=true;
 await page.route('**/api/abyss/rift*',async route=>{
  if(first&&route.request().method()==='GET'){first=false;await route.fulfill({status:200,contentType:'application/json',body:'{"ok":true,"run":'});return;}
  await route.continue();
 });
 await page.goto('/abyss/rift?riftPayloadDebug=1');await expect(page.locator('#rift-start')).toHaveText('Retry loading');
 expect(await page.evaluate(()=>window.RiftPayloadDiagnostics.count)).toBe(0);
 await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toBeEnabled();
 await expect.poll(()=>page.evaluate(()=>window.RiftPayloadDiagnostics.count)).toBe(1);
});
