const {test,expect}=require('@playwright/test');
for(const mode of ['movement','jump','combo'])test('play '+mode+' drill and reset without campaign changes',async({page})=>{
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const campaign=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.goto('/abyss/rift?practice='+mode);await expect(page.locator('#rift-practice-guide')).toBeVisible();await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift?practice='+mode)).json()).run;
 await expect.poll(async()=>(await saved())?.status).toBe('fighting');
 if(mode==='combo')await page.keyboard.down('KeyJ');else{await page.keyboard.down('KeyD');if(mode==='jump')await page.keyboard.down('Space');}
 await expect.poll(async()=>(await saved())?.status,{timeout:20000}).toBe('complete');
 await page.keyboard.up('KeyJ');await page.keyboard.up('KeyD');await page.keyboard.up('Space');
 await expect(page.locator('#rift-overlay-title')).toHaveText('Drill complete.');
 const complete=await saved();expect(complete.practice.completed).toBe(true);expect(complete.drops).toHaveLength(0);expect(complete.level).toBeFalsy();
 await page.locator('#rift-practice-reset').click();await expect.poll(async()=>(await saved())?.status).toBe('fighting');expect((await saved()).practice.hits).toBe(0);
 await page.getByRole('link',{name:'Return to campaign',exact:true}).click();await expect(page.locator('#rift-start')).toBeEnabled();
 expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(campaign);expect(errors).toEqual([]);
});

test('practice fits a narrow screen and restores its saved drill',async({page})=>{
 await page.setViewportSize({width:390,height:844});
 await page.goto('/abyss/rift?practice=jump');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const before=(await(await page.request.get('/api/abyss/rift?practice=jump')).json()).run;
 await page.reload();await expect(page.locator('#rift-start')).toHaveText('Resume drill');
 expect((await(await page.request.get('/api/abyss/rift?practice=jump')).json()).run).toEqual(before);
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await expect(page.locator('#rift-loadout-presets')).toBeHidden();await expect(page.locator('#rift-pending-hud')).toBeHidden();await expect(page.locator('#rift-campaign')).toBeHidden();
 await page.screenshot({path:'test-results/practice-mobile.png',fullPage:true});
});
