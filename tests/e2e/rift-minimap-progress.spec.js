const {test,expect}=require('@playwright/test');

test('minimap tier clears advance with confirmed snapshots and survive reload',async({page},testInfo)=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const route=page.locator('#rift-minimap-progress');await expect(route.locator('[data-state=cleared]')).toHaveCount(1);await expect(route.locator('[data-state=ahead]')).toHaveCount(2);
 await expect(route.locator('[aria-current=step]')).toContainText('Tier 1');await expect(page.locator('#rift-minimap .map-floor')).toHaveAttribute('data-cleared','true');
 await page.locator('#rift-auto').uncheck();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.locator('#rift-next').click();
 await expect(route.locator('[aria-current=step]')).toContainText('Tier 2');await expect(route.locator('[data-state=cleared]')).toHaveCount(1);await expect(route.locator('[data-state=current]')).toHaveCount(1);await expect(page.locator('#rift-minimap .map-floor')).toHaveAttribute('data-cleared','false');
 await page.locator('#rift-pause').click();await expect(page.locator('#rift-start')).toBeEnabled();await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();
 await expect(route.locator('[aria-current=step]')).toContainText('Tier 2');await expect(route.locator('[data-state=cleared]')).toHaveCount(1);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await page.locator('#rift-minimap').screenshot({path:testInfo.outputPath('tier-progress-mobile.png')});
});

test('minimap distinguishes defeat, completed and expired clears from practice and previous missions',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint&room=final');await expect(page.locator('#rift-start')).toBeEnabled();
 const route=page.locator('#rift-minimap-progress');await expect(route.locator('[data-state=cleared]')).toHaveCount(3);
 await page.evaluate(async()=>{window.progressRun=(await(await fetch('/api/abyss/rift')).json()).run;});
 await page.evaluate(()=>{const run=window.progressRun;run.status='defeated';run.room_splits=[null,null,null];window.RiftMinimap.update(run);});
 await expect(route.locator('[data-state=cleared]')).toHaveCount(2);await expect(route.locator('[data-state=ended]')).toContainText('Not cleared');
 await page.evaluate(()=>{window.progressRun.status='complete';window.RiftMinimap.update(window.progressRun);});await expect(route.locator('[data-state=cleared]')).toHaveCount(3);
 await page.evaluate(()=>{window.progressRun.status='banked';window.RiftMinimap.update(window.progressRun);});await expect(route.locator('[data-state=cleared]')).toHaveCount(3);
 await page.evaluate(()=>{window.progressRun.status='expired';window.RiftMinimap.update(window.progressRun);});await expect(route.locator('[data-state=cleared]')).toHaveCount(2);await expect(route.locator('[data-state=ended]')).toContainText('Not cleared');
 await page.evaluate(()=>{window.progressRun.status='expired';window.progressRun.room_splits=[1,2,3];window.RiftMinimap.update(window.progressRun);});await expect(route.locator('[data-state=cleared]')).toHaveCount(3);
 await page.evaluate(()=>{const run=window.progressRun;run.room=0;run.status='fighting';run.room_splits=[null,null,null];run.completed_levels=[run.level.id];window.RiftMinimap.update(run);});await expect(route.locator('[data-state=cleared]')).toHaveCount(0);await expect(route.locator('[data-state=ahead]')).toHaveCount(2);
 await page.evaluate(()=>{const run=window.progressRun;run.practice={arena:run.level.rooms[0]};window.RiftMinimap.update(run);});await expect(route).toBeHidden();await expect(page.locator('#rift-minimap .map-floor')).toHaveAttribute('data-cleared','false');
});
