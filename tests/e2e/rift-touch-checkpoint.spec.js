const {test,expect}=require('@playwright/test');
test.use({hasTouch:true,isMobile:true,viewport:{width:390,height:844}});
test('touch-only checkpoint banking and receipt flow',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.locator('#rift-auto').tap();await expect(page.locator('#rift-auto')).not.toBeChecked();
 await page.locator('#rift-start').tap();await expect(page.locator('#rift-room-actions')).toBeVisible();
 await expect(page.locator('#rift-checkpoint-total')).toHaveText('30 gold · 1 item ready to bank');
 await page.locator('#rift-exit').tap();await expect(page.locator('#rift-banked')).toHaveText('30 gold · 1 item');
 await page.locator('#rift-receipt > summary').tap();await expect(page.locator('#rift-receipt-list > li')).toHaveCount(1);
 await page.locator('#rift-receipt-list > li > details > summary').tap();await expect(page.locator('#rift-receipt-list > li > details')).toHaveAttribute('open','');
 const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const saved=await read();expect(saved.status).toBe('banked');expect(saved.banked_gold).toBe(30);expect(saved.banked_items).toHaveLength(1);expect(saved.drops.every(drop=>drop.banked)).toBe(true);
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-banked')).toHaveText('30 gold · 1 item');expect((await read()).banked_items).toEqual(saved.banked_items);
});
