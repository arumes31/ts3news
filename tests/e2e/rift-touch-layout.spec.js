const {test,expect}=require('@playwright/test');
test.use({hasTouch:true,isMobile:true,viewport:{width:390,height:844}});

test('movement-pad alignment persists and resets independently',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('.rift-settings > summary').tap();
 const setting=page.locator('#rift-touch-layout');await expect(setting).toBeVisible();
 await page.locator('#rift-render-rate').selectOption('30');
 for(const [value,align] of [['left','flex-start'],['right','flex-end']]){
  await setting.selectOption(value);await expect(page.locator('.rift-touch')).toHaveCSS('justify-content',align);
  expect(await page.evaluate(()=>JSON.parse(localStorage.getItem('riftTouchLayout')))).toEqual({version:1,alignment:value});
 }
 await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(page.locator('.rift-touch')).toHaveCSS('justify-content','flex-end');
 await page.locator('.rift-settings > summary').tap();await expect(setting).toHaveValue('right');
 await page.locator('#rift-reset-touch-layout').tap();await expect(setting).toHaveValue('center');await expect(page.locator('.rift-touch')).toHaveCSS('justify-content','center');
 await expect(page.locator('#rift-render-rate')).toHaveValue('30');
 expect(await page.evaluate(()=>JSON.parse(localStorage.getItem('riftTouchLayout')))).toEqual({version:1,alignment:'center'});
});

test('invalid saved touch layout safely uses the centered default',async({page})=>{
 await page.addInitScript(()=>localStorage.setItem('riftTouchLayout',JSON.stringify({version:1,alignment:'outside'})));
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 await expect(page.locator('#rift-touch-layout')).toHaveValue('center');await expect(page.locator('.rift-touch')).toHaveCSS('justify-content','center');
});


test('changing pad alignment releases controls that were held',async({page})=>{
 await page.goto('/abyss/rift?practice=skills');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').tap();await expect(page.locator('#rift-overlay')).toBeHidden();
 const left=page.locator('[data-move=left]');await left.hover();await page.mouse.down();await expect(left).toHaveAttribute('aria-pressed','true');
 await page.locator('#rift-touch-layout').evaluate(node=>{node.value='right';node.dispatchEvent(new Event('change'));});
 await expect(left).toHaveAttribute('aria-pressed','false');await expect(page.locator('.rift-touch')).toHaveCSS('justify-content','flex-end');await page.mouse.up();
 await left.hover();await page.mouse.down();await expect(left).toHaveAttribute('aria-pressed','true');await page.mouse.up();
});
