const {test,expect}=require('@playwright/test');
test('skip remembers boss introductions locally without changing combat',async({page})=>{
 await page.goto('/abyss/rift?scenario=boss-windup');await expect(page.locator('#rift-start')).toBeEnabled();
 const intro=page.locator('#rift-boss-intro');await expect(intro).toBeVisible();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const before=await saved();await expect(intro).toContainText(before.enemies.find(e=>e.kind==='boss').name);
 await page.locator('#rift-boss-intro-skip').click();await expect(intro).toBeHidden();expect(await saved()).toEqual(before);
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(intro).toBeHidden();
 await page.locator('.rift-settings > summary').click();await page.locator('#rift-boss-intro-reset').click();await expect(intro).toBeVisible();
 expect(await saved()).toEqual(before);
});
test('boss intro works without local storage and fits narrow screens',async({page})=>{
 await page.addInitScript(()=>{Storage.prototype.getItem=()=>{throw Error('blocked')};Storage.prototype.setItem=()=>{throw Error('blocked')};});
 await page.setViewportSize({width:390,height:844});await page.goto('/abyss/rift?scenario=boss-windup');await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-boss-intro')).toBeVisible();
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await page.locator('#rift-boss-intro').screenshot({path:'test-results/boss-intro-mobile.png'});
 await page.locator('#rift-boss-intro-skip').click();await expect(page.locator('#rift-boss-intro')).toBeHidden();
});

test('introductions expire without remembering a skip and do not hide other bosses',async({page})=>{
 await page.goto('/abyss/rift?scenario=boss-windup');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();const bosses=data.bestiary.filter(actor=>actor.kind==='boss');
 await page.locator('#rift-boss-intro-skip').click();
 const fresh={...data.run,id:'other-boss',enemies:[{...bosses.find(b=>b.name!==data.run.enemies.find(e=>e.kind==='boss').name),id:'other',hp:100}]};
 await page.clock.install();await page.evaluate(run=>window.RiftBossIntro.update(run),fresh);
 await expect(page.locator('#rift-boss-intro')).toBeVisible();await page.clock.fastForward(8100);await expect(page.locator('#rift-boss-intro')).toBeHidden();
 await page.evaluate(run=>window.RiftBossIntro.update({...run,id:'another-room'}),fresh);await expect(page.locator('#rift-boss-intro')).toBeVisible();
});
