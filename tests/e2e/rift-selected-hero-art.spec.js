const {test,expect}=require('@playwright/test');
for(const [subclass,atlas,other] of [['vanguard','a','b'],['oracle','b','a']])test('loads only selected hero atlas '+subclass,async({page})=>{
 const requests=[];page.on('request',r=>{if(/rift_heroes_[ab]\.png/.test(r.url()))requests.push(r.url());});
 await page.goto('/abyss/rift?subclass='+subclass);await expect(page.locator('#rift-start')).toBeEnabled();
 expect(requests.filter(url=>url.includes('rift_heroes_'+atlas+'.png'))).toHaveLength(1);
 expect(requests.filter(url=>url.includes('rift_heroes_'+other+'.png'))).toHaveLength(0);
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.locator('#rift-pause').click();
});

test('saved build and current build wait for both hero atlases',async({page})=>{
 let release;const gate=new Promise(resolve=>release=resolve);let requested=0;
 await page.route('**/static/rift_heroes_b.png*',async route=>{requested++;await gate;await route.continue();});
 await page.route('**/api/abyss/rift',async route=>{const response=await route.fetch();const data=await response.json();data.run.build.class='oracle';data.run.player.kind='oracle';await route.fulfill({response,json:data});});
 await page.goto('/abyss/rift?scenario=visual&subclass=vanguard',{waitUntil:'domcontentloaded'});
 await expect.poll(()=>requested).toBe(1);await expect(page.locator('#rift-start')).toBeDisabled();
 release();await expect(page.locator('#rift-start')).toBeEnabled();
 await page.evaluate(()=>Promise.all([RiftRenderer.prepareBuild({class:'warden'}),RiftRenderer.prepareBuild({class:'oracle'})]));expect(requested).toBe(1);
});
