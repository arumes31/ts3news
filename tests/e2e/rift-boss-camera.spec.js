const {test,expect}=require('@playwright/test');
for(const paused of [true,false])for(const [label,playerX,bossX] of [['left edge',180,35],['right edge',1450,1565],['behind player',850,350]])test('frame large boss '+label+' paused='+paused,async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const {run}=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(({run,playerX,bossX,paused})=>{run.status='fighting';run.paused=paused;run.player.x=playerX;run.enemies=[{...run.enemies[0],id:'large-boss',kind:'boss',x:bossX,y:410,hp:100,max_hp:100}];run.events=[];window.RiftRenderer.snapshot(run,true);},{run,playerX,bossX,paused});
 await expect.poll(()=>page.evaluate(()=>window.RiftRenderer.cameraFraming?.bosses?.includes('large-boss'))).toBe(true);
 const camera=await page.evaluate(()=>window.RiftRenderer.cameraFraming.x);
 expect(bossX-camera-84).toBeGreaterThanOrEqual(0);expect(bossX-camera+84).toBeLessThanOrEqual(960);
 expect(playerX-camera).toBeGreaterThanOrEqual(64);expect(playerX-camera).toBeLessThanOrEqual(896);
});

test('camera keeps ordinary follow and frames multiple nearby bosses when they fit',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();const {run}=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(run=>{run.paused=true;run.player.x=900;run.enemies=[];window.RiftRenderer.snapshot(run,true);},run);
 await expect.poll(()=>page.evaluate(()=>window.RiftRenderer.cameraFraming?.x)).toBe(550);
 await page.evaluate(run=>{run.paused=true;run.player.x=800;run.enemies=[400,1150].map((x,i)=>({id:'boss-'+i,kind:'boss',x,y:410,hp:100,max_hp:100,facing:1,pose:'idle'}));window.RiftRenderer.snapshot(run,true);},run);
 await expect.poll(()=>page.evaluate(()=>window.RiftRenderer.cameraFraming?.bosses?.length)).toBe(2);
 const x=await page.evaluate(()=>window.RiftRenderer.cameraFraming.x);expect(400-x-84).toBeGreaterThanOrEqual(0);expect(1150-x+84).toBeLessThanOrEqual(960);
 await page.evaluate(()=>document.getElementById('rift-overlay').hidden=true);await page.locator('#rift-canvas').screenshot({path:'test-results/boss-camera-framing.png'});
});
