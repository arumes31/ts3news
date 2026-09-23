const {test,expect}=require('@playwright/test');
for(const forced of [false,true])test('hazard legend maps all renderer patterns without color, forced='+forced,async({page})=>{
 await page.emulateMedia({forcedColors:forced?'active':'none'});await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-field-guide > summary').click();
 const legend=page.locator('#rift-hazard-legend');await expect(legend).toBeVisible();await expect(legend.locator('dt')).toHaveCount(7);await expect(legend).toContainText('Dashed border');await expect(legend).toContainText('Solid border');await expect(legend).toContainText('SAFE');await expect(legend).toContainText('OFF');
 const samples=await legend.locator('canvas').evaluateAll(nodes=>nodes.map(c=>({kind:c.dataset.kind,image:c.toDataURL(),profile:RiftRenderer.getHazardPatternInfo(c.dataset.kind).label,monochrome:(()=>{const d=c.getContext('2d').getImageData(0,0,c.width,c.height).data;for(let i=0;i<d.length;i+=4)if(d[i]!==d[i+1]||d[i]!==d[i+2])return false;return true;})()})));
 expect(new Set(samples.map(s=>s.image)).size).toBe(7);for(const sample of samples){expect(sample.monochrome).toBe(true);await expect(legend).toContainText(sample.profile);}
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await legend.screenshot({path:'test-results/hazard-legend'+(forced?'-forced':'')+'.png'});
});
