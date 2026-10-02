const {test,expect}=require('@playwright/test');
for(const mode of ['movement','jump','combo','toxic_tide'])test('play '+mode+' drill and reset without campaign changes',async({page})=>{
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const campaign=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.goto('/abyss/rift?practice='+mode);await expect(page.locator('#rift-practice-guide')).toBeVisible();await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift?practice='+mode)).json()).run;
 await expect.poll(async()=>(await saved())?.status).toBe('fighting');
 if(mode==='combo')await page.keyboard.down('KeyJ');else{await page.keyboard.down('KeyD');if(mode==='jump'||mode==='toxic_tide')await page.keyboard.down('Space');}
 await expect.poll(async()=>(await saved())?.status,{timeout:20000}).toBe('complete');
 await page.keyboard.up('KeyJ');await page.keyboard.up('KeyD');await page.keyboard.up('Space');
 await expect(page.locator('#rift-overlay-title')).toHaveText('Drill complete.');
 const complete=await saved();expect(complete.practice.completed).toBe(true);expect(complete.drops).toHaveLength(0);expect(complete.level).toBeFalsy();
 let navigations=0;const navigated=()=>navigations++;page.on('framenavigated',navigated);
 await page.locator('#rift-practice-reset').click();await expect.poll(async()=>(await saved())?.status).toBe('fighting');expect((await saved()).practice.hits).toBe(0);expect(navigations).toBe(0);page.off('framenavigated',navigated);
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

test('directional guard blocks only while facing the training attacker',async({page})=>{
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.goto('/abyss/rift?practice=guard');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift?practice=guard')).json()).run;
 await page.keyboard.press('KeyA');await page.keyboard.down('KeyL');
 await expect.poll(async()=>(await saved()).stats.damage_taken).toBeGreaterThan(0);
 expect((await saved()).stats.guards).toBe(0);await page.keyboard.press('KeyD');
 await expect.poll(async()=>(await saved()).status,{timeout:15000}).toBe('complete');await page.keyboard.up('KeyL');
 await expect(page.locator('#rift-practice-progress')).toHaveText('Drill complete');expect((await saved()).stats.guards).toBe(3);expect(errors).toEqual([]);
 await page.locator('#rift-practice-reset').click();await expect.poll(async()=>(await saved()).stats.guards).toBe(0);
});

test('hazard practice punishes missed warnings and rewards three evasions',async({page})=>{
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.goto('/abyss/rift?practice=hazard');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift?practice=hazard')).json()).run;
 await expect.poll(async()=>(await saved()).stats.damage_taken).toBeGreaterThan(0);expect((await saved()).practice.dodges||0).toBe(0);
 await page.locator('#rift-practice-reset').click();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 for(const [index,key] of ['KeyD','KeyA','KeyD'].entries()){
  await page.keyboard.down(key);await expect.poll(async()=>(await saved()).practice.dodges||0,{timeout:7000}).toBe(index+1);await page.keyboard.up(key);
 }
 await expect(page.locator('#rift-overlay-title')).toHaveText('Drill complete.');const run=await saved();expect(run.stats.damage_taken).toBe(0);expect(run.drops).toHaveLength(0);expect(errors).toEqual([]);
});

test('practice recovery controls preserve the drill and do not enter campaign storage',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();const campaign=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.goto('/abyss/rift?practice=guard&scenario=practice-tools');await expect(page.locator('[data-practice-action="practice_health"]')).toBeEnabled();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift?practice=guard')).json()).run;
 const initial=await saved();expect(initial.player.hp).toBeLessThan(initial.player.max_hp);expect(initial.player.mana).toBe(12);
 await page.locator('[data-practice-action="practice_health"]').click();await expect.poll(async()=>(await saved()).player.hp).toBe(initial.player.max_hp);
 await page.locator('[data-practice-action="practice_mana"]').click();await expect.poll(async()=>(await saved()).player.mana).toBe(100);
 await page.locator('[data-practice-action="practice_cooldowns"]').click();await expect(page.locator('#rift-practice-tool-status')).toContainText('Reset skill cooldowns applied');
 const after=await saved();for(const skill of [...after.build.skills,...after.build.signatures,after.build.ultimate])expect(after.skill_timers[skill.id]).toBe(0);
 expect(after.skill_timers.slowed).toBe(.6);expect(after.skill_timers['hazard-0']).toBe(.7);expect(after.practice.hits).toBe(2);expect(after.paused).toBe(true);expect(after.stats).toEqual(initial.stats);expect(after.id).toBe(initial.id);
 expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(campaign);
});

test('health refill pauses a running practice fight before recovery',async({page})=>{
 await page.goto('/abyss/rift?practice=guard');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift?practice=guard')).json()).run;
 await expect.poll(async()=>(await saved()).stats.damage_taken).toBeGreaterThan(0);
 await page.locator('[data-practice-action="practice_health"]').click();await expect(page.locator('#rift-practice-tool-status')).toContainText('Refill health applied');
 const after=await saved();expect(after.paused).toBe(true);expect(after.player.hp).toBe(after.player.max_hp);expect(after.stats.damage_taken).toBeGreaterThan(0);await expect(page.locator('#rift-start')).toHaveText('Resume drill');
});

test('recovery pauses a restored drill that was not running in this page',async({page})=>{
 await page.goto('/abyss/rift?practice=guard');await expect(page.locator('#rift-start')).toBeEnabled();
 const response=await page.request.post('/api/abyss/rift?practice=guard',{data:{kind:'start',request_id:'restored-practice-tool',run_id:'',revision:1,input:{}}});expect(response.ok()).toBe(true);
 await page.reload();await expect(page.locator('#rift-start')).toHaveText('Resume drill');
 expect((await(await page.request.get('/api/abyss/rift?practice=guard')).json()).run.paused).toBe(false);
 await page.locator('[data-practice-action="practice_mana"]').click();await expect(page.locator('#rift-practice-tool-status')).toContainText('Refill mana applied');
 expect((await(await page.request.get('/api/abyss/rift?practice=guard')).json()).run.paused).toBe(true);
});
