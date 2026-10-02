const {test,expect}=require('@playwright/test');

test('gold and gear pickups have separate notices without replay announcements',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 await expect(page.locator('#rift-pickup-notices')).toBeHidden();
 await page.evaluate(async()=>{const run=(await(await fetch('/api/abyss/rift')).json()).run;run.drops[0].collected=false;window.RiftLoot.update(run,true);run.drops[0].collected=true;window.RiftLoot.update(run);window.RiftLoot.update(run);});
 await expect(page.locator('#rift-pickup-notices p')).toHaveCount(2);await expect(page.locator('#rift-pickup-notices')).toContainText('Gold picked up: +30');await expect(page.locator('#rift-pickup-notices')).toContainText('Gear picked up:');await expect(page.locator('#rift-pickup-notices')).toContainText('unbanked');
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await page.evaluate(async()=>{const run=(await(await fetch('/api/abyss/rift')).json()).run;run.status='complete';window.RiftLoot.update(run);});await expect(page.locator('#rift-pickup-notices')).toBeHidden();
 await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-pickup-notices')).toBeHidden();
 await page.evaluate(async()=>{const run=(await(await fetch('/api/abyss/rift')).json()).run;run.room=1;window.RiftLoot.update(run);});await expect(page.locator('#rift-pickup-notices')).toBeHidden();
});

test('defeat explains lost floor and bag drops and shows retained rewards',async({page})=>{
 await page.route('**/api/abyss/rift*',async route=>{const response=await route.fetch();const data=await response.json();if(data.run){data.run.status='defeated';data.run.player.hp=0;data.run.gold=0;data.run.drops=[];data.run.banked_gold=120;data.run.banked_items=['Saved Blade'];data.hazard_hit_damage=0;}await route.fulfill({response,json:data});});
 await page.goto('/abyss/rift?scenario=checkpoint');
 await expect(page.locator('#rift-overlay-copy')).toContainText('collected bag items and uncollected floor drops');await expect(page.locator('#rift-overlay-copy')).toContainText('Kept: 120 gold and 1 banked item');await expect(page.locator('#rift-overlay-copy')).toContainText('equipped gear is safe');
 await expect(page.locator('#rift-receipt-total')).toContainText('120 gold · 1 item safely banked');
});
