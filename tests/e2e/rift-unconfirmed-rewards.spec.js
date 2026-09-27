const {test,expect}=require('@playwright/test');
for(const lost of [false,true])test('reward totals wait for '+(lost?'read recovery after a lost attack response':'a confirmed attack response'),async({page})=>{
 await page.goto('/abyss/rift?scenario=hunt');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 const totals=()=>page.evaluate(()=>Object.fromEntries(['rift-gold','rift-banked','rift-loot-count','rift-pending-hud','rift-checkpoint-total'].map(id=>[id,document.getElementById(id).textContent])));const before=await totals();
 let release,observe,confirmed,intercepted=false,posts=0;const held=new Promise(resolve=>release=resolve),pending=new Promise(resolve=>observe=resolve);
 await page.route('**/api/abyss/rift',async route=>{
  if(route.request().method()==='POST'){posts++;const body=route.request().postDataJSON();if(body.kind==='step'&&body.input.attack&&!intercepted){intercepted=true;const response=await route.fetch();confirmed=(await response.json()).run;observe();await held;if(lost)await route.abort();else await route.fulfill({response});return;}}
  await route.continue();
 });
 await page.keyboard.down('KeyJ');await pending;await page.keyboard.up('KeyJ');
 try{expect(confirmed.gold).toBeGreaterThan(Number(before['rift-gold'].replaceAll(',','')));await page.waitForTimeout(400);expect(await totals()).toEqual(before);}finally{release();}
 if(lost){await expect(page.locator('#rift-start')).toHaveText('Recover expedition');expect(await totals()).toEqual(before);const stopped=posts;await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toHaveText('Resume expedition');expect(posts).toBe(stopped);}
 await expect(page.locator('#rift-gold')).toHaveText(confirmed.gold.toLocaleString('en-US'));await expect(page.locator('#rift-banked')).toHaveText(before['rift-banked']);if(!lost)await page.keyboard.press('Escape');
});
