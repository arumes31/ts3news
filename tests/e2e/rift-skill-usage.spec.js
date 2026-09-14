const {test,expect}=require('@playwright/test');

test('protocol accepts skill count maps and rejects malformed per-skill counts',async({page})=>{
  await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
  const outcomes=await page.evaluate(async()=>{const data=await(await fetch('/api/abyss/rift')).json();return [{guard:2},{guard:-1},{guard:1.5},{guard:'2'},[]].map(counts=>{data.run.stats.skill_uses=counts;try{window.RiftProtocol.validate(data,'GET');return true;}catch(_){return false;}});});
  expect(outcomes).toEqual([true,false,false,false,false]);
});

test('skill usage displays confirmed casts and survives reload',async({page})=>{
  await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();
  await page.keyboard.press('1');
  const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
  await expect.poll(async()=>(await read()).stats.skill_uses?.guard||0).toBe(1);await page.keyboard.press('Escape');
  await page.locator('.rift-run-statistics > summary').click();
  const count=page.locator('#rift-statistics dt').filter({hasText:'Casts · Iron Guard (optional)'}).locator('xpath=following-sibling::dd[1]');await expect(count).toHaveText('1');
  await page.reload();await page.locator('.rift-run-statistics > summary').click();await expect(count).toHaveText('1');
  await page.evaluate(async()=>{const run=(await(await fetch('/api/abyss/rift')).json()).run;delete run.stats.skill_uses;window.RiftHUD.update(run,false,true);});
  await expect(page.locator('#rift-statistics')).toContainText('Earlier casts without per-skill records');
  await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
