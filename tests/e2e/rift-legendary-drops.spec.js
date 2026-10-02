const {test,expect}=require('@playwright/test');

test('legendary floor drops retain their marker without sparkle and leave the count after pickup',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.locator('.rift-settings > summary').click();await page.locator('#rift-loot-sparkle').uncheck();
 await page.emulateMedia({reducedMotion:'reduce'});
 await page.evaluate(async()=>{await window.RiftRenderer.ready;const data=await(await fetch('/api/abyss/rift')).json(),run=data.run;const rare=data.rarities.find(r=>r.legendary);run.drops[0].gear.Rarity=rare.value;run.drops[0].collected=false;run.drops[0].banked=false;run.drops[0].x=run.player.x+100;window.legendaryRun=run;window.legendaryDrawn=false;const ctx=document.querySelector('#rift-canvas').getContext('2d'),original=ctx.fillText;ctx.fillText=function(text,...args){if(text==='◆ LEGENDARY')window.legendaryDrawn=true;return original.call(this,text,...args);};window.RiftLoot.update(run,true);window.RiftRenderer.snapshot(run,true);});
 await expect(page.locator('#rift-legendary-drops')).toHaveText('◆ 1 legendary drop on the battlefield');
 await expect.poll(()=>page.evaluate(()=>window.legendaryDrawn)).toBe(true);
 await page.evaluate(()=>{window.legendaryRun.drops[0].collected=true;window.RiftLoot.update(window.legendaryRun);window.RiftRenderer.snapshot(window.legendaryRun,true);});
 await expect(page.locator('#rift-legendary-drops')).toHaveText('No legendary drops on the battlefield');
 await page.evaluate(()=>{const drop=window.legendaryRun.drops[0];drop.collected=false;drop.banked=true;window.RiftLoot.update(window.legendaryRun);});await expect(page.locator('#rift-legendary-drops')).toHaveText('No legendary drops on the battlefield');
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
