const {test,expect}=require('@playwright/test');
test('campaign and bestiary art reuse renderer atlas versions',async({page})=>{
 const requested=[];page.on('request',request=>{if(request.resourceType()==='image')requested.push(request.url());});
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 const required=await page.evaluate(()=>{
  const root=document.getElementById('rift-app');
  return [...window.RiftRenderer.getCriticalAtlasKeys().map(key=>key==='props'?RiftPropSections.panels[0].url:key==='regions'?RiftRegionSections.regions[0].url:root.dataset[key]),...window.RiftBestiary.assets.map(path=>window.RiftBestiary.assetURL(path)),...RiftRegionSections.regions.map(region=>region.url)].map(url=>new URL(url,location.href).href);
 });
 expect(new Set(required).size).toBe(16);
 await page.locator('details').filter({has:page.locator('#rift-monsters')}).locator(':scope > summary').click();
 await expect(page.locator('#rift-monsters > article')).not.toHaveCount(0);
 await page.locator('#rift-monsters > article').first().getByRole('button',{name:/Inspect/}).click();
 await expect(page.locator('#rift-monster-preview')).toBeVisible();
 const referenced=await page.evaluate(()=>{
  const nodes=[...document.querySelectorAll('.rift-monster-art,.rift-level-art')];
  return nodes.filter(node=>node.style.backgroundImage).map(node=>new URL(node.style.backgroundImage.slice(5,-2),location.href).href);
 });
 expect(referenced.length).toBeGreaterThan(0);
 for(const url of referenced)expect(required).toContain(url);
 expect(requested.some(url=>url.includes('rift_props.png'))).toBe(false);
 // Give CSS background loading a rendering opportunity before checking requests.
 await page.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));
 for(const url of new Set(required)){
  expect(new URL(url).searchParams.get('v')).toBeTruthy();
  const matches=requested.filter(value=>new URL(value).pathname===new URL(url).pathname);
  // Offscreen mission thumbnails remain lazy; every requested atlas uses one version.
  if(required.indexOf(url)<7||matches.length)expect(matches,url).toEqual([url]);
 }
});
