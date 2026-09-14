const {test,expect}=require('@playwright/test');

test('campaign collapsed-on-load preference persists without hiding its summary or changing saves',async({page})=>{
 await page.addInitScript(()=>{if(!sessionStorage.getItem('collapseSeed')){localStorage.setItem('riftCampaignView',JSON.stringify({startCollapsed:'bad',favorites:[4]}));sessionStorage.setItem('collapseSeed','1');}});
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();const campaign=page.locator('#rift-campaign');await expect(campaign).toHaveAttribute('open','');
 await page.locator('.rift-settings > summary').click();const preference=page.locator('#rift-campaign-start-collapsed');await expect(preference).not.toBeChecked();await preference.check();await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(campaign).not.toHaveAttribute('open','');await expect(page.locator('#rift-campaign > summary')).toBeVisible();
 await page.locator('#rift-campaign > summary').focus();await page.keyboard.press('Enter');await expect(campaign).toHaveAttribute('open','');await page.locator('#rift-mission-search').fill('forest');
 await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(campaign).not.toHaveAttribute('open','');await page.locator('.rift-settings > summary').click();await expect(preference).toBeChecked();await preference.uncheck();await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(campaign).toHaveAttribute('open','');
 expect(await page.evaluate(()=>JSON.parse(localStorage.getItem('riftCampaignView')).favorites)).toEqual([4]);expect((await(await page.request.get('/api/abyss/rift')).json()).run).toBeNull();
});
