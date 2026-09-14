const {test,expect}=require('@playwright/test');

test('character metadata loads while critical artwork is still pending',async({page})=>{
  let release;const gate=new Promise(resolve=>release=resolve);let fetched=false;let mutations=0;
  page.on('request',request=>{if(request.url().includes('/api/abyss/rift')&&request.method()==='POST')mutations++;});
  await page.route('**/static/rift_regions.png*',async route=>{await gate;await route.continue();});
  page.on('response',response=>{if(response.url().endsWith('/api/abyss/rift')&&response.request().method()==='GET')fetched=true;});
  try{
    await page.goto('/abyss/rift',{waitUntil:'domcontentloaded'});
    await expect.poll(()=>fetched,{timeout:1500}).toBe(true);await expect(page.locator('#rift-start')).toBeDisabled();expect((await(await page.request.get('/api/abyss/rift')).json()).run).toBeNull();expect(mutations).toBe(0);
  }finally{release();}
  await expect(page.locator('#rift-start')).toBeEnabled();
});

test('closed bestiary defers cards and repeated opening preserves the live roster',async({page})=>{
  await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
  const catalog=(await(await page.request.get('/api/abyss/rift')).json()).bestiary;
  await expect(page.locator('#rift-monster-count')).toHaveText(catalog.length+' monsters');await expect(page.locator('#rift-monsters article')).toHaveCount(0);
  await page.locator('.rift-bestiary > summary').click();await expect(page.locator('#rift-monsters article')).toHaveCount(catalog.length);
  await page.locator('#rift-monster-search').fill('Chronos');await expect(page.locator('#rift-monsters article:visible')).toHaveCount(1);
  await page.locator('.rift-bestiary > summary').click();await page.locator('.rift-bestiary > summary').click();await expect(page.locator('#rift-monsters article')).toHaveCount(catalog.length);await expect(page.locator('#rift-monsters article:visible')).toHaveCount(1);
});

test('unchanged combat identity text does not mutate on every confirmed snapshot',async({page})=>{
  await page.goto('/abyss/rift');await page.locator('#rift-start').click();
  let snapshots=0;page.on('response',response=>{if(response.request().postDataJSON()?.kind==='step')snapshots++;});
  await page.evaluate(()=>{window.identityMutations=0;const observer=new MutationObserver(records=>window.identityMutations+=records.length);for(const id of ['rift-name','rift-class','rift-style','rift-room'])observer.observe(document.getElementById(id),{childList:true});});
  await expect.poll(()=>snapshots).toBeGreaterThanOrEqual(5);expect(await page.evaluate(()=>window.identityMutations)).toBe(0);await page.keyboard.press('Escape');
});
