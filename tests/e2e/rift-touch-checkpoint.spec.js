const {test,expect}=require('@playwright/test');
test.use({hasTouch:true,isMobile:true});
for(const viewport of [{width:320,height:568},{width:390,height:844},{width:844,height:390}])for(const action of ['next','exit'])test('checkpoint '+action+' at '+viewport.width+'x'+viewport.height,async({page})=>{
 await page.setViewportSize(viewport);await page.goto('/abyss/rift?scenario=checkpoint&condition=wounded');
 await page.locator('#rift-auto').uncheck();await page.locator('#rift-start').tap();
 const panel=page.locator('#rift-room-actions');await expect(panel).toBeVisible();
 const bounds=await panel.evaluate(node=>{const r=node.getBoundingClientRect(),v=document.querySelector('#rift-viewport').getBoundingClientRect();return {left:r.left>=v.left,right:r.right<=v.right,top:r.top>=v.top,bottom:r.bottom<=v.bottom};});
 expect(bounds).toEqual({left:true,right:true,top:true,bottom:true});
 expect(await panel.evaluate(node=>node.getBoundingClientRect().bottom<=document.querySelector('.rift-touch').getBoundingClientRect().top)).toBe(true);
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 if(action==='next'){await panel.scrollIntoViewIfNeeded();await page.evaluate(()=>window.scrollBy(0,-100));await panel.screenshot({path:test.info().outputPath('checkpoint.png')});}
 const button=page.locator('#rift-'+action);await button.scrollIntoViewIfNeeded();
 expect(await button.evaluate(node=>{const r=node.getBoundingClientRect();return node.contains(document.elementFromPoint(r.x+r.width/2,r.y+r.height/2));})).toBe(true);
 expect((await button.boundingBox()).height).toBeGreaterThanOrEqual(44);
 const response=page.waitForResponse(r=>r.url().endsWith('/api/abyss/rift')&&r.request().postDataJSON()?.kind===action);
 await button.tap();expect((await response).ok()).toBe(true);
 await expect(panel).toBeHidden();
 if(action==='next')await page.locator('#rift-pause').tap();
});
