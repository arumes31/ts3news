const {test,expect}=require('@playwright/test');

test('bestiary filters to confirmed room encounters and preserves records on reload',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const data=await(await page.request.get('/api/abyss/rift')).json(),records=data.run.monster_records;expect(Object.keys(records).length).toBeGreaterThan(0);
 const expected=data.bestiary.filter(unit=>Object.hasOwn(records,unit.art_key));await page.locator('.rift-bestiary > summary').click();await page.locator('#rift-monster-seen').check();await expect(page.locator('#rift-monsters article:visible')).toHaveCount(expected.length);
 const unit=expected[0];await page.getByRole('button',{name:'Inspect '+unit.name,exact:true}).click();await expect(page.locator('#rift-monster-record')).toContainText('First recorded encounter:');await expect(page.locator('#rift-monster-record')).toContainText('Defeats: '+records[unit.art_key].defeats);
 await page.locator('#rift-monster-close').click();await page.locator('#rift-monster-clear').click();await expect(page.locator('#rift-monster-seen')).not.toBeChecked();await expect(page.locator('#rift-monsters article:visible')).toHaveCount(data.bestiary.length);
 await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();expect((await(await page.request.get('/api/abyss/rift')).json()).run.monster_records).toEqual(records);
});

test('unrecorded and practice creatures do not claim historical encounters',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();await page.locator('.rift-bestiary > summary').click();await page.getByRole('button',{name:'Inspect '+data.bestiary[0].name,exact:true}).click();await expect(page.locator('#rift-monster-record')).toContainText('No recorded encounter');
 await page.goto('/abyss/rift?practice=combo');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('.rift-bestiary > summary').click();await expect(page.locator('#rift-monster-seen')).toBeDisabled();await page.getByRole('button',{name:'Inspect '+data.bestiary[0].name,exact:true}).click();await expect(page.locator('#rift-monster-record')).toContainText('Practice does not add encounters');
});
