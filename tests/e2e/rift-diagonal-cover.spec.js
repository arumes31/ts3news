const {test,expect}=require('@playwright/test');
for(const level of [3,13])test('diagonal ranged cover renders and reloads in mission '+level,async({page},testInfo)=>{
 await page.goto('/abyss/rift?scenario=checkpoint&level='+level);await expect(page.locator('#rift-start')).toBeEnabled();
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run,cover=run.level.rooms[0].high_cover;
 expect(cover).toHaveLength(3);expect(run.level.tactic).toContain('diagonal stone pillars');
 const map=page.locator('#rift-minimap');await expect(map.locator('[data-kind=stone]')).toHaveCount(3);
 const positions=await map.locator('[data-kind=stone]').evaluateAll(nodes=>nodes.map(node=>({x:Number(node.getAttribute('x')),y:Number(node.getAttribute('y'))})));
 expect(positions.map(p=>p.x)).toEqual(cover.map(p=>Math.round(p.x*2)/10));
 expect(Math.sign(positions[2].y-positions[0].y)).toBe(level===3?1:-1);
 await page.evaluate(async run=>{await window.RiftRenderer.ready;run.player.x=700;run.paused=true;window.RiftDisplay.cameraSmooth=false;document.querySelector('#rift-overlay').hidden=true;window.RiftRenderer.snapshot(run,true);},run);
 await page.waitForTimeout(150);await page.locator('#rift-canvas').screenshot({path:testInfo.outputPath('diagonal-stone-pillars.png')});
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();
 expect((await(await page.request.get('/api/abyss/rift')).json()).run.level.rooms[0].high_cover).toEqual(cover);
 await page.setViewportSize({width:390,height:844});await expect(map.locator('[data-kind=stone]')).toHaveCount(3);expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await map.evaluate(node=>node.scrollIntoView({block:'center',behavior:'instant'}));await map.screenshot({path:testInfo.outputPath('diagonal-minimap-mobile.png')});
});
