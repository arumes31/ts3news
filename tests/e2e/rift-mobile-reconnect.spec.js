const {test,expect}=require('@playwright/test');

test.use({hasTouch:true,isMobile:true});
for(const viewport of [{width:320,height:568},{width:390,height:844},{width:844,height:390}])test('reconnect remains reachable at '+viewport.width+'x'+viewport.height,async({page})=>{
 await page.setViewportSize(viewport);
 await page.goto('/abyss/rift?practice=skills');
 const start=page.locator('#rift-start');
 await expect(start).toBeEnabled();await start.tap();
 await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.route('**/api/abyss/rift**',async route=>{
  if(route.request().method()==='POST')await route.fulfill({status:503,contentType:'application/json',body:JSON.stringify({error:'Connection interrupted. Please recover your saved expedition and try again.'})});
  else await route.continue();
 });
 await expect(start).toHaveText('Recover expedition');
 await page.locator('#rift-overlay-title').scrollIntoViewIfNeeded();
 expect(await page.locator('#rift-overlay-title').evaluate(title=>title.getBoundingClientRect().top>=document.querySelector('#rift-viewport').getBoundingClientRect().top)).toBe(true);
 await start.scrollIntoViewIfNeeded();
 const geometry=await start.evaluate(button=>{
  const r=button.getBoundingClientRect(),v=document.querySelector('#rift-viewport').getBoundingClientRect();
  return {inside:r.top>=v.top&&r.bottom<=v.bottom,hit:button.contains(document.elementFromPoint(r.x+r.width/2,r.y+r.height/2)),height:r.height};
 });
 expect(geometry.inside).toBe(true);expect(geometry.hit).toBe(true);expect(geometry.height).toBeGreaterThanOrEqual(44);
 await page.locator('.rift-game').screenshot({path:test.info().outputPath('reconnect.png')});
 await page.unroute('**/api/abyss/rift**');
 await start.tap();await expect(start).toHaveText('Resume drill');
 await start.tap();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.locator('#rift-pause').tap();
});
