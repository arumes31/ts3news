const {test,expect}=require('@playwright/test');

test('circular court has two keyboard-accessible hazard-free routes and saved bounds',async({page},info)=>{
 test.setTimeout(90000);
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.goto('/abyss/rift?scenario=circular-court');await expect(page.locator('#rift-start')).toHaveText('Resume expedition');await page.locator('#rift-auto').uncheck();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const before=await saved(),held=new Set();
 async function keys(next){for(const k of [...held])if(!next.has(k)){await page.keyboard.up(k);held.delete(k);}for(const k of next)if(!held.has(k)){await page.keyboard.down(k);held.add(k);}}
 async function walk(x,y){try{await expect.poll(async()=>{const run=await saved(),p=run.player,next=new Set();if(Math.abs(x-p.x)>5)next.add(x>p.x?'d':'a');if(Math.abs(y-p.y)>5)next.add(y>p.y?'s':'w');await keys(next);return next.size===0;},{timeout:15000,intervals:[30]}).toBe(true);}finally{await keys(new Set());}}
 await expect(page.locator('#rift-minimap [data-kind="round"]')).toHaveCount(1);
 await page.locator('#rift-viewport').screenshot({path:info.outputPath('court-desktop.png'),style:'#rift-overlay {visibility:hidden!important}'});
 await page.locator('#rift-start').click();
 for(const [x,y] of [[400,350],[1200,350],[1440,402.5],[1200,450],[1000,465],[600,465],[400,450],[160,410]])await walk(x,y);
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const after=await saved();expect(after.player.hp).toBe(before.player.hp);expect(after.stats.hazard_contacts||0).toBe(before.stats.hazard_contacts||0);
 await page.setViewportSize({width:390,height:844});await page.locator('#rift-viewport').screenshot({path:info.outputPath('court-mobile.png'),style:'#rift-overlay {visibility:hidden!important}'});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toHaveText('Resume expedition');expect((await saved()).level.rooms[0].round).toEqual(before.level.rooms[0].round);expect(errors).toEqual([]);
});

test('circular campaign preview shows its actual boundary and route guidance',async({page})=>{
 await page.goto('/abyss/rift?mission=2');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-mission-preview > summary').click();
 await expect(page.locator('#rift-room-previews [data-terrain="round"]')).toHaveCount(1);await expect(page.locator('#rift-room-previews')).toContainText('upper and lower routes avoid the floor hazards');
});
