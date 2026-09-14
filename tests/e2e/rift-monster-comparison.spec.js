const {test,expect}=require('@playwright/test');

test('monster comparison uses shared stats and handles empty and duplicate choices',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json(),a=data.bestiary.find(unit=>unit.kind==='boss'),b=data.bestiary.find(unit=>unit.kind==='archer');
 await page.locator('.rift-bestiary > summary').click();await page.locator('#rift-compare > summary').focus();await page.keyboard.press('Enter');
 await expect(page.locator('#rift-monster-comparison')).toBeHidden();await page.locator('#rift-compare-left').selectOption(a.art_key);await page.locator('#rift-compare-right').selectOption(a.art_key);await expect(page.locator('#rift-compare-status')).toHaveText('Choose two different creatures.');
 await page.locator('#rift-compare-right').selectOption(b.art_key);const table=page.locator('#rift-monster-comparison');await expect(table).toBeVisible();await expect(table.locator('caption')).toHaveText(a.name+' compared with '+b.name);
 const health=table.locator('tbody tr').filter({has:page.getByRole('rowheader',{name:'Health',exact:true})});for(const value of [a.max_hp,b.max_hp])await expect(health).toContainText(new Intl.NumberFormat('en-US',{maximumFractionDigits:1}).format(value));
 await expect(table).toContainText('Attack windup');await expect(table.locator('.rift-compare-different').first()).toBeVisible();
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await page.locator('#rift-compare-clear').click();await expect(table).toBeHidden();await expect(page.locator('#rift-compare-left')).toBeFocused();expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(data.run);
});
