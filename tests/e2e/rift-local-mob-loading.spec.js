const {test,expect}=require('@playwright/test');
test('idle fighter preview defers companion artwork even when the equipped build needs it',async({page})=>{
 const paths=[];page.on('request',r=>{if(/rift_mobs(?:_row\d)?\.png/.test(r.url()))paths.push(new URL(r.url()).pathname);});
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 const build=(await(await page.request.get('/api/abyss/rift')).json()).build;expect(build.pets>0||[...(build.skills||[]),...(build.signatures||[]),build.ultimate].some(skill=>skill?.kind==='pack')).toBe(true);expect(paths).toEqual([]);
});
test('local and shared species plus saved waves prepare only their required rows',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();const paths=[];
 page.on('request',r=>{if(/rift_mobs_row\d\.png/.test(r.url()))paths.push(new URL(r.url()).pathname);});
 await page.evaluate(()=>RiftRenderer.prepareRun({level:{id:1,region:0},enemies:[{kind:'archer'}],encounter_plan:[[{name:'Goblin',kind:'goblin',art_key:'mob:goblin'}],[],[{kind:'boss'}]],room_objective:{waves:[[{kind:'spore'}]]}}));
 expect(paths.sort()).toEqual([0,1,3,5].map(row=>'/static/rift_mobs_row'+row+'.png'));
});
for(const dependency of ['pets','skill','signature','ultimate','enemy','saved'])test('wolf artwork prepared for '+dependency,async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();const paths=[];
 page.on('request',r=>{if(/rift_mobs_row4\.png/.test(r.url()))paths.push(r.url());});
 await page.evaluate(async dependency=>{
  const build={class:'vanguard'};if(dependency==='pets')build.pets=1;if(dependency==='skill')build.skills=[{kind:'pack'}];if(dependency==='signature')build.signatures=[{kind:'pack'}];if(dependency==='ultimate')build.ultimate={kind:'pack'};
  await RiftRenderer.prepareRun({level:{id:1,region:0},build,enemies:dependency==='enemy'?[{name:'dragon',kind:'boss',art_key:'mob:dragon',shot:'pack'}]:[],projectiles:dependency==='saved'?[{kind:'pack'}]:[]});
 },dependency);expect(paths).toHaveLength(1);
});
test('legacy fighting saves prepare every local fallback row',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();const paths=[];
 page.on('request',r=>{if(/rift_mobs_row\d\.png/.test(r.url()))paths.push(new URL(r.url()).pathname);});
 await page.evaluate(()=>RiftRenderer.prepareRun({status:'fighting',level:{id:1,region:0},enemies:[]}));
 expect(paths.sort()).toEqual([0,1,2,3,4,5].map(row=>'/static/rift_mobs_row'+row+'.png'));
});

test('invalid row dimensions recover with a fresh URL and concurrent retries share one load',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 const urls=[];let invalid=true;
 await page.route('**/static/rift_mobs_row3.png*',route=>{urls.push(route.request().url());return invalid?route.fulfill({contentType:'image/svg+xml',body:'<svg xmlns="http://www.w3.org/2000/svg" width="1" height="1"/>'}):route.continue();});
 const error=await page.evaluate(async()=>{try{await RiftRenderer.prepareCreatures([{kind:'boss'}]);return '';}catch(error){return error.message;}});expect(error).toContain('local creature artwork');
 invalid=false;await page.evaluate(()=>Promise.all([RiftRenderer.prepareCreatures([{kind:'boss'}]),RiftRenderer.prepareCreatures([{kind:'boss'}])]));
 expect(urls).toHaveLength(2);expect(urls[1]).toMatch(/[?&]retry=1$/);
});

test('starting a companion build waits for wolf decode before play',async({page})=>{
 await page.addInitScript(()=>{
  const decode=HTMLImageElement.prototype.decode;window.wolfDecodes=0;const gate=new Promise(resolve=>window.releaseWolf=resolve);
  HTMLImageElement.prototype.decode=function(){if(!this.src.includes('rift_mobs_row4.png'))return decode.call(this);window.wolfDecodes++;return gate.then(()=>decode.call(this));};
 });
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();
 await expect.poll(()=>page.evaluate(()=>window.wolfDecodes)).toBe(1);await expect(page.locator('#rift-overlay')).toBeVisible();
 await page.evaluate(()=>window.releaseWolf());await expect(page.locator('#rift-overlay')).toBeHidden();
});
