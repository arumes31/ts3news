const {test,expect}=require('@playwright/test');
test('boss weak-point window shows its bonus and disappears on expiry',async({page})=>{
 await page.goto('/abyss/rift?scenario=boss-windup');await expect(page.locator('#rift-start')).toBeEnabled();
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.evaluate(run=>{document.getElementById('rift-overlay').hidden=true;const boss=run.enemies.find(e=>e.kind==='boss');boss.windup=0;boss.weak_point=.8;window.RiftHUD.update(run,true);},run);
 await expect(page.locator('#rift-boss-attack')).toBeVisible();await expect(page.locator('#rift-boss-attack')).toHaveText('Weak point · +25% damage · 0.8s');
 await page.evaluate(run=>{const boss=run.enemies.find(e=>e.kind==='boss');boss.windup=0;boss.weak_point=0;window.RiftHUD.update(run,true);},run);
 await expect(page.locator('#rift-boss-attack')).toBeHidden();
});
