const {test,expect}=require('@playwright/test');
test.use({hasTouch:true,viewport:{width:844,height:390}});

for(const viewport of [{width:844,height:390},{width:390,height:844}])test('fullscreen sound and exit at '+viewport.width+'x'+viewport.height,async({page})=>{
 await page.setViewportSize(viewport);
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 await expect(page.locator('#rift-fullscreen-exit')).toBeHidden();
 await page.locator('#rift-fullscreen').tap();await expect.poll(()=>page.evaluate(()=>document.fullscreenElement?.id)).toBe('rift-viewport');
 const exit=page.locator('#rift-fullscreen-exit'),sound=page.locator('#rift-fullscreen-sound');
 for(const button of [exit,sound]){
  await expect(button).toBeVisible();
  const result=await button.evaluate(node=>{const r=node.getBoundingClientRect();return {width:r.width,height:r.height,reachable:node.contains(document.elementFromPoint(r.x+r.width/2,r.y+r.height/2))};});
  expect(result.width).toBeGreaterThanOrEqual(44);expect(result.height).toBeGreaterThanOrEqual(44);expect(result.reachable).toBe(true);
 }
 await sound.tap();await expect(sound).toHaveText('Sound off');await expect(page.locator('#rift-sound')).toHaveText('Sound off');
 await sound.tap();await expect(sound).toHaveText('Sound on');
 await page.screenshot({path:test.info().outputPath('fullscreen-tools.png')});
 await page.evaluate(()=>window.RiftDisplay.toggleScreenshot());await expect(exit).toBeVisible();
 await exit.tap();await expect.poll(()=>page.evaluate(()=>document.fullscreenElement)).toBeNull();await expect(exit).toBeHidden();
 await expect(page.locator('#rift-canvas')).toBeFocused();
});
