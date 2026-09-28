const {test,expect}=require('@playwright/test');
test('Escape during a pending resume preserves the pause request',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 let release;const gate=new Promise(resolve=>release=resolve);let resumes=0,pauses=0;
 await page.route('**/api/abyss/rift',async route=>{
  if(route.request().method()!=='POST')return route.continue();
  const kind=route.request().postDataJSON().kind;
  if(kind==='resume'){resumes++;await gate;}if(kind==='pause')pauses++;
  await route.continue();
 });
 await page.locator('#rift-start').click();await expect.poll(()=>resumes).toBe(1);
 await page.keyboard.press('Escape');release();
 await expect(page.locator('#rift-overlay-title')).toHaveText('A moment by the lantern.');
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 expect(run.paused).toBe(true);expect(run.room).toBe(0);expect(pauses).toBe(1);
});
