const {test,expect}=require('@playwright/test');
for(const mode of ['silent','reduced'])test(mode+' mode completes three tiers and banks the expedition', async ({ page }) => {
  test.setTimeout(180_000);
  const errors=[];page.on('pageerror',error=>errors.push(error.message));
  await page.emulateMedia({reducedMotion:mode==='reduced'?'reduce':'no-preference'});
  if(mode==='silent')await page.addInitScript(()=>localStorage.setItem('riftAudio:muted','true'));
  await page.goto('/abyss/rift?subclass=bloodblade');
  await page.locator('#rift-auto').uncheck();
  await page.locator('#rift-start').click();
  const held=new Set();
  async function controls(wanted){for(const key of [...held])if(!wanted.has(key)){await page.keyboard.up(key);held.delete(key);}for(const key of wanted)if(!held.has(key)){await page.keyboard.down(key);held.add(key);}}
  const started=Date.now();let complete=false,expectedFightGold=0,expectedItems=0;const visited=new Set();
  while(Date.now()-started<145_000){
    const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
    visited.add(run.room);
    const preferences=await page.evaluate(()=>({muted:RiftAudio.muted,played:RiftAudio.played,reduced:RiftRenderer.reduced}));
    if(mode==='silent'){expect(preferences.muted).toBe(true);expect(preferences.played).toBe(0);}else expect(preferences.reduced).toBe(true);
    expect(run.status,'expedition should remain survivable').not.toBe('defeated');
    if(run.status==='complete'){complete=true;expect(expectedFightGold).toBeGreaterThan(0);expect(expectedItems).toBeGreaterThan(0);expect(run.banked_gold).toBe(expectedFightGold+(run.banked_objective_gold||0));expect(run.banked_items.length).toBe(expectedItems);for(const drop of run.drops.filter(d=>d.gear))expect(drop.gear.found_boss).toContain(run.level.name);break;}
    if(run.status==='cleared'){
      const unbanked=run.drops.filter(drop=>!drop.banked);expectedFightGold+=unbanked.reduce((sum,drop)=>sum+drop.gold,0);expectedItems+=unbanked.filter(drop=>drop.gear).length;
      await controls(new Set());
      await expect(page.locator('#rift-next')).toBeVisible();
      if(run.room===2)await page.screenshot({path:'test-results/rift-'+mode+'-boss-cleared.png'});
      await page.locator('#rift-next').click();
      await expect(page.locator('#rift-next')).toBeHidden();continue;
    }
    const p=run.player,target=run.enemies.filter(e=>e.hp>0).sort((a,b)=>Math.abs(a.x-p.x)-Math.abs(b.x-p.x))[0];
    const dx=target.x-p.x,dy=target.y-p.y,wanted=new Set(['Space']);
    if(Math.abs(dy)>10)wanted.add(dy>0?'s':'w');
    if(Math.abs(dx)>60 || Math.sign(dx)!==p.facing)wanted.add(dx>0?'d':'a');
    if(Math.abs(dx)<120&&Math.abs(dy)<30){
      const [builder,finisher]=run.build.signatures;
      if(run.resource>0&&!(run.skill_timers[finisher.id]>0)&&p.mana>=finisher.cost)wanted.add('e');
      else if(!(run.skill_timers[builder.id]>0)&&p.mana>=builder.cost)wanted.add('q');
      else wanted.add('j');
    }
    await controls(wanted);
    await page.waitForTimeout(120);
  }
  await controls(new Set());
  expect(complete).toBe(true);expect([...visited].sort()).toEqual([0,1,2]);expect(errors).toEqual([]);
  const finished=(await(await page.request.get('/api/abyss/rift')).json()).run;
  await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));
  await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();
  const restored=(await(await page.request.get('/api/abyss/rift')).json()).run;
  expect(restored.banked_gold).toBe(finished.banked_gold);expect(restored.banked_items).toEqual(finished.banked_items);
  await expect(page.locator('#rift-overlay-title')).toHaveText('Returned from the ruins.');
});
