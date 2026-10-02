const {test,expect}=require('@playwright/test');

test('diagnostic preview and download allow only technical fields and never mutate the run',async({page},testInfo)=>{
 await page.goto('/abyss/rift?scenario=checkpoint&private=URL_SECRET');await expect(page.locator('#rift-start')).toBeEnabled();
 const before=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.locator('.rift-settings > summary').click();
 await expect(page.locator('#rift-download-diagnostics')).toBeDisabled();
 await page.evaluate(()=>{
  localStorage.setItem('private-account','STORAGE_SECRET');document.cookie='private=COOKIE_SECRET';
  window.RiftDisplay.privateAccount='DISPLAY_SECRET';window.RiftAudio.privateAccount='AUDIO_SECRET';
  window.RiftInputDiagnostics={samples:[{total:120,queue:20,request:100,actions:['PRIVATE_SKILL']}],account:'TIMING_SECRET'};
  window.RiftPayloadDiagnostics={samples:[],maxResponseBytes:1024,totalResponseBytes:2048,count:2,raw:'RAW_SECRET'};
 });
 let mutations=0;page.on('request',r=>{if(r.url().includes('/api/abyss/rift')&&r.method()==='POST')mutations++;});
 await page.locator('#rift-prepare-diagnostics').click();
 const preview=await page.locator('#rift-diagnostic-preview').inputValue(),data=JSON.parse(preview);
 expect(Object.keys(data).sort()).toEqual(['audio','display','environment','format','last_confirmed_state','measurements','notes'].sort());
 expect(data.format).toBe('rift-brawl-diagnostics-v1');expect(data.last_confirmed_state).toEqual({schema:before.schema,status:before.status,mission:before.level.id,tier:before.room+1,paused:before.paused});
 expect(data.measurements.input_total_ms).toEqual({samples:1,average:120,p95:120,max:120});
 expect(data.measurements.frame_interval_ms).toBeNull();
 expect(preview).not.toMatch(/SECRET|PRIVATE_SKILL|COOKIE|checkpoint|Rowan/);
 for(const key of ['id','start_key','last_request_id','banked_items','banked_gold','player','build','user','events','url','storage','logs'])expect(preview).not.toContain('"'+key+'"');
 expect(preview.length).toBeLessThan(6000);
 const pending=page.waitForEvent('download');await page.locator('#rift-download-diagnostics').click();const download=await pending;
 expect(download.suggestedFilename()).toBe('rift-brawl-diagnostics.json');const chunks=[];for await(const chunk of await download.createReadStream())chunks.push(chunk);
 expect(Buffer.concat(chunks).toString('utf8')).toBe(preview);
 expect(mutations).toBe(0);expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(before);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await page.locator('#rift-diagnostics').screenshot({path:testInfo.outputPath('diagnostic-preview-mobile.png')});
});

test('diagnostics bound samples, reject unknown values and keep download failures private',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('.rift-settings > summary').click();
 await page.evaluate(()=>{
  window.RiftDiagnostics.update({schema:1,status:'PRIVATE_STATUS',level:{id:'PRIVATE_MISSION'},room:-1,paused:'PRIVATE_PAUSE',player:{name:'PRIVATE_NAME'}});
  window.RiftDisplay.fps='PRIVATE_FPS';window.RiftDisplay.enemyNames='PRIVATE_NAME';
  window.RiftRenderer.frameDiagnostics={samples:Array.from({length:1000},()=>({interval:20,render:2,secret:'PRIVATE'}))};
  window.RiftInputDiagnostics={samples:[{total:-1,queue:Infinity,request:'PRIVATE'}]};
 });
 await page.locator('#rift-prepare-diagnostics').click();const text=await page.locator('#rift-diagnostic-preview').inputValue(),data=JSON.parse(text);
 expect(text).not.toContain('PRIVATE');expect(data.last_confirmed_state).toEqual({schema:1,status:null,mission:null,tier:null,paused:null});
 expect(data.display.fps).toBeNull();expect(data.display.enemyNames).toBeNull();expect(data.measurements.frame_interval_ms).toEqual({samples:120,average:20,p95:20,max:20});expect(data.measurements.input_total_ms).toBeNull();
 await page.evaluate(()=>URL.createObjectURL=()=>{throw new Error('PRIVATE_ERROR');});await page.locator('#rift-download-diagnostics').click();
 await expect(page.locator('#rift-diagnostic-status')).toHaveText('Download unavailable. You can select and copy the preview instead.');
 await expect(page.locator('#rift-diagnostic-preview')).toHaveValue(text);
});


test('diagnostics remain available after an initial read failure without browser storage',async({page})=>{
 await page.addInitScript(()=>{Storage.prototype.getItem=()=>{throw new Error('PRIVATE_STORAGE_FAILURE');};});
 await page.route('**/api/abyss/rift',route=>route.abort());
 await page.goto('/abyss/rift');await page.locator('.rift-settings > summary').click();await page.locator('#rift-prepare-diagnostics').click();
 const text=await page.locator('#rift-diagnostic-preview').inputValue(),data=JSON.parse(text);
 expect(data.last_confirmed_state).toBeNull();expect(data.measurements.frame_interval_ms).toBeNull();expect(text).not.toContain('PRIVATE');await expect(page.locator('#rift-download-diagnostics')).toBeEnabled();
});

test('diagnostics summarize live frame debug samples without exporting raw samples',async({page})=>{
 await page.goto('/abyss/rift?riftFrameDebug=1');await expect(page.locator('#rift-start')).toBeEnabled();
 await expect.poll(()=>page.evaluate(()=>window.RiftRenderer.frameDiagnostics.samples.length)).toBeGreaterThan(2);
 await page.locator('.rift-settings > summary').click();await page.locator('#rift-prepare-diagnostics').click();
 const data=JSON.parse(await page.locator('#rift-diagnostic-preview').inputValue());
 for(const key of ['frame_interval_ms','render_ms']){const stats=data.measurements[key];expect(stats.samples).toBeGreaterThan(2);expect(stats.samples).toBeLessThanOrEqual(120);expect(stats.average).toBeGreaterThanOrEqual(0);expect(stats.max).toBeGreaterThanOrEqual(stats.p95);expect(Array.isArray(stats)).toBe(false);}
});


test('diagnostics report the confirmed bank-and-leave terminal state',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.locator('#rift-exit').click();
 await expect(page.locator('#rift-banked')).toHaveText('30 gold · 1 item');
 await page.locator('.rift-settings > summary').click();await page.locator('#rift-prepare-diagnostics').click();
 const data=JSON.parse(await page.locator('#rift-diagnostic-preview').inputValue());expect(data.last_confirmed_state.status).toBe('banked');
});
