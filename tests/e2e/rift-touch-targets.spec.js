const {test,expect}=require('@playwright/test');

test.use({hasTouch:true,isMobile:true});
for(const viewport of [{width:320,height:568},{width:390,height:844},{width:844,height:390}])test('combat touch targets at '+viewport.width+'x'+viewport.height,async({page})=>{
 await page.setViewportSize(viewport);
 await page.goto('/abyss/rift?practice=skills');await expect(page.locator('#rift-start')).toBeEnabled();
 if(viewport.width<viewport.height)await expect(page.locator('#rift-orientation-hint')).toBeVisible();else await expect(page.locator('#rift-orientation-hint')).toBeHidden();
 await page.locator('#rift-start').tap();await expect(page.locator('#rift-overlay')).toBeHidden();
 const sizes=await page.locator('.rift-stage-top button,.rift-actionbar button,.rift-signatures button,.rift-touch button').evaluateAll(nodes=>nodes.filter(node=>node.getClientRects().length).map(node=>{const r=node.getBoundingClientRect();return {id:node.id||node.dataset.bind||node.dataset.move||node.textContent,width:r.width,height:r.height};}));
 expect(sizes.length).toBeGreaterThan(10);
 for(const direction of ['left','right','up','down'])await expect(page.locator('[data-move='+direction+']')).toBeVisible();
 expect(sizes.filter(r=>r.width<44||r.height<44)).toEqual([]);
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await page.locator('#rift-pause').tap();await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
 const start=await page.locator('#rift-start').boundingBox();expect(start.width).toBeGreaterThanOrEqual(44);expect(start.height).toBeGreaterThanOrEqual(44);
 await page.locator('#rift-start').tap();await expect(page.locator('#rift-overlay')).toBeHidden();
 if(viewport.width<viewport.height){await expect(page.locator('#rift-orientation-hint')).toBeVisible();await page.locator('#rift-orientation-hint').screenshot({path:test.info().outputPath('orientation-hint.png')});}
 await page.locator('.rift-game').screenshot({path:test.info().outputPath('touch-controls.png')});
});
