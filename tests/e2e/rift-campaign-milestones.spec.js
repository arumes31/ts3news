const {test,expect}=require('@playwright/test');

test('campaign percentage follows confirmed completion and survives reload',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint&room=final');await page.locator('#rift-auto').uncheck();await expect(page.locator('#rift-progress')).toContainText('0%');await expect(page.locator('#rift-region-badge')).toBeHidden();
 await page.locator('#rift-start').click();await page.locator('#rift-next').click();await expect(page.locator('#rift-progress')).toContainText('1/100 completed · 1%');
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-progress')).toContainText('1%');await expect(page.locator('#rift-regional-milestone')).toContainText('1/5');
});

test('regional milestones handle halfway, full completion, duplicates and the cosmetic badge',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const show=async(ids,selected=1)=>page.evaluate(async({ids,selected})=>{const run=(await(await fetch('/api/abyss/rift')).json()).run;run.completed_levels=ids;window.RiftCampaignTools.update(run,selected);},{ids,selected});
 await show([1,2,3,4,4,0,101]);await expect(page.locator('#rift-progress')).toContainText('4/100 completed · 4%');await expect(page.locator('#rift-regional-milestone')).toContainText('4/5 · 1 more mission');
 await show([1,2,3,4,5]);await expect(page.locator('#rift-regional-milestone')).toContainText('5/10 · 5 more missions');await expect(page.locator('#rift-region-badge')).toBeHidden();
 await show(Array.from({length:10},(_,i)=>i+1));await expect(page.locator('#rift-regional-milestone')).toContainText('Region complete · 10/10');await expect(page.locator('#rift-region-badge')).toBeVisible();await expect(page.locator('#rift-region-badge')).toContainText('First region complete');
 await show(Array.from({length:10},(_,i)=>i+1),11);await expect(page.locator('#rift-regional-milestone')).toContainText('0/5');await expect(page.locator('#rift-region-badge')).toBeVisible();
 await show(Array.from({length:100},(_,i)=>i+1));await expect(page.locator('#rift-progress')).toContainText('100/100 completed · 100%');
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
