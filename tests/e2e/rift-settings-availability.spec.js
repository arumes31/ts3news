const {test,expect}=require('@playwright/test');

async function changeSettings(page){
 await page.locator('.rift-settings > summary').click();await page.locator('#rift-text-scale').selectOption('1.25');await page.locator('#rift-interface-muted').check();await page.locator('#rift-steady-ambience').check();await expect(page.locator('#rift-settings-summary')).toContainText('Text 125%');await expect(page.locator('#rift-settings-summary')).toContainText('Interface muted');await page.locator('.rift-settings > summary').click();
 expect(await page.evaluate(()=>JSON.parse(localStorage.getItem('riftDisplay')).textScale)).toBe(1.25);expect(await page.evaluate(()=>JSON.parse(localStorage.getItem('riftAudio:interfaceMuted')))).toBe(true);
}

test('settings remain editable while paused without resuming or writing the expedition',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');const before=(await(await page.request.get('/api/abyss/rift')).json()).run;
 let writes=0;page.on('request',r=>{if(r.url().includes('/api/abyss/rift')&&r.method()==='POST')writes++;});await changeSettings(page);expect(writes).toBe(0);expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(before);await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
});

test('settings remain editable on a defeated response without starting a retry',async({page})=>{
 await page.route('**/api/abyss/rift',async route=>{const response=await route.fetch(),data=await response.json();if(data.run){data.run.status='defeated';data.run.player.hp=0;data.run.gold=0;data.run.drops=[];data.hazard_hit_damage=0;}await route.fulfill({response,json:data});});await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-defeat-guide')).toBeVisible();
 let writes=0;page.on('request',r=>{if(r.url().includes('/api/abyss/rift')&&r.method()==='POST')writes++;});await changeSettings(page);expect(writes).toBe(0);await expect(page.locator('#rift-defeat-guide')).toBeVisible();await page.setViewportSize({width:390,height:844});await page.locator('.rift-settings > summary').click();await expect(page.locator('#rift-text-scale')).toBeVisible();expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});

test('settings remain editable after a failed combat request without triggering recovery',async({page})=>{
 await page.route('**/api/abyss/rift',route=>route.request().method()==='POST'&&route.request().postDataJSON()?.kind==='step'?route.abort():route.continue());await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await expect(page.locator('#rift-start')).toHaveText('Recover expedition');const before=(await(await page.request.get('/api/abyss/rift')).json()).run;
 let writes=0;page.on('request',r=>{if(r.url().includes('/api/abyss/rift')&&r.method()==='POST')writes++;});await changeSettings(page);expect(writes).toBe(0);await expect(page.locator('#rift-start')).toHaveText('Recover expedition');expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(before);
});
