const {test,expect}=require('@playwright/test');

test('recent damage and healing use separate confirmed deltas and expire by combat time',async({page})=>{
  await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
  await page.evaluate(async()=>{window.recentRun=(await(await fetch('/api/abyss/rift')).json()).run;Object.assign(window.recentRun.stats,{seconds:10,damage_taken:100,healing:50});window.RiftHUD.update(window.recentRun,false,true);});
  const summary=page.locator('#rift-recent-damage');await expect(summary).toHaveText('Last 5 combat seconds: 0 damage taken · 0 healing');
  await page.evaluate(()=>{Object.assign(window.recentRun.stats,{seconds:11,damage_taken:125,healing:58});window.RiftHUD.update(window.recentRun,true);window.RiftHUD.update(window.recentRun,true);});
  await expect(summary).toHaveText('Last 5 combat seconds: 25 damage taken · 8 healing');
  await page.evaluate(()=>{window.recentRun.paused=true;window.RiftHUD.update(window.recentRun,false);});await expect(summary).toContainText('25 damage taken · 8 healing');
  await page.evaluate(()=>{Object.assign(window.recentRun.stats,{seconds:14,damage_taken:135,healing:61});window.RiftHUD.update(window.recentRun,true);});await expect(summary).toContainText('35 damage taken · 11 healing');
  await page.evaluate(()=>{window.recentRun.stats.seconds=16;window.RiftHUD.update(window.recentRun,true);});await expect(summary).toContainText('10 damage taken · 3 healing');
  await page.evaluate(()=>{window.RiftHUD.update(window.recentRun,false,true);});await expect(summary).toContainText('0 damage taken · 0 healing');
  await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
