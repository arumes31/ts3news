const {test,expect}=require('@playwright/test');

for(const phase of ['load','fight','bank'])test('missing character during '+phase+' stops play and offers Abyss recovery',async({page})=>{
  let posts=0;
  await page.route('**/api/abyss/rift',async route=>{
    const request=route.request();
    if(request.method()==='POST')posts++;
    const kind=request.method()==='POST'?request.postDataJSON().kind:'';
    if(phase==='load'&&request.method()==='GET'||phase==='fight'&&kind==='step'||phase==='bank'&&kind==='exit')return route.fulfill({status:410,contentType:'application/json',body:JSON.stringify({ok:false,code:'CHARACTER_MISSING',error:'The Abyss character is no longer available.'})});
    await route.continue();
  });
  await page.route('**/abyss',route=>route.fulfill({contentType:'text/html',body:'<h1>Abyss</h1>'}));
  await page.goto('/abyss/rift'+(phase==='bank'?'?scenario=checkpoint':''));
  if(phase!=='load')await page.locator('#rift-start').click();
  if(phase==='bank')await page.locator('#rift-exit').click();
  await expect(page.locator('#rift-overlay-title')).toHaveText('Character unavailable.');
  await expect(page.locator('#rift-character-return')).toBeVisible();
  await expect(page.locator('#rift-character-return')).toHaveAttribute('href','/abyss');
  await expect(page.locator('#rift-start')).toBeDisabled();
  if(phase==='bank')await expect(page.locator('#rift-overlay-copy')).toContainText('Reward delivery is unconfirmed');
  const stopped=posts;await page.keyboard.press('Space');await page.waitForTimeout(350);expect(posts).toBe(stopped);
  await page.locator('#rift-character-return').click();await expect(page).toHaveURL(/\/abyss$/);
});
