const { test, expect } = require('@playwright/test');
const path = require('node:path');

test.beforeEach(async({page})=>{
  // Fast frontend iteration against the unchanged production simulation.
  // Final release verification runs without this override.
  if(process.env.RIFT_WORKSPACE_ASSETS)await page.route(/\/static\/rift\.js(?:\?|$)/,route=>route.fulfill({path:path.resolve(__dirname,'../../internal/bot/webassets/rift.js'),contentType:'application/javascript'}));
});

test('brief action taps survive an in-flight movement request',async({page})=>{
  await page.goto('/abyss/rift?subclass=vanguard');
  await page.locator('#rift-start').click();
  let release, intercept;
  const pending=new Promise(resolve=>intercept=resolve);
  await page.route('**/api/abyss/rift',async route=>{
    const body=route.request().postDataJSON();
    if(body?.kind==='step'&&intercept){const notify=intercept;intercept=null;await new Promise(resolve=>{release=resolve;notify();});}
    await route.continue();
  });
  await pending;
  await page.keyboard.press('Space');
  release();
  await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.skill_timers.jump||0).toBeGreaterThan(0);
  await page.keyboard.press('Escape');
});

test('repeated start clicks create only one expedition while audio resumes',async({page})=>{
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();
  const starts=[];page.on('request',request=>{if(request.method()==='POST'&&['start','resume'].includes(request.postDataJSON()?.kind))starts.push(request);});
  await page.evaluate(()=>{
    const original=window.RiftAudio.setActive;
    let calls=0;
    window.RiftAudio.setActive=async(...args)=>{const delay=++calls===1?50:450;await new Promise(resolve=>setTimeout(resolve,delay));return original(...args);};
    document.querySelector('#rift-start').click();
    document.querySelector('#rift-start').click();
  });
  await expect(page.locator('#rift-overlay')).toBeHidden();
  await page.waitForTimeout(600);
  expect(starts).toHaveLength(1);
  await page.keyboard.press('Escape');
});

test('audio failure does not prevent an expedition from starting',async({page})=>{
  await page.goto('/abyss/rift');
  await page.evaluate(()=>{window.RiftAudio.setActive=async()=>{throw new Error('Audio unavailable');};});
  await page.locator('#rift-start').click();
  await expect(page.locator('#rift-overlay')).toBeHidden();
  await page.keyboard.press('Escape');
  await expect(page.locator('#rift-overlay-title')).toHaveText('A moment by the lantern.');
});

test('Escape pauses when a settings input has focus',async({page})=>{
  await page.goto('/abyss/rift');
  await page.locator('#rift-start').click();
  await page.locator('#rift-campaign > summary').click();
  await page.locator('#rift-auto').focus();
  await page.keyboard.press('Escape');
  await expect(page.locator('#rift-overlay-title')).toHaveText('A moment by the lantern.');
});

test('pause cannot overwrite a terminal result received by a pending step',async({page})=>{
  await page.goto('/abyss/rift');
  await page.locator('#rift-start').click();
  let release, intercepted;
  const pending=new Promise(resolve=>intercepted=resolve);
  await page.route('**/api/abyss/rift',async route=>{
    if(route.request().postDataJSON()?.kind==='step'&&intercepted){
      const response=await route.fetch(),data=await response.json();
      data.run.status='defeated';data.run.player.hp=0;
      const notify=intercepted;intercepted=null;
      await new Promise(resolve=>{release=resolve;notify();});
      await route.fulfill({response,json:data});return;
    }
    await route.continue();
  });
  await pending;await page.keyboard.press('Escape');release();
  await expect(page.locator('#rift-overlay-title')).toHaveText('The rift takes its toll.');
  await page.waitForTimeout(300);
  await expect(page.locator('#rift-overlay-title')).toHaveText('The rift takes its toll.');
});

test('campaign search combines with difficulty, progress and favorites',async({page})=>{
  await page.goto('/abyss/rift');
  await expect(page.locator('[data-level]')).toHaveCount(100);
  await page.locator('#rift-region').selectOption('9');
  await page.locator('#rift-difficulty').selectOption('Mythic');
  await page.locator('#rift-completion').selectOption('unfinished');
  await expect(page.locator('[data-level]:visible')).toHaveCount(10);
  await page.locator('#rift-mission-search').fill('100');
  await expect(page.locator('[data-level]:visible')).toHaveCount(1);
  await page.locator('[data-level="100"]').click();
  await page.locator('#rift-favorite').click();
  await page.locator('#rift-favorites-only').check();
  await page.locator('#rift-compact').check();
  await page.reload();
  await expect(page.locator('[data-level]:visible')).toHaveCount(1);
  await expect(page.locator('#rift-level-description')).toContainText('Mission 100');
  await expect(page.locator('#rift-start')).toHaveText('Enter mission 100');
  await expect(page.locator('#rift-favorite')).toHaveAttribute('aria-pressed','true');
  await expect(page.locator('#rift-levels')).toHaveClass(/compact/);
  await page.locator('#rift-mission-search').fill('not a mission');
  await expect(page.locator('#rift-filter-empty')).toBeVisible();
  await page.locator('#rift-clear-filters').click();
  await expect(page.locator('[data-level]:visible')).toHaveCount(100);
});

test('mission shortcuts do not replace an active expedition',async({page})=>{
  await page.goto('/abyss/rift');
  await page.locator('#rift-next-mission').click();
  await expect(page.locator('#rift-level-description')).toContainText('Mission 2');
  await page.locator('#rift-previous-mission').click();
  await expect(page.locator('#rift-level-description')).toContainText('Mission 1');
  await page.locator('#rift-unfinished').click();
  await expect(page.locator('#rift-level-description')).toContainText('Mission 2');
  await page.locator('#rift-start').click();
  await page.locator('#rift-campaign > summary').click();
  await expect(page.locator('#rift-next-mission')).toBeDisabled();
  await expect(page.locator('#rift-previous-mission')).toBeDisabled();
  await expect(page.locator('#rift-unfinished')).toBeDisabled();
  await page.keyboard.press('Escape');
});

test('completion filters reflect confirmed regional progress',async({page})=>{
  await page.goto('/abyss/rift?scenario=checkpoint&room=final');
  await page.locator('#rift-start').click();
  await expect.poll(async()=>(await(await page.request.get('/api/abyss/rift')).json()).run.level.id).toBe(2);
  await page.keyboard.press('Escape');
  await page.locator('#rift-campaign > summary').click();
  await page.locator('#rift-completion').selectOption('complete');
  await expect(page.locator('[data-level]:visible')).toHaveCount(1);
  await expect(page.locator('#rift-filter-count')).toHaveText('1 of 100 missions');
  await expect(page.locator('#rift-region-progress')).toContainText('Mossbound Ruins: 1/10');
  await page.locator('#rift-show-selected').click();
  await expect(page.locator('[data-level="2"]')).toBeVisible();
});
