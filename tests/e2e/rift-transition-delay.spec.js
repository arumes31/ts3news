const {test,expect}=require('@playwright/test');

test('transition delay rejects invalid storage, persists and allows immediate continuation',async({page})=>{
 await page.addInitScript(()=>{if(!sessionStorage.getItem('delaySeed')){localStorage.setItem('riftTransitionDelay','-1');sessionStorage.setItem('delaySeed','1');}});
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('.rift-settings > summary').click();const delay=page.locator('#rift-transition-delay');await expect(delay).toHaveValue('1.2');await delay.selectOption('10');
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('.rift-settings > summary').click();await expect(delay).toHaveValue('10');await page.locator('.rift-settings > summary').click();
 const before=(await(await page.request.get('/api/abyss/rift')).json()).run;await page.locator('#rift-start').click();await expect(page.locator('#rift-transition')).toContainText('Next:');await page.waitForTimeout(1600);expect((await(await page.request.get('/api/abyss/rift')).json()).run.room).toBe(before.room);await page.locator('#rift-next').click();await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.room).toBe(before.room+1);
});

test('automatic advancement honors the selected longer countdown',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('.rift-settings > summary').click();await page.locator('#rift-transition-delay').selectOption('3');await page.locator('.rift-settings > summary').click();const before=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.locator('#rift-start').click();await expect(page.locator('#rift-transition')).toContainText('Next:');await page.waitForTimeout(1600);expect((await(await page.request.get('/api/abyss/rift')).json()).run.room).toBe(before.room);await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.room,{timeout:5000}).toBe(before.room+1);
});
