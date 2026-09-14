const {test,expect}=require('@playwright/test');

test('receipt banking time persists across reload and identifies older receipts',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await page.locator('#rift-auto').uncheck();await page.locator('#rift-start').click();await page.locator('#rift-exit').click();
 await expect(page.locator('#rift-banked-at')).toContainText('Last banked:');
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;expect(run.banked_at_ms).toBeGreaterThan(0);
 const expected=await page.evaluate(ms=>'Last banked: '+new Date(ms).toLocaleString(),run.banked_at_ms);await expect(page.locator('#rift-banked-at')).toHaveText(expected);
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-banked-at')).toHaveText(expected);
 const restored=(await(await page.request.get('/api/abyss/rift')).json()).run;expect(restored.banked_at_ms).toBe(run.banked_at_ms);
 await page.evaluate(async()=>{const run=(await(await fetch('/api/abyss/rift')).json()).run;delete run.banked_at_ms;window.RiftLoot.update(run,true);});await expect(page.locator('#rift-banked-at')).toHaveText('Banking time unavailable for this older receipt.');
});
