const {test,expect}=require('@playwright/test');
test('regional landmark names identify current room, survive reload and fit mobile',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const {run}=await(await page.request.get('/api/abyss/rift')).json();const name=run.level.rooms[run.room].name;
 expect(name).toContain('Ivy Arch');const title=page.locator('#rift-minimap p strong');await expect(title).toHaveText(name);
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(title).toHaveText(name);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await page.locator('#rift-minimap').screenshot({path:'test-results/landmark-minimap-mobile.png'});
 await page.evaluate(run=>{run.level.rooms[run.room].name='Legacy courtyard';RiftMinimap.update(run);},run);await expect(title).toHaveText('Legacy courtyard');
 await page.evaluate(run=>{run.level.rooms[run.room].name='<img src=x onerror=alert(1)>';RiftMinimap.update(run);},run);await expect(title).toHaveText('<img src=x onerror=alert(1)>');await expect(title.locator('img')).toHaveCount(0);
});
