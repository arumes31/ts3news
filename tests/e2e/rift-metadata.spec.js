const {test,expect}=require('@playwright/test');
test('public metadata is split, cookie-free and conditionally cached',async({page})=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));
 const metadataRequest=page.waitForRequest(r=>new URL(r.url()).pathname==='/api/abyss/rift/metadata');
 const privateResponse=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/abyss/rift'&&r.request().headers()['x-rift-metadata']==='separate');
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const request=await metadataRequest;expect((await request.allHeaders()).cookie).toBeUndefined();
 const response=await privateResponse,privateBody=await response.json();
 for(const key of ['levels','rooms','bestiary','class_names','class_options','rarities'])expect(privateBody[key]).toBeUndefined();
 expect(privateBody.build.name).toBeTruthy();expect(privateBody.run).toBeTruthy();
 const initial=await page.request.get('/api/abyss/rift/metadata');expect(initial.ok()).toBe(true);const metadata=await initial.json();
 expect(Object.keys(metadata).sort()).toEqual(['bestiary','class_names','class_options','levels','rarities','rooms']);expect(metadata.levels).toHaveLength(100);
 const cached=await page.request.get('/api/abyss/rift/metadata',{headers:{'If-None-Match':initial.headers().etag}});expect(cached.status()).toBe(304);expect((await cached.body()).length).toBe(0);
 await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();expect(errors).toEqual([]);
});
for(const failure of ['unavailable','malformed'])test('metadata '+failure+' can be retried without starting an expedition',async({page})=>{
 let starts=0;page.on('request',r=>{if(r.method()==='POST'&&r.postDataJSON()?.kind==='start')starts++;});
 await page.route('**/api/abyss/rift/metadata',route=>route.fulfill({status:failure==='unavailable'?503:200,contentType:'application/json',body:failure==='unavailable'?'unavailable':'{"levels":'}));
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toHaveText('Retry loading');expect(starts).toBe(0);
 await page.unroute('**/api/abyss/rift/metadata');await page.locator('#rift-start').click();
 await expect(page.locator('#rift-start')).not.toHaveText('Retry loading');await expect(page.locator('#rift-start')).toBeEnabled();expect(starts).toBe(0);
});
