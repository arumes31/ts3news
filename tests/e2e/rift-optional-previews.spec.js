const {test,expect}=require('@playwright/test');

// Thumbnails normally reuse combat atlas URLs. Tag only CSS preview consumers
// to isolate their network dependency from the required canvas bitmap loads.
test('pending and failed optional thumbnails never block first play',async({page})=>{
 const held=[],errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.addInitScript(()=>{window.optionalPreviewURL=src=>{const url=new URL(src,location.href);url.searchParams.set('optionalPreview','1');return url.href;};});
 await page.route('**/static/rift.js*',async route=>{
  const response=await route.fetch(),source=await response.text(),anchor="art.dataset.regionArt=window.RiftRegionSections.regions[level.region].url;";
  expect(source.split(anchor)).toHaveLength(2);
  await route.fulfill({response,body:source.replace(anchor,"art.dataset.regionArt=window.optionalPreviewURL(window.RiftRegionSections.regions[level.region].url);")});
 });
 await page.route('**/static/rift_bestiary.js*',async route=>{
  const response=await route.fetch(),source=await response.text();
  expect(source.split('assetURL(pose.asset)')).toHaveLength(3);
  await route.fulfill({response,body:source.replaceAll('assetURL(pose.asset)','window.optionalPreviewURL(assetURL(pose.asset))')});
 });
 await page.route('**/*optionalPreview=1*',route=>{held.push(route);});
 try{
  await page.goto('/abyss/rift',{waitUntil:'domcontentloaded'});
  await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-campaign').evaluate(node=>{node.open=true;});
  await page.locator('#rift-campaign [data-region-art]').first().scrollIntoViewIfNeeded();
  await expect.poll(()=>held.length).toBeGreaterThan(0);
  await expect(page.locator('#rift-monsters article')).toHaveCount(0);
  await page.locator('#rift-monsters').evaluate(node=>{node.closest('details').open=true;});
  await expect.poll(()=>page.locator('#rift-monsters article').count()).toBeGreaterThan(100);
  await page.locator('#rift-monsters article').first().scrollIntoViewIfNeeded();
  await expect.poll(()=>held.length).toBeGreaterThan(1);
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
  const frame=await page.evaluate(()=>window.RiftRenderer.frameCount);
  await expect.poll(()=>page.evaluate(()=>window.RiftRenderer.frameCount)).toBeGreaterThan(frame+10);
  for(const route of held.splice(0))await route.abort('failed');
  await page.keyboard.press('Escape');await expect(page.locator('#rift-start')).toHaveText('Resume expedition');await expect(page.locator('#rift-start')).toBeEnabled();
  expect(errors).toEqual([]);
 }finally{for(const route of held.splice(0))await route.abort('failed').catch(()=>{});}
});
