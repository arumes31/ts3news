const {test,expect}=require('@playwright/test');

test('protocol accepts skill count maps and rejects malformed per-skill counts',async({page})=>{
  await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
  const outcomes=await page.evaluate(async()=>{const data=await(await fetch('/api/abyss/rift')).json();return [{guard:2},{guard:-1},{guard:1.5},{guard:'2'},[]].map(counts=>{data.run.stats.skill_uses=counts;try{window.RiftProtocol.validate(data,'GET');return true;}catch(_){return false;}});});
  expect(outcomes).toEqual([true,false,false,false,false]);
  const mana=await page.evaluate(async()=>{const data=await(await fetch('/api/abyss/rift')).json();return [{guard:12.5},{guard:-1},{guard:Infinity},{guard:'2'},[]].map(counts=>{data.run.stats.skill_mana=counts;try{window.RiftProtocol.validate(data,'GET');return true;}catch(_){return false;}});});
  expect(mana).toEqual([true,false,false,false,false]);
});

test('skill usage displays confirmed casts and survives reload',async({page})=>{
  await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();
  await page.keyboard.press('1');
  const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
  await expect.poll(async()=>(await read()).stats.skill_uses?.guard||0).toBe(1);await page.keyboard.press('Escape');
  await page.locator('.rift-run-statistics > summary').click();
  const count=page.locator('#rift-statistics dt').filter({hasText:'Casts · Iron Guard (optional)'}).locator('xpath=following-sibling::dd[1]');await expect(count).toHaveText('1');
  await page.reload();await page.locator('.rift-run-statistics > summary').click();await expect(count).toHaveText('1');
  const run=await read();expect(run.stats.skill_mana.guard).toBe(run.build.skills.find(skill=>skill.id==='guard').cost);
  const mana=page.locator('#rift-statistics dt').filter({hasText:'Mana · Iron Guard (optional)'}).locator('xpath=following-sibling::dd[1]');await expect(mana).toHaveText(String(run.stats.skill_mana.guard));
  await page.evaluate(async()=>{const run=(await(await fetch('/api/abyss/rift')).json()).run;delete run.stats.skill_uses;delete run.stats.skill_mana;window.RiftHUD.update(run,false,true);});
  await expect(page.locator('#rift-statistics')).toContainText('Earlier casts without per-skill records');
  await expect(page.locator('#rift-statistics')).toContainText('Earlier mana without per-skill records');
  await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});

test('hit and healing breakdown validates records and separates unattributed healing',async({page})=>{
  await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
  const validation=await page.evaluate(async()=>{
    const data=await(await fetch('/api/abyss/rift')).json();
    const result={};
    for(const key of ['skill_hits','skill_healing','skill_barrier']){
      result[key]=[{guard:2},{guard:1.5},{guard:-1},{guard:NaN},{guard:'2'},[]].map(value=>{
        data.run.stats[key]=value;try{window.RiftProtocol.validate(data,'GET');return true;}catch(_){return false;}
      });delete data.run.stats[key];
    }
    data.run.stats.skill_hits={guard:3};data.run.stats.skill_healing={guard:18};data.run.stats.healing=25;
    data.run.stats.skill_barrier={guard:30};data.run.stats.barrier_blocked=40;
    window.RiftHUD.update(data.run,false,true);return result;
  });
  expect(validation.skill_hits).toEqual([true,false,false,false,false,false]);
  expect(validation.skill_healing).toEqual([true,true,false,false,false,false]);
  expect(validation.skill_barrier).toEqual([true,true,false,false,false,false]);
  await page.locator('.rift-run-statistics > summary').click();
  const value=label=>page.locator('#rift-statistics dt').filter({hasText:label}).locator('xpath=following-sibling::dd[1]');
  await expect(value('Hits · Iron Guard (optional)')).toHaveText('3');
  await expect(value('Healing · Iron Guard (optional)')).toHaveText('18');
  await expect(value('Healing without per-skill records')).toHaveText('7');
  await expect(value('Absorbed · Iron Guard (optional)')).toHaveText('30');
  await expect(value('Absorption without per-skill records')).toHaveText('10');
  await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
