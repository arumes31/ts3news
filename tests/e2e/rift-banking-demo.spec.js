const {test,expect}=require('@playwright/test');
test('banking demonstration requires checkpoint confirmation and preserves its receipt',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();const campaign=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.locator('#rift-practice-links a[href$="practice=banking"]').click();await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-practice-instructions')).toContainText('no gold or gear value');await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 const read=async()=>(await(await page.request.get('/api/abyss/rift?practice=banking')).json()).run;
 await expect.poll(async()=>(await read()).clock,{timeout:10000}).toBeGreaterThan(5);expect(await page.evaluate(()=>RiftRenderer.lastPickupRadius?.radius)).toBe(65);
 await page.locator('#rift-viewport').screenshot({path:'test-results/banking-demo.png'});
 const drops=(await read()).drops;
 for(const drop of drops){
  const current=await read();const key=drop.y<current.player.y?'w':'s';if(Math.abs(drop.y-current.player.y)>15){await page.keyboard.down(key);try{await expect.poll(async()=>Math.abs((await read()).player.y-drop.y),{intervals:[20]}).toBeLessThan(20);}finally{await page.keyboard.up(key);}}
  await page.keyboard.down('d');try{await expect.poll(async()=>(await read()).drops.find(d=>d.id===drop.id).collected,{intervals:[20]}).toBe(true);}finally{await page.keyboard.up('d');}
 }
 await expect(page.locator('[data-practice-action="practice_bank"]')).toHaveAttribute('aria-disabled','true');await page.keyboard.down('d');try{await expect.poll(async()=>(await read()).practice.checkpoint_ready).toBe(true);}finally{await page.keyboard.up('d');}
 await expect(page.locator('#rift-practice-bank-receipt')).toContainText('3 unbanked');await page.locator('[data-practice-action="practice_bank"]').click();await expect(page.locator('#rift-practice-bank-receipt')).toContainText('3 secured');await expect(page.locator('#rift-practice-tool-status')).toContainText('Practice receipt confirmed');
 await expect(page.locator('#rift-practice-progress')).toHaveText('Drill complete');const done=await read();expect(done.drops.every(d=>d.banked)).toBe(true);expect(done.gold).toBe(0);expect(done.drops.every(d=>d.gold===0&&!d.gear&&!d.needs_gear)).toBe(true);
 await page.reload();await expect(page.locator('#rift-practice-bank-receipt')).toContainText('3 secured');await expect(page.locator('#rift-practice-progress')).toHaveText('Drill complete');await page.locator('#rift-practice-reset').click();await expect.poll(async()=>(await read()).drops.filter(d=>d.collected).length).toBe(0);expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(campaign);
});
