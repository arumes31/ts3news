const {test,expect}=require('@playwright/test');

test('receipt rarity filtering preserves totals and explicitly excludes unknown rarity',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.evaluate(async()=>{const data=await(await fetch('/api/abyss/rift')).json(),run=data.run,rare=data.rarities.find(r=>r.rare_or_better),common=data.rarities.find(r=>!r.rare_or_better);run.banked_items=['Plain Sword','Rare Sword','Rare Sword','Older Ring'];run.banked_gold=120;run.banked_loot=[{name:'Plain Sword',rarity:common.value},{name:'Rare Sword',rarity:rare.value},{name:'Rare Sword',rarity:rare.value}];window.RiftLoot.update(run,true);});
 await page.locator('#rift-receipt > summary').click();await page.locator('#rift-receipt-rarity').selectOption('rare');
 await expect(page.locator('#rift-receipt-list li')).toHaveCount(1);await expect(page.locator('#rift-receipt-list')).toHaveText('Rare Sword × 2');await expect(page.locator('#rift-receipt-unknown')).toContainText('1 item without recognized rarity data');await expect(page.locator('#rift-receipt-total')).toHaveText('120 gold · 4 items safely banked');
 await page.locator('#rift-receipt-search').fill('ring');await expect(page.locator('#rift-receipt-empty')).toHaveText('No banked items match these filters.');
 await page.locator('#rift-receipt-rarity').selectOption('all');await expect(page.locator('#rift-receipt-list')).toHaveText('Older Ring');await expect(page.locator('#rift-receipt-unknown')).toBeHidden();
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
