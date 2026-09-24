const {test,expect}=require('@playwright/test');
test.use({hasTouch:true,isMobile:true});
for(const width of [320,390])for(const large of [false,true])test('readable combat labels '+width+' large '+large,async({page})=>{
 await page.setViewportSize({width,height:844});
 await page.goto('/abyss/rift?practice=skills&subclass=elementalist');await expect(page.locator('#rift-start')).toBeEnabled();
 if(large){await page.locator('.rift-settings > summary').tap();await page.locator('#rift-large-action-bar').check();}
 await page.locator('#rift-start').tap();
 const labels=page.locator('.rift-actionbar .rift-action-label,.rift-signatures .rift-action-label');
 expect(await labels.count()).toBeGreaterThan(6);
 const sizes=await labels.evaluateAll(nodes=>nodes.map(node=>{const r=node.getBoundingClientRect(),b=node.closest('button').getBoundingClientRect();return {name:node.textContent,font:parseFloat(getComputedStyle(node).fontSize),fits:r.left>=b.left&&r.right<=b.right&&r.top>=b.top&&r.bottom<=b.bottom&&node.scrollWidth<=node.clientWidth+1};}));
 for(const label of sizes){expect(label.font,label.name).toBeGreaterThanOrEqual(14);expect(label.fits,label.name).toBe(true);}
 const secondary=await page.locator('.rift-actionbar small,.rift-signatures small,.rift-signatures .rift-role-mark').evaluateAll(nodes=>nodes.filter(n=>n.getClientRects().length).map(n=>parseFloat(getComputedStyle(n).fontSize)));
 expect(Math.min(...secondary)).toBeGreaterThanOrEqual(12);
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await page.locator('#rift-signatures').screenshot({path:test.info().outputPath('ability-labels.png')});
 await page.locator('#rift-pause').tap();
});
