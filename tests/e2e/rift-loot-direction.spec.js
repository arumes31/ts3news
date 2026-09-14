const {test,expect}=require('@playwright/test');

test('nearest drop direction ignores collected and banked loot and follows player position',async({page})=>{
  await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
  const cue=page.locator('#rift-nearest-drop');await expect(cue).toHaveText('No uncollected drops');
  await page.evaluate(async()=>{const run=(await(await fetch('/api/abyss/rift')).json()).run;window.lootDirectionRun=run;run.player.x=400;run.player.y=400;const drop=run.drops[0];run.drops=[{...drop,id:'left',x:100,y:400,collected:false},{...drop,id:'near',x:500,y:300,collected:false},{...drop,id:'collected',x:401,y:400,collected:true},{...drop,id:'banked',x:402,y:400,collected:false,banked:true}];window.RiftLoot.update(run);});
  await expect(cue).toHaveText('Nearest drop: ↗ upper right');
  await page.evaluate(()=>{window.lootDirectionRun.player.x=600;window.RiftLoot.update(window.lootDirectionRun);});await expect(cue).toHaveText('Nearest drop: ↖ upper left');
  await page.evaluate(()=>{window.lootDirectionRun.drops[1].collected=true;window.RiftLoot.update(window.lootDirectionRun);});await expect(cue).toHaveText('Nearest drop: ← left');
  await page.evaluate(()=>{window.lootDirectionRun.drops[0].banked=true;window.RiftLoot.update(window.lootDirectionRun);});await expect(cue).toHaveText('No uncollected drops');
  await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
