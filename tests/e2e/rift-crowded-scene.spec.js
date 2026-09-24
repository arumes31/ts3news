const {test,expect}=require('@playwright/test');
test('seeded crowded scene renders with bounded frame diagnostics',async({page})=>{
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 const url='/abyss/rift?scenario=visual&seed=crowded-v1&crowd=120&riftFrameDebug=1';
 await page.goto(url);await expect(page.locator('#rift-start')).toBeEnabled();
 const first=(await(await page.request.get('/api/abyss/rift')).json()).run;
 expect(first.enemies).toHaveLength(120);expect(first.paused).toBe(true);
 expect(new Set(first.enemies.map(actor=>actor.id)).size).toBe(120);
 await expect.poll(()=>page.evaluate(()=>window.RiftRenderer.frameDiagnostics.count)).toBeGreaterThan(10);
 await expect(page.locator('#rift-frame-diagnostics')).toContainText('Render');
 await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();
 const second=(await(await page.request.get('/api/abyss/rift')).json()).run;
 expect(second).toEqual(first);expect(errors).toEqual([]);
});
