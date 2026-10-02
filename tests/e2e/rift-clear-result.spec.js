const {test,expect}=require('@playwright/test');

test('first clear reports actual records, survives reload and appears on its mission card',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint&room=final&condition=wounded');await page.locator('#rift-auto').uncheck();await expect(page.locator('#rift-clear-result')).toBeHidden();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.locator('#rift-next').click();
 const result=page.locator('#rift-clear-result');await expect(result).toContainText('Mission 1 · First clear!');await expect(result).toContainText('finish HP 170.0/340.0');await expect(result).toContainText('fewest damaging hits 0');
 await expect(page.locator('#rift-history-1')).toContainText('Most HP at finish 170.0/340.0');await expect(page.locator('#rift-history-1')).toContainText('Fewest damaging hits 0');
 const saved=await result.textContent();await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(result).toHaveText(saved);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});

test('repeat-clear presentation distinguishes ties and improved records and validates results',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const validation=await page.evaluate(async()=>{const data=await(await fetch('/api/abyss/rift')).json();const base={mission:1,first:false,records:[]};const result=[base,{...base,mission:101},{...base,first:'false'},{...base,records:['unknown']},undefined].map(clear=>{data.run.last_clear=clear;try{window.RiftProtocol.validate(data,'GET');return true;}catch(_){return false;}});data.run.last_clear=base;window.RiftHUD.update(data.run,false,true);return result;});
 expect(validation).toEqual([true,false,false,false,true]);await expect(page.locator('#rift-clear-result')).toHaveText('Mission 1 · Repeat clear. No personal records improved.');
 await page.evaluate(async()=>{const run=(await(await fetch('/api/abyss/rift')).json()).run;run.last_clear={mission:1,first:false,records:['time']};run.mission_history[1].best_seconds=12.5;window.RiftHUD.update(run,false,true);});await expect(page.locator('#rift-clear-result')).toHaveText('Mission 1 · Repeat clear. New personal records: clear time 12.5s');
});
