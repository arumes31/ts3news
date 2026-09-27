const {test,expect}=require('@playwright/test');
test('custom skill order and empty slots survive failed recovery reads',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 const slots=page.locator('#rift-loadout select');const original=await slots.evaluateAll(nodes=>nodes.map(n=>n.value));expect(original).toHaveLength(3);
 await slots.nth(0).selectOption('');await slots.nth(1).selectOption(original[0]);await slots.nth(0).selectOption(original[1]);await slots.nth(2).selectOption('');const selected=[original[1],original[0],''];
 let failedStart=false,failedRead=false,posts=0;
 await page.route('**/api/abyss/rift',route=>{
  if(route.request().method()==='POST'){posts++;if(!failedStart){failedStart=true;return route.fulfill({status:500,body:'unavailable'});}}
  else if(failedStart&&!failedRead){failedRead=true;return route.fulfill({status:500,body:'unavailable'});}
  return route.continue();
 });
 await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toHaveText('Recover expedition');await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toHaveText('Retry loading');
 expect(await slots.evaluateAll(nodes=>nodes.map(n=>n.value))).toEqual(selected);
 await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toHaveText('Enter the ruins →');expect(await slots.evaluateAll(nodes=>nodes.map(n=>n.value))).toEqual(selected);expect(posts).toBe(1);
 const started=page.waitForResponse(response=>response.url().endsWith('/api/abyss/rift')&&response.request().postDataJSON()?.kind==='start');await page.locator('#rift-start').click();expect((await(await started).json()).run.build.skills.map(s=>s.id)).toEqual(selected.filter(Boolean));await page.keyboard.press('Escape');
});

test('recovery of an already committed start uses the saved expedition build',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();const slots=page.locator('#rift-loadout select');const original=await slots.evaluateAll(nodes=>nodes.map(n=>n.value));let committed=null;
 await page.route('**/api/abyss/rift',async route=>{if(route.request().method()==='POST'&&route.request().postDataJSON().kind==='start'&&!committed){const response=await route.fetch();committed=(await response.json()).run;await route.abort();return;}await route.continue();});
 await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toHaveText('Recover expedition');expect(committed).not.toBeNull();await slots.nth(0).selectOption('');
 await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toHaveText('Resume expedition');expect(await slots.evaluateAll(nodes=>nodes.map(n=>n.value))).toEqual(original);await expect(slots.nth(0)).toBeDisabled();
});
