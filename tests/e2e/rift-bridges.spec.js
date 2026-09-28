const {test,expect}=require('@playwright/test');

for(const reduced of [false,true])test((reduced?'reduced motion: ':'')+'bridge rails constrain keyboard movement and recover with the deck',async({page},info)=>{
 test.setTimeout(60000);
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.emulateMedia({reducedMotion:reduced?'reduce':'no-preference'});
 await page.goto('/abyss/rift?scenario=bridge');await expect(page.locator('#rift-start')).toHaveText('Resume expedition');await page.locator('#rift-auto').uncheck();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const hold=async(key,check)=>{await page.keyboard.down(key);try{await check();}finally{await page.keyboard.up(key);}};
 const before=await saved();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await hold('d',async()=>{await expect.poll(async()=>(await saved()).player.x).toBeGreaterThan(375);await page.waitForTimeout(400);});
 expect((await saved()).player.x).toBeLessThanOrEqual(390);
 await page.keyboard.press('Space');await hold('d',()=>page.waitForTimeout(400));expect((await saved()).player.x).toBeLessThanOrEqual(390);
 await hold('s',()=>expect.poll(async()=>(await saved()).player.y,{intervals:[20]}).toBeGreaterThanOrEqual(400));
 await hold('d',()=>expect.poll(async()=>(await saved()).player.x,{intervals:[20]}).toBeGreaterThanOrEqual(490));
 await hold('w',()=>page.waitForTimeout(600));const onDeck=await saved();expect(onDeck.player.y).toBeGreaterThanOrEqual(380);expect(onDeck.floor).toBe('wood');
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const paused=await saved();expect(paused.player.hp).toBe(before.player.hp);
 await expect(page.locator('#rift-minimap [data-kind="bridge"]')).toHaveCount(1);
 await page.locator('#rift-viewport').screenshot({path:info.outputPath('bridge-desktop.png'),style:'#rift-overlay {visibility:hidden!important}'});
 await page.setViewportSize({width:390,height:844});await page.locator('#rift-viewport').screenshot({path:info.outputPath('bridge-mobile.png'),style:'#rift-overlay {visibility:hidden!important}'});
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
 const recovered=await saved();expect(recovered.level.rooms[0].bridges).toEqual(paused.level.rooms[0].bridges);expect(recovered.player.x).toBe(paused.player.x);expect(recovered.player.y).toBe(paused.player.y);
 await page.locator('#rift-start').click();await hold('d',()=>expect.poll(async()=>(await saved()).player.x).toBeGreaterThan(630));expect(errors).toEqual([]);
});

test('campaign bridge preview explains the walkable deck',async({page})=>{
 await page.goto('/abyss/rift?mission=4');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-mission-preview > summary').click();
 await expect(page.locator('#rift-room-previews [data-terrain="bridge"]')).toHaveCount(1);
 await expect(page.locator('#rift-room-previews [data-terrain="bridge"] title')).toContainText('gaps are not walkable');
});
