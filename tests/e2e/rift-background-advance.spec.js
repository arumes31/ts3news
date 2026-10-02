const {test,expect}=require('@playwright/test');
for(const pending of [false,true])test('background visibility '+(pending?'during pending advance':'during countdown')+' persists pause without repeated banking',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;const initial=await read();
 let release;const gate=new Promise(resolve=>release=resolve);const writes=[];let advancing=false;
 await page.route('**/api/abyss/rift',async route=>{if(route.request().method()==='POST'){const kind=route.request().postDataJSON().kind;writes.push(kind);if(pending&&kind==='advance'){advancing=true;await gate;}}await route.continue();});
 await page.locator('#rift-start').click();await expect(page.locator('#rift-transition')).toContainText('Next:');if(pending)await expect.poll(()=>advancing).toBe(true);
 try{await page.evaluate(()=>{Object.defineProperty(document,'hidden',{configurable:true,value:true});Object.defineProperty(document,'visibilityState',{configurable:true,value:'hidden'});document.dispatchEvent(new Event('visibilitychange'));});}finally{release();}
 await expect.poll(async()=>(await read()).paused).toBe(true);const paused=await read(),savedWrites=[...writes];expect(paused.room).toBe(initial.room+(pending?1:0));expect(paused.banked_gold).toBe(pending?30:0);expect(writes.filter(k=>k==='advance')).toHaveLength(pending?1:0);
 await page.waitForTimeout(1800);expect(await read()).toEqual(paused);expect(writes).toEqual(savedWrites);
 await page.evaluate(()=>{Object.defineProperty(document,'hidden',{configurable:true,value:false});Object.defineProperty(document,'visibilityState',{configurable:true,value:'visible'});document.dispatchEvent(new Event('visibilitychange'));});await page.waitForTimeout(1400);expect(await read()).toEqual(paused);expect(writes).toEqual(savedWrites);await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
});
