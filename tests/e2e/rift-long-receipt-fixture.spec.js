const {test,expect}=require('@playwright/test');
for(const width of [1280,390])test('long receipt remains searchable and survives reload at '+width+'px',async({page})=>{
 await page.setViewportSize({width,height:900});
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.goto('/abyss/rift?scenario=long-receipt&subclass=arcanist');
 await expect(page.locator('#rift-start')).toBeEnabled();
 const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const run=await read();expect(run.status).toBe('complete');expect(run.level.id).toBe(100);expect(run.banked_items.length).toBeGreaterThan(1000);expect(run.banked_loot).toHaveLength(run.banked_items.length);
 await page.locator('#rift-receipt > summary').click();
 await expect(page.locator('#rift-receipt-total')).toContainText(run.banked_items.length.toLocaleString('en-US')+' items');
 const groups=new Set(run.banked_items).size;
 await expect(page.locator('#rift-receipt-list > li')).toHaveCount(Math.min(200,groups));
 const first=page.locator('#rift-receipt-list > li > details').first();await first.locator('summary').click();await expect(first.locator('ul > li').first()).toContainText('Mission ');
 await page.locator('#rift-receipt-search').fill('missing-fixture-item-xyz');await expect(page.locator('#rift-receipt-empty')).toBeVisible();
 await page.locator('#rift-receipt-search').fill('');await expect(page.locator('#rift-receipt-list > li')).toHaveCount(Math.min(200,groups));
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();
 const saved=await read();expect(saved.banked_items).toEqual(run.banked_items);expect(saved.banked_loot).toEqual(run.banked_loot);expect(errors).toEqual([]);
});
