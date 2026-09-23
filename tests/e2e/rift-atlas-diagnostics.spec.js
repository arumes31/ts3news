const {test,expect}=require('@playwright/test');
for(const enabled of [false,true])test('atlas diagnostics opt-in: '+enabled,async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint'+(enabled?'&riftAtlasDebug=1':''));await expect(page.locator('#rift-start')).toBeEnabled();
 if(!enabled){expect(await page.evaluate(()=>window.RiftRenderer.atlasDiagnostics)).toBeNull();return;}
 await expect.poll(()=>page.evaluate(()=>window.RiftRenderer.atlasDiagnostics?.frames||0)).toBeGreaterThan(0);
 expect(await page.evaluate(()=>window.RiftRenderer.atlasDiagnostics.errors)).toEqual([]);
 const result=await page.evaluate(()=>{
  const run=window.RiftRenderer;const ctx=document.getElementById('rift-canvas').getContext('2d');
  const original=ctx.strokeRect.bind(ctx);window.atlasOutlines=0;ctx.strokeRect=(...args)=>{if(ctx.strokeStyle==='#55e7e2')window.atlasOutlines++;return original(...args);};
  const img=new Image();img.width=16;img.height=16;
  for(let i=0;i<50;i++)run.checkAtlasBounds(img,15,0,4,4);
  return {errors:run.atlasDiagnostics.errors};
 });
 expect(result.errors.length).toBeLessThanOrEqual(32);expect(result.errors.length).toBeGreaterThan(0);
 expect(result.errors[0].source).toEqual([15,0,4,4]);
 await expect.poll(()=>page.evaluate(()=>window.atlasOutlines)).toBeGreaterThan(0);
});
