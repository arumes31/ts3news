const {test,expect}=require('@playwright/test');
test.use({viewport:{width:1280,height:900},locale:'en-US',timezoneId:'UTC',reducedMotion:'reduce'});
for(const [seed,level,room] of [['ruins-v1',1,0],['forge-v1',15,1],['boss-v1',100,2]])test(seed+' reproduces the complete visual snapshot',async({page},testInfo)=>{
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.addInitScript(()=>{localStorage.setItem('riftReducedMotion','true');localStorage.setItem('rift-auto','false');});
 const url='/abyss/rift?scenario=visual&seed='+seed+'&level='+level+'&room='+room;
 await page.goto(url);await expect(page.locator('#rift-start')).toBeEnabled();
 const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const first=await read();expect(first.level.id).toBe(level);expect(first.room).toBe(room);expect(first.paused).toBe(true);
 await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();expect(await read()).toEqual(first);
 await page.evaluate(async()=>{await RiftRenderer.ready;await document.fonts.ready;});
 await page.locator('#rift-viewport').screenshot({path:testInfo.outputPath(seed+'.png'),animations:'disabled'});
 expect(errors).toEqual([]);
});
