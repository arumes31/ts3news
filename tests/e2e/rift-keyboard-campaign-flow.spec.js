const {test,expect}=require('@playwright/test');
test('campaign selection and start work using only keyboard input',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 async function tabTo(selector){
  for(let i=0;i<220;i++){
   if(await page.locator(selector).evaluate(el=>el===document.activeElement))return;
   await page.keyboard.press('Tab');
  }
  throw new Error('Keyboard could not reach '+selector);
 }
 await tabTo('#rift-campaign > summary');await page.keyboard.press('Enter');await expect(page.locator('#rift-campaign')).not.toHaveAttribute('open','');
 await page.keyboard.press('Enter');await expect(page.locator('#rift-campaign')).toHaveAttribute('open','');
 await tabTo('#rift-region');await page.keyboard.press('Home');await page.keyboard.press('ArrowDown');await page.keyboard.press('ArrowDown');await page.keyboard.press('ArrowDown');await page.keyboard.press('Enter');await expect(page.locator('#rift-region')).toHaveValue('2');
 await tabTo('#rift-levels [data-level="21"]');await page.keyboard.press('ArrowRight');await expect(page.locator('[data-level="22"]')).toBeFocused();await page.keyboard.press('Enter');
 await expect(page.locator('#rift-start')).toHaveText('Enter mission 22');
 await tabTo('#rift-start');await page.keyboard.press('Enter');await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;expect(run.level.id).toBe(22);expect(run.room).toBe(0);expect(run.status).toBe('fighting');
});
