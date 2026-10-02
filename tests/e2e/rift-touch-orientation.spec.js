const {test,expect}=require('@playwright/test');
test.use({hasTouch:true,isMobile:true,viewport:{width:390,height:844}});
for(const signal of ['screen','legacy','viewport'])test('held attack resets on '+signal+' orientation change',async({page})=>{
 let input=null,attacks=0;await page.route('**/api/abyss/rift*',async route=>{const body=route.request().postDataJSON();if(body?.kind==='step'){input=body.input;if(input.attack)attacks++;}await route.continue();});
 await page.goto('/abyss/rift?practice=skills');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').tap();await expect(page.locator('#rift-overlay')).toBeHidden();
 const attack=page.locator('[data-bind=attack]');await attack.scrollIntoViewIfNeeded();const box=await attack.boundingBox();
 const cdp=await page.context().newCDPSession(page);
 await cdp.send('Input.dispatchTouchEvent',{type:'touchStart',touchPoints:[{id:1,x:box.x+box.width/2,y:box.y+box.height/2}]});
 await expect.poll(()=>input?.attack).toBe(true);
 if(signal==='viewport')await page.setViewportSize({width:844,height:390});
 else await page.evaluate(signal=>{(signal==='screen'?screen.orientation:window).dispatchEvent(new Event(signal==='screen'?'change':'orientationchange'));},signal);
 await expect(attack).toHaveAttribute('aria-pressed','false');await expect.poll(()=>input?.attack).toBe(false);
 await expect(page.locator('#rift-overlay')).toBeHidden();
 await cdp.send('Input.dispatchTouchEvent',{type:'touchEnd',touchPoints:[]});
 const before=attacks;await attack.tap();await expect.poll(()=>attacks).toBeGreaterThan(before);
});

test('same-orientation toolbar-sized resize preserves held input',async({page})=>{
 await page.goto('/abyss/rift?practice=skills');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').tap();
 const attack=page.locator('[data-bind=attack]');await attack.hover();await page.mouse.down();await expect(attack).toHaveAttribute('aria-pressed','true');
 await page.setViewportSize({width:390,height:760});await expect(attack).toHaveAttribute('aria-pressed','true');await page.mouse.up();
});
