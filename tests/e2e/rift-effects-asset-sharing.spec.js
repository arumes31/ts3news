const {test,expect}=require('@playwright/test');

for(const mode of ['fresh','saved'])test('skill icons and canvas share one effects sheet: '+mode,async({page,context})=>{
 const requested=[];page.on('request',r=>{if(new URL(r.url()).pathname==='/static/rift_effects.png')requested.push(r.url());});
 const cdp=await context.newCDPSession(page);await cdp.send('Network.setCacheDisabled',{cacheDisabled:true});
 await page.goto('/abyss/rift'+(mode==='saved'?'?scenario=visual':''));await expect(page.locator('#rift-start')).toBeEnabled();
 if(mode==='fresh'){await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();}
 const expected=await page.locator('#rift-app').getAttribute('data-effects');
 const icon=page.locator('.rift-skill-icon').first();await expect(icon).toBeAttached();
 await expect.poll(()=>requested.length).toBeGreaterThanOrEqual(1);
 expect(requested).toEqual([new URL(expected,page.url()).href]);
 await expect(icon).toHaveCSS('background-image',`url("${new URL(expected,page.url()).href}")`);
});
