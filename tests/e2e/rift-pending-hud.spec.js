const {test,expect}=require('@playwright/test');

test('pending HUD counts collected and floor loot, excludes banked drops, and clears on banking',async({page})=>{
  await page.goto('/abyss/rift?scenario=checkpoint');await page.locator('#rift-auto').uncheck();
  await expect(page.locator('#rift-pending-hud')).toHaveText('Unbanked: 30 gold · 1 gear');
  await page.evaluate(async()=>{const run=(await(await fetch('/api/abyss/rift')).json()).run;run.drops.push({...run.drops[0],id:'floor',gold:45,collected:false},{...run.drops[0],id:'banked',gold:999,banked:true});window.RiftLoot.update(run);});
  await expect(page.locator('#rift-pending-hud')).toHaveText('Unbanked: 75 gold · 2 gear · 1 uncollected');
  await expect(page.locator('#rift-loot-count')).toHaveText('1 item pending');
  await page.locator('#rift-start').click();await expect(page.locator('#rift-pending-hud')).toHaveText('Unbanked: 30 gold · 1 gear');
  await page.locator('#rift-exit').click();await expect(page.locator('#rift-banked')).toHaveText('30 gold · 1 item');await expect(page.locator('#rift-pending-hud')).toHaveText('Unbanked: 0 gold · 0 gear');
  await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
