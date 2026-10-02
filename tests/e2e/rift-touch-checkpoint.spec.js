const {test,expect}=require('@playwright/test');
test.use({hasTouch:true,isMobile:true});
for(const viewport of [{width:320,height:568},{width:390,height:844},{width:844,height:390}])for(const action of ['next','exit'])test('checkpoint '+action+' at '+viewport.width+'x'+viewport.height,async({page})=>{
 await page.setViewportSize(viewport);await page.goto('/abyss/rift?scenario=checkpoint&condition=wounded');
 await page.locator('#rift-auto').uncheck();await page.locator('#rift-start').tap();await expect(page.locator('#rift-overlay')).toBeHidden();
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

test('touch-only checkpoint banking and receipt flow',async({page})=>{
 await page.setViewportSize({width:390,height:844});
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.locator('#rift-auto').tap();await expect(page.locator('#rift-auto')).not.toBeChecked();
 await page.locator('#rift-start').tap();await expect(page.locator('#rift-overlay')).toBeHidden();await expect(page.locator('#rift-room-actions')).toBeVisible();
 await expect(page.locator('#rift-checkpoint-total')).toHaveText('30 gold · 1 item ready to bank');
 await page.locator('#rift-exit').tap();await expect(page.locator('#rift-banked')).toHaveText('30 gold · 1 item');
 await page.locator('#rift-receipt > summary').tap();await expect(page.locator('#rift-receipt-list > li')).toHaveCount(1);
 await page.locator('#rift-receipt-list > li > details > summary').tap();await expect(page.locator('#rift-receipt-list > li > details')).toHaveAttribute('open','');
 const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const saved=await read();expect(saved.status).toBe('banked');expect(saved.banked_gold).toBe(30);expect(saved.banked_items).toHaveLength(1);expect(saved.drops.every(drop=>drop.banked)).toBe(true);
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('#rift-banked')).toHaveText('30 gold · 1 item');expect((await read()).banked_items).toEqual(saved.banked_items);
});
