const {test,expect}=require('@playwright/test');

test('cosmetic awards require full campaign or ten distinct subclass clears',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const before=(await(await page.request.get('/api/abyss/rift')).json()).run;
 const show=async(count,missions,repeats=1)=>page.evaluate(async({count,missions,repeats})=>{const run=(await(await fetch('/api/abyss/rift')).json()).run;run.completed_levels=Array.from({length:count},(_,i)=>i+1);run.mission_history={};for(let i=1;i<=missions;i++)run.mission_history[i]={attempts:repeats,completions:repeats,last_outcome:'completed',completed_by_class:{vanguard:repeats}};window.RiftCampaignTools.update(run,1);},{count,missions,repeats});
 await show(99,1,100);await expect(page.locator('#rift-campaign-badge')).toBeHidden();await expect(page.locator('#rift-class-badges')).toBeEmpty();
 await show(100,9);await expect(page.locator('#rift-campaign-badge')).toHaveText('✦ Campaign complete · 100 missions');await expect(page.locator('#rift-class-badges')).toBeEmpty();
 await show(100,10);await expect(page.locator('#rift-class-badges')).toContainText('Vanguard mastery · 10 distinct missions');await expect(page.locator('#rift-cosmetic-awards')).toContainText('No gold, gear or combat bonuses');
 await show(100,10);await expect(page.locator('#rift-class-badges .rift-region-badge')).toHaveCount(1);
 const after=(await(await page.request.get('/api/abyss/rift')).json()).run;expect(after).toEqual(before);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
