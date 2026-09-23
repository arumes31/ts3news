const {test,expect}=require('@playwright/test');
test('destroying generators permanently shuts down linked hazards',async({page})=>{
 test.setTimeout(90000);
 await page.goto('/abyss/rift?scenario=generators');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const progress=page.locator('#rift-room-objective-progress');await expect(progress).toContainText('Generators 0/3');
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.locator('#rift-viewport').screenshot({path:'test-results/generators-powered.png'});
 await page.keyboard.down('KeyJ');try{await expect(progress).toContainText('Generators 1/3');}finally{await page.keyboard.up('KeyJ');}
 const hazards=(await saved()).level.rooms[1].hazards;expect(hazards.map(h=>!!h.disabled)).toEqual([true,false,false]);
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();expect((await saved()).level.rooms[1].hazards.map(h=>!!h.disabled)).toEqual([true,false,false]);await page.locator('#rift-auto').uncheck();
 await page.evaluate(()=>{window.generatorCues=[];const play=window.RiftAudio.play;window.RiftAudio.play=function(kind,...args){if(kind.startsWith('generator_'))window.generatorCues.push(kind);return play.call(this,kind,...args);};});
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 const move=async(key,axis,target,up=true)=>{await page.keyboard.down(key);try{await expect.poll(async()=>(await saved()).player[axis],{intervals:[20],timeout:10000})[up?'toBeGreaterThan':'toBeLessThan'](target);}finally{await page.keyboard.up(key);}};
 await move('KeyD','x',220);await move('KeyS','y',410);const safe=await saved();
 await expect.poll(async()=>(await saved()).clock,{timeout:12000,intervals:[250]}).toBeGreaterThan(safe.clock+7.2);expect((await saved()).player.hp).toBe(safe.player.hp);await expect(page.locator('#rift-area-effects')).toHaveAttribute('data-effect-state','none');await page.locator('#rift-viewport').screenshot({path:'test-results/generator-disabled.png'});
 await move('KeyW','y',355,false);
 for(const x of [430,680]){await move('KeyD','x',x);await page.keyboard.down('KeyJ');try{await expect(progress).toContainText(x===430?'Generators 2/3':'Generators 3/3');}finally{await page.keyboard.up('KeyJ');}}
 await expect.poll(async()=>(await saved()).status).toBe('cleared');const result=await saved();expect(result.level.rooms[1].hazards.every(h=>h.disabled)).toBe(true);expect(result.stats.kills).toBe(0);expect(result.drops.length).toBe(0);await expect.poll(()=>page.evaluate(()=>window.generatorCues.filter(k=>k==='generator_shutdown').length)).toBe(2);
 await page.locator('#rift-next').click();await expect.poll(async()=>(await saved()).room).toBe(2);await expect(page.locator('#rift-room-objective')).toBeHidden();
});
test('generator protocol validates hazard links and mobile layout',async({page})=>{
 await page.goto('/abyss/rift?scenario=generators');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();
 expect(await page.evaluate(data=>{const valid=v=>{try{window.RiftProtocol.validate(v,'GET');return true;}catch(_){return false;}};const flag=structuredClone(data);flag.run.level.rooms[1].hazards[0].disabled=true;const link=structuredClone(data);link.run.level.rooms[1].hazards[0].generator_id='missing';const duplicate=structuredClone(data);duplicate.run.level.rooms[1].hazards[0].generator_id=duplicate.run.level.rooms[1].hazards[1].generator_id;const count=structuredClone(data);count.run.room_objective.collected=1;return [data,flag,link,duplicate,count].map(valid);},data)).toEqual([true,false,false,false,false]);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
