const {test,expect}=require('@playwright/test');
test.use({hasTouch:true,viewport:{width:1000,height:1200}});
for(const source of ['keyboard','touch'])test('reconnect clears held '+source+' inputs and accepts fresh presses',async({page})=>{
 let fail=false;const inputs=[];
 await page.route('**/api/abyss/rift*',async route=>{
  const body=route.request().postDataJSON();
  if(body?.kind==='step'){inputs.push(body.input);if(fail){await route.abort();return;}}
  await route.continue();
 });
 await page.goto('/abyss/rift?practice=skills');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').tap();await expect(page.locator('#rift-overlay')).toBeHidden();
 let cdp;
 if(source==='keyboard'){await page.locator('#rift-canvas').focus();await page.keyboard.down('KeyA');await page.keyboard.down('KeyJ');}
 else{
  const attack=page.locator('[data-bind=attack]');await attack.scrollIntoViewIfNeeded();const box=await attack.boundingBox();
  cdp=await page.context().newCDPSession(page);await cdp.send('Input.dispatchTouchEvent',{type:'touchStart',touchPoints:[{id:1,x:box.x+box.width/2,y:box.y+box.height/2}]});
 }
 await expect.poll(()=>inputs.some(input=>input.attack&&(source==='touch'||input.x===-1))).toBe(true);
 fail=true;await expect(page.locator('#rift-start')).toHaveText('Recover expedition');
 if(source==='touch')await expect(page.locator('[data-bind=attack]')).toHaveAttribute('aria-pressed','false');
 // Use a separate mouse pointer to recover while the original finger remains down.
 fail=false;await page.locator('#rift-start').click();await expect(page.locator('#rift-start')).toHaveText('Resume drill');
 inputs.length=0;await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await expect.poll(()=>inputs.length).toBeGreaterThanOrEqual(3);
 expect(inputs.every(input=>!input.attack&&!input.guard&&!input.jump&&input.x===0&&input.y===0&&!input.skill)).toBe(true);
 if(source==='keyboard'){await page.keyboard.up('KeyA');await page.keyboard.up('KeyJ');await page.locator('#rift-canvas').focus();await page.keyboard.press('KeyJ');}
 else {await cdp.send('Input.dispatchTouchEvent',{type:'touchEnd',touchPoints:[]});await page.locator('[data-bind=attack]').tap();}
 await expect.poll(()=>inputs.some(input=>input.attack)).toBe(true);
 await page.locator('#rift-pause').tap();
});
