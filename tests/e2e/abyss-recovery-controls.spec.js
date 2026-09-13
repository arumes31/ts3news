const {test, expect} = require('@playwright/test');
const path = require('path');

test('recovery choices stay stationary through their pulse and accept a pointer click', async ({page}) => {
  await page.route('**/static/abyss_ui200.css*', route => route.fulfill({path:path.join(__dirname,'../../internal/bot/webassets/abyss_ui200.css'),contentType:'text/css'}));
  await page.route('**/api/abyss/combat/state', route => route.fulfill({json:{ok:false}}));
  let revives = 0;
  await page.route('**/api/abyss/revive', route => {
    revives++;
    return route.fulfill({json:{ok:false,error:'Recovery fixture rejection'}});
  });
  await page.goto('/abyss?active=1');
  await page.evaluate(() => {downed=true;canRev=true;inRun=true;renderState();setBusy(false);});
  await page.getByRole('button', {name:'I understand', exact:true}).click();
  const button = page.locator('#btnRevive');
  await expect(button).toBeVisible();
  const boxes = await button.evaluate(node => {
    const pulse = node.getAnimations().find(animation => animation.animationName === 'ab-heartbeat');
    if (!pulse) throw new Error('Expected the normal-motion recovery pulse');
    pulse.pause();
    const frames = [0, .12, .24, .36, .7].map(phase => {
      pulse.currentTime = Number(pulse.effect.getTiming().duration) * phase;
      const rect = node.getBoundingClientRect();
      return {x:rect.x,y:rect.y,width:rect.width,height:rect.height};
    });
    pulse.play();
    return frames;
  });
  for (const box of boxes) for (const key of ['x','y','width','height']) expect(box[key]).toBeCloseTo(boxes[0][key], 1);
  await button.click({timeout:5000});
  await expect(page.locator('#abToastHost')).toContainText('Recovery fixture rejection');
  await expect(button).toBeEnabled();
  expect(revives).toBe(1);
  await button.focus();
  await page.keyboard.press('Enter');
  await expect.poll(() => revives).toBe(2);
  await expect(button).toBeEnabled();
  await page.screenshot({path:test.info().outputPath('stationary-recovery.png')});
});
