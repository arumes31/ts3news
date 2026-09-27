const {test,expect}=require('@playwright/test');
test('free practice keeps hit totals readable without announcing each basic hit',async({page})=>{
 await page.goto('/abyss/rift?practice=skills');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift?practice=skills')).json()).run;
 const initial=await saved(),target=initial.enemies[0];
 await page.keyboard.down('KeyD');try{await expect.poll(async()=>(await saved()).player.x,{intervals:[20]}).toBeGreaterThan(target.x-60);}finally{await page.keyboard.up('KeyD');}
 await page.evaluate(()=>{window.practiceMessages=[];const node=document.getElementById('rift-practice-announcement');window.practiceObserver=new MutationObserver(()=>practiceMessages.push(node.textContent));practiceObserver.observe(node,{childList:true,subtree:true,characterData:true});});
 await page.keyboard.down('KeyJ');try{await expect.poll(async()=>(await saved()).practice.hits,{timeout:10000}).toBeGreaterThanOrEqual(6);}finally{await page.keyboard.up('KeyJ');}
 expect(await page.evaluate(()=>{practiceObserver.disconnect();return practiceMessages;})).toEqual([]);
 await expect(page.locator('#rift-practice-progress')).toContainText('basic target hits');await expect(page.locator('#rift-practice-announcement')).toHaveText('Free practice · No time limit');
});
