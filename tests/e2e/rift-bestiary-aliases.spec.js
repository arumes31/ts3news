const {test,expect}=require('@playwright/test');
test('canonical monster aliases search without replacing identity or bypassing filters',async({page})=>{
 let canonical,artKey;
 await page.route(/\/api\/abyss\/rift(?:\/metadata)?(?:\?.*)?$/,async route=>{
  const response=await route.fetch(),data=await response.json();
  if(Array.isArray(data.bestiary)){
   canonical=data.bestiary[0].name;artKey=data.bestiary[0].art_key;
   data.bestiary[0].aliases=['Écho Sentinel Test','<img src=x onerror=alert(1)> AliasProbe',null,45];
   data.bestiary[1].aliases='IgnoredMalformedAlias';
  }
  await route.fulfill({response,json:data});
 });
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.locator('details').filter({has:page.locator('#rift-monsters')}).locator(':scope > summary').click();
 const search=page.locator('#rift-monster-search'),cards=page.locator('#rift-monsters > article:visible');
 const total=await cards.count();expect(total).toBeGreaterThan(2);
 for(const query of ['ECHO SENTINEL TEST','echó-sentinel-test','AliasProbe']){
  await search.fill(query);await expect(cards).toHaveCount(1);await expect(cards.locator('strong')).toHaveText(canonical);await expect(cards).toHaveAttribute('data-art-key',artKey);
 }
 await expect(page.locator('#rift-monsters img')).toHaveCount(0);
 await cards.getByRole('button',{name:'Inspect '+canonical,exact:true}).click();await expect(page.locator('#rift-monster-detail')).toContainText(canonical);
 const family=await cards.getAttribute('data-family');
 const other=await page.locator('#rift-monster-family option').evaluateAll((options,current)=>options.map(o=>o.value).find(v=>v&&v!==current),family);
 await page.locator('#rift-monster-family').selectOption(other);await expect(cards).toHaveCount(0);
 await page.locator('#rift-monster-clear').click();await expect(cards).toHaveCount(total);
 await search.fill('IgnoredMalformedAlias');await expect(cards).toHaveCount(0);
 await search.fill(canonical);expect(await cards.count()).toBeGreaterThan(0);
});
