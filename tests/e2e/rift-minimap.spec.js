const {test,expect}=require('@playwright/test');
test('arena minimap tracks movement, terrain and display preferences',async({page})=>{
 await page.goto('/abyss/rift?scenario=drop-edge');await expect(page.locator('#rift-start')).toBeEnabled();const map=page.locator('#rift-minimap');await expect(map).toBeVisible();await expect(map.locator('[data-kind=ledge]')).toHaveCount(1);await expect(map.locator('[data-kind=enemy]')).toHaveCount(1);await expect(map).toContainText('from anywhere');
 await page.locator('#rift-auto').uncheck();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();const marker=map.locator('[data-kind=player]'),before=Number(await marker.getAttribute('cx'));
 await page.keyboard.down('KeyD');try{await expect.poll(async()=>Number(await marker.getAttribute('cx'))).toBeGreaterThan(before+35);}finally{await page.keyboard.up('KeyD');}
 await page.locator('.rift-settings > summary').click();await page.getByLabel('Arena minimap',{exact:true}).uncheck();await expect(map).toBeHidden();await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(map).toBeHidden();await page.locator('.rift-settings > summary').click();await page.getByLabel('Arena minimap',{exact:true}).check();await expect(map).toBeVisible();await page.locator('#rift-clean-screenshot').check();await expect(map).toBeHidden();await page.locator('#rift-clean-screenshot').uncheck();await expect(map).toBeVisible();await page.locator('.rift-settings > summary').click();
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await map.screenshot({path:'test-results/minimap-mobile.png'});
});
test('minimap shows saved exit coordinates and hazard state without inventing a bank location',async({page})=>{
 await page.goto('/abyss/rift?scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();
 const map=page.locator('#rift-minimap');await expect(map.locator('[data-phase=active]')).not.toHaveCount(0);await expect(map.locator('[data-phase=active]').first()).toHaveCSS('fill','rgb(219, 103, 78)');await map.screenshot({path:'test-results/minimap-active-hazards.png'});
 await page.evaluate(run=>{run=structuredClone(run);run.room_objective={kind:'escape_collapse',zone:{x:1400,y:460},complete:false};run.level.rooms[run.room].hazards=[{x:400,y:350,w:100,h:40,kind:'fire',period:7,offset:0,duration:1}];run.clock=0;RiftMinimap.update(run);},data.run);
 const exit=map.locator('[data-kind=exit]');await expect(exit).toHaveAttribute('transform','translate(280 40.8)');await expect(map.locator('[data-phase=warning]')).toHaveCount(1);await expect(exit.locator('title')).toContainText('Exit');
 await page.evaluate(run=>{run=structuredClone(run);run.room_objective={kind:'escort_spirit',zone:{x:1450,y:320},complete:false};RiftMinimap.update(run);},data.run);await expect(exit).toHaveAttribute('transform','translate(290 7.2)');
 await page.evaluate(run=>{run=structuredClone(run);run.room_objective=null;run.status='cleared';RiftMinimap.update(run);},data.run);await expect(map.locator('[data-kind=exit]')).toHaveCount(0);await expect(map).toContainText('Bank & leave');await expect(map.locator('[data-phase=active]')).toHaveCount(0);
});

for(const scenario of ['collapse','relic'])test('minimap locates the actual '+scenario+' exit',async({page})=>{
 await page.goto('/abyss/rift?scenario='+scenario);await expect(page.locator('#rift-start')).toBeEnabled();const run=(await(await page.request.get('/api/abyss/rift')).json()).run;const z=run.room_objective.zone;
 await expect(page.locator('#rift-minimap [data-kind=exit]')).toHaveAttribute('transform','translate('+(Math.round(z.x*.2*10)/10)+' '+(Math.round((6+(z.y-315)*.24)*10)/10)+')');await expect(page.locator('#rift-minimap svg')).toHaveAttribute('aria-label',new RegExp('Exit '+Math.round(z.x)+', '+Math.round(z.y)));await page.locator('#rift-minimap').screenshot({path:'test-results/minimap-'+scenario+'.png'});
});
test('minimap refreshes destroyed terrain, defeated enemies, disabled hazards and practice guidance',async({page})=>{
 await page.goto('/abyss/rift?scenario=terrain-cover');await expect(page.locator('#rift-start')).toBeEnabled();const run=(await(await page.request.get('/api/abyss/rift')).json()).run,map=page.locator('#rift-minimap');await expect(map.locator('[data-kind=wood]')).toHaveCount(1);await expect(map.locator('[data-kind=stone]')).toHaveCount(1);
 await page.evaluate(run=>{run=structuredClone(run);run.enemies[0].hp=0;run.enemies.push({id:'cage',name:'Captive',kind:'cage',x:400,y:410,hp:1});run.level.rooms[0].cover[0].hp=0;run.level.rooms[0].hazards=[{x:500,y:350,w:100,h:30,kind:'fire',period:7,offset:0,duration:1,disabled:true}];RiftMinimap.update(run);},run);
 await expect(map.locator('[data-kind=wood]')).toHaveCount(0);await expect(map.locator('[data-kind=enemy]')).toHaveCount(0);await expect(map.locator('[data-kind=objective]')).toHaveCount(1);await expect(map.locator('[data-phase=off]')).toHaveCount(1);
 await page.evaluate(run=>{run.practice={arena:run.level.rooms[0]};RiftMinimap.update(run);},run);await expect(map).toContainText('Practice arena: no checkpoint banking');
});
