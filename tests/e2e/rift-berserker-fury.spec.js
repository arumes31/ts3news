const {test,expect}=require('@playwright/test');
test('Berserker fury is readable at threshold and clears on healing or class change',async({page})=>{
 await page.goto('/abyss/rift?subclass=berserker&scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-class-primer')).toContainText('Fury grants +15% damage');const {run}=await(await page.request.get('/api/abyss/rift')).json();
 const node=page.locator('#rift-berserker-fury');
 for(const hp of [30.01,30,20,31,0]){
  await page.evaluate(({run,hp})=>{run.player.max_hp=100;run.player.hp=hp;RiftHUD.update(run,false);},{run,hp});await expect(node).toBeVisible();await expect(node).toHaveAttribute('data-active',String(hp>0&&hp<=30));await expect(node).toContainText(hp>0&&hp<=30?'Fury active':'Fury inactive');
 }
 await page.evaluate(run=>{run.player.max_hp=100;run.player.hp=30;RiftHUD.update(run,false);},run);await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await node.scrollIntoViewIfNeeded();await node.screenshot({path:'test-results/berserker-fury-mobile.png'});
 await page.evaluate(run=>{run.build.class='vanguard';RiftHUD.update(run,false);},run);await expect(node).toBeHidden();
});
