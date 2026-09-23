const {test,expect}=require('@playwright/test');
for(const rebound of [false,true])test((rebound?'rebound':'Space')+' jump vaults low cover but tall cover blocks movement',async({page})=>{
 if(rebound)await page.addInitScript(()=>localStorage.setItem('riftBindings',JSON.stringify({version:1,bindings:{jump:['KeyV']}})));
 await page.goto('/abyss/rift?scenario=vault-cover');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.keyboard.down('KeyD');try{await expect.poll(async()=>(await saved()).player.x,{intervals:[20]}).toBeGreaterThan(270);await page.waitForTimeout(300);}finally{await page.keyboard.up('KeyD');}
 expect((await saved()).player.x).toBeLessThan(300);await expect(page.locator('#rift-terrain-hint')).toHaveAttribute('data-kind','low');await expect(page.locator('#rift-terrain-hint')).toContainText(rebound?'Move + V to vault':'Move + Space / K to vault');
 if(!rebound)await page.locator('#rift-viewport').screenshot({path:'test-results/low-cover-vault-hint.png'});
 await page.keyboard.down('KeyD');await page.keyboard.press(rebound?'KeyV':'Space');try{await expect.poll(async()=>(await saved()).player.x,{intervals:[20]}).toBeGreaterThan(395);}finally{await page.keyboard.up('KeyD');}
 await expect.poll(async()=>(await saved()).player.jump).toBe(0);expect((await saved()).stats.jumps).toBe(1);
 await page.keyboard.down('KeyD');try{await expect.poll(async()=>(await saved()).player.x,{intervals:[20]}).toBeGreaterThan(610);}finally{await page.keyboard.up('KeyD');}
 await expect(page.locator('#rift-terrain-hint')).toHaveAttribute('data-kind','tall');await expect(page.locator('#rift-terrain-hint')).toHaveText('Tall cover · Walk around · Blocks projectiles');
 await page.keyboard.down('KeyD');await page.keyboard.down(rebound?'KeyV':'Space');await page.waitForTimeout(1200);await page.keyboard.up('KeyD');await page.keyboard.up(rebound?'KeyV':'Space');expect((await saved()).player.x).toBeLessThan(650);if(!rebound)await page.locator('#rift-viewport').screenshot({path:'test-results/tall-cover-hint.png'});
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
