const {test,expect}=require('@playwright/test');

test('separate objective bonus survives banking, replay, reload and receipt copy',async({page,context})=>{
 await context.grantPermissions(['clipboard-read','clipboard-write']);
 let bank;page.on('request',request=>{if(request.url().includes('/api/abyss/rift')&&request.method()==='POST'){const body=request.postDataJSON();if(body.kind==='advance')bank=body;}});
 await page.goto('/abyss/rift?scenario=objective-rewards');await expect(page.locator('#rift-start')).toBeEnabled();
 await expect(page.locator('#rift-objective-reward-terms')).toContainText('5 gold per completed objective');
 await expect(page.locator('#rift-objective-reward-total')).toHaveText('Objective bonus ready to bank: 40 gold.');
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 expect((await saved()).gold).toBe(30);expect((await saved()).drops[0].gold).toBe(30);
 await page.locator('#rift-start').click();await expect.poll(async()=>(await saved()).level.id).toBe(2);
 await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const result=await saved();expect(result.banked_gold).toBe(70);expect(result.banked_objective_gold).toBe(40);expect(result.last_objectives.reward_gold).toBe(40);
 expect(bank).toBeTruthy();for(let n=0;n<2;n++){const replay=await(await page.request.post('/api/abyss/rift',{data:bank})).json();expect(replay.run.banked_gold).toBe(70);expect(replay.run.banked_objective_gold).toBe(40);}
 const extra=await page.request.post('/api/abyss/rift',{data:{...bank,kind:'bank',revision:result.revision+1,request_id:'extra-objective-bank'}});expect(extra.status()).toBe(409);expect((await saved()).banked_objective_gold).toBe(40);
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();
 await page.locator('#rift-receipt > summary').click();await expect(page.locator('#rift-receipt-breakdown')).toHaveText('Fight loot: 30 gold · Objective bonuses: 40 gold');
 await page.locator('#rift-copy-receipt').click();await expect(page.locator('#rift-receipt-copy-status')).toHaveText('Receipt copied.');expect((await page.evaluate(()=>navigator.clipboard.readText())).replace(/\r\n/g,'\n')).toContain('Fight loot: 30 gold\nObjective bonuses: 40 gold');
 await expect(page.locator('#rift-objectives-last-title')).toContainText('Objective bonus: 40 gold');
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await page.locator('#rift-receipt').screenshot({path:'test-results/objective-reward-receipt.png'});
});

test('bonus protocol rejects inflated or inconsistent receipts and keeps legacy snapshots valid',async({page})=>{
 await page.goto('/abyss/rift?scenario=objective-rewards');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();
 const accepted=await page.evaluate(data=>{const valid=v=>{try{window.RiftProtocol.validate(v,'GET');return true;}catch(_){return false;}};const changed=fn=>{const v=structuredClone(data);fn(v.run);return v;};return [data,changed(r=>r.objectives.reward_per_objective=500),changed(r=>r.objectives.reward_gold=40),changed(r=>r.banked_objective_gold=99),changed(r=>r.objectives.entries.push(r.objectives.entries[0])),changed(r=>{delete r.objectives.reward_per_objective;delete r.objectives.reward_gold;delete r.banked_objective_gold;})].map(valid);},data);expect(accepted).toEqual([true,false,false,false,false,true]);
});
