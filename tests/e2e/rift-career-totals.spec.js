const {test,expect}=require('@playwright/test');
const value=(page,label)=>page.locator('#rift-career-statistics dt').filter({hasText:new RegExp('^'+label+'$')}).locator('xpath=following-sibling::dd[1]');

test('career rewards include confirmed banking and survive reload and replay',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint&room=final');await page.locator('#rift-auto').uncheck();
 await expect(value(page,'Gold banked')).toHaveText('0');await expect(value(page,'Gear pieces banked')).toHaveText('0');
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.locator('#rift-next').click();
 await expect(value(page,'Gold banked')).toHaveText('70');await expect(value(page,'Gear pieces banked')).toHaveText('1');
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(value(page,'Gold banked')).toHaveText('70');
 await page.locator('#rift-replay').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;expect(run.banked_gold).toBe(0);expect(run.banked_items).toEqual([]);expect(run.past_expeditions.gold).toBe(70);expect(run.past_expeditions.gear).toBe(1);
 await page.locator('.rift-run-statistics > summary').click();await expect(value(page,'Gold banked')).toHaveText('70');await expect(value(page,'Gear pieces banked')).toHaveText('1');
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});

test('career display sums recorded expeditions and validates saved totals',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const validation=await page.evaluate(async()=>{
  const data=await(await fetch('/api/abyss/rift')).json(),run=data.run;
  const totals={enemies:100,bosses:10,treasure_goblins:3,gold:200,gear:7};
  const results=[totals,{...totals,enemies:-1},{...totals,bosses:1.5},{...totals,gold:'200'},{},[],undefined].map(past=>{run.past_expeditions=past;try{window.RiftProtocol.validate(data,'GET');return true;}catch(_){return false;}});
  run.past_expeditions=totals;Object.assign(run.stats,{kills:4,bosses:1,treasure_goblins:2});run.banked_gold=30;run.banked_items=['Sword','Sword'];window.RiftHUD.update(run,false,true);return results;
 });
 expect(validation).toEqual([true,false,false,false,false,false,true]);
 for(const [label,count] of [['Enemies defeated','104'],['Bosses defeated','11'],['Treasure goblins defeated','5'],['Gold banked','230'],['Gear pieces banked','9']])await expect(value(page,label)).toHaveText(count);
 await page.locator('.rift-run-statistics > summary').click();await expect(page.locator('.rift-run-statistics')).toContainText('Older expeditions without saved records are not included');await expect(page.locator('.rift-run-statistics')).toContainText('Mission completion, personal bests, subclass clears and career totals survive economy resets');await expect(page.locator('.rift-run-statistics')).toContainText('A reset expires the current expedition and removes its unbanked loot');
});
