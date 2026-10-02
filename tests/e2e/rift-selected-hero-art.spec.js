const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path');
for(const [subclass,atlas,row] of [['vanguard','a',0],['oracle','b',0],['berserker','a',1],['alchemist','b',5]])test('loads only selected hero row '+subclass,async({page})=>{
 const requests=[];page.on('request',r=>{if(/rift_heroes_[ab](?:_row\d)?\.png/.test(r.url()))requests.push(new URL(r.url()).pathname);});
 await page.goto('/abyss/rift?subclass='+subclass);await expect(page.locator('#rift-start')).toBeEnabled();
 expect(requests).toEqual(['/static/rift_heroes_'+atlas+'_row'+row+'.png']);
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.locator('#rift-pause').click();
});

test('incorrect section dimensions are rejected and can retry',async({page})=>{
 await page.goto('/abyss/rift?subclass=vanguard');await expect(page.locator('#rift-start')).toBeEnabled();
 let attempts=0;
 await page.route('**/static/rift_heroes_b_row5.png*',async route=>{attempts++;if(attempts===1)await route.fulfill({contentType:'image/png',body:fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift_heroes_b.png'))});else await route.continue();});
 expect(await page.evaluate(()=>RiftRenderer.prepareBuild({class:'alchemist'}).then(()=>'',error=>error.message))).toContain('Could not load character artwork');
 await page.evaluate(()=>RiftRenderer.prepareBuild({class:'alchemist'}));expect(attempts).toBe(2);
});

test('missing manifest keeps artwork readiness blocked with recovery',async({page})=>{
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.route('**/static/rift_hero_sections.js*',route=>route.fulfill({contentType:'application/javascript',body:'window.RiftHeroSections={version:1};'}));
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toHaveText('Reload artwork');
 await expect(page.locator('#rift-start')).toHaveAttribute('data-artwork-retry','true');expect(errors).toEqual([]);
});

for(const [saved,atlas,row,alias] of [['oracle','b',0,'warden'],['berserker','a',1,'berserker']])test('saved and current build wait for separate rows: '+saved,async({page})=>{
 let release;const gate=new Promise(resolve=>release=resolve);let requested=0;
 await page.route('**/static/rift_heroes_'+atlas+'_row'+row+'.png*',async route=>{requested++;await gate;await route.continue();});
 await page.route('**/api/abyss/rift',async route=>{const response=await route.fetch();const data=await response.json();data.run.build.class=saved;data.run.player.kind=saved;await route.fulfill({response,json:data});});
 await page.goto('/abyss/rift?scenario=visual&subclass=vanguard',{waitUntil:'domcontentloaded'});
 await expect.poll(()=>requested).toBe(1);await expect(page.locator('#rift-start')).toBeDisabled();
 release();await expect(page.locator('#rift-start')).toBeEnabled();
 await page.evaluate(([saved,alias])=>Promise.all([RiftRenderer.prepareBuild({class:alias}),RiftRenderer.prepareBuild({class:saved})]),[saved,alias]);expect(requested).toBe(1);
});

test('hero artwork failure remains retryable and concurrent retries share a request',async({page})=>{
 await page.goto('/abyss/rift?subclass=vanguard');await expect(page.locator('#rift-start')).toBeEnabled();
 let attempts=0;await page.route('**/static/rift_heroes_b_row0.png*',async route=>{attempts++;if(attempts===1)await route.abort();else await route.continue();});
 expect(await page.evaluate(()=>RiftRenderer.prepareBuild({class:'oracle'}).then(()=>'',error=>error.message))).toContain('Could not load character artwork');
 await page.evaluate(()=>Promise.all([RiftRenderer.prepareBuild({class:'oracle'}),RiftRenderer.prepareBuild({class:'warden'})]));
 expect(attempts).toBe(2);
});
