const {test,expect}=require('@playwright/test');
test('campaign and bestiary art reuse renderer atlas versions',async({page})=>{
 const requested=[];page.on('request',request=>{if(request.resourceType()==='image')requested.push(request.url());});
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 const required=await page.evaluate(()=>{
  const root=document.getElementById('rift-app');
  return [...window.RiftRenderer.getCriticalAtlasKeys().map(key=>key==='props'?document.getElementById('rift-props-asset').href:root.dataset[key]),...window.RiftBestiary.assets.map(path=>window.RiftBestiary.assetURL(path))].map(url=>new URL(url,location.href).href);
 });
 expect(new Set(required).size).toBe(21);
 await page.locator('details').filter({has:page.locator('#rift-monsters')}).locator(':scope > summary').click();
 await expect(page.locator('#rift-monsters > article')).not.toHaveCount(0);
 await page.locator('#rift-monsters > article').first().getByRole('button',{name:/Inspect/}).click();
 await expect(page.locator('#rift-monster-preview')).toBeVisible();
 const referenced=await page.evaluate(()=>{
  const nodes=[...document.querySelectorAll('.rift-monster-art,.rift-level-art')];
  return nodes.map(node=>new URL(node.style.backgroundImage.slice(5,-2),location.href).href);
 });
 expect(referenced.length).toBeGreaterThan(100);
 for(const url of referenced)expect(required).toContain(url);
 // Give CSS background loading a rendering opportunity before checking requests.
 await page.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));
 for(const url of required){
  expect(new URL(url).searchParams.get('v')).toBeTruthy();
  const matches=requested.filter(value=>new URL(value).pathname===new URL(url).pathname);
  expect(matches,url).toEqual([url]);
 }
});
