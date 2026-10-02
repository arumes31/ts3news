const { test, expect } = require('@playwright/test');

test('visual intensity choices persist independently and reject corrupt values',async({page})=>{
  await page.addInitScript(()=>{if(!sessionStorage.getItem('visualSeed')){localStorage.setItem('riftDisplay',JSON.stringify({version:1,flashIntensity:99,particleIntensity:0.5,hazardLabels:false}));sessionStorage.setItem('visualSeed','1');}});
  await page.goto('/abyss/rift');await page.locator('.rift-settings > summary').click();
  await expect(page.locator('#rift-flash-intensity')).toHaveValue('1');await expect(page.locator('#rift-particle-intensity')).toHaveValue('0.5');await expect(page.locator('#rift-hazard-labels')).not.toBeChecked();
  await page.locator('#rift-motion-intensity').selectOption('0');await page.locator('#rift-flash-intensity').selectOption('0');await page.locator('#rift-damage-motion').uncheck();await page.locator('#rift-loot-sparkle').uncheck();
  const saved=await page.evaluate(()=>JSON.parse(localStorage.getItem('riftDisplay')));expect(saved).toMatchObject({motionIntensity:0,flashIntensity:0,damageMotion:false,lootSparkle:false,particleIntensity:0.5});
  await page.reload();await page.locator('.rift-settings > summary').click();await expect(page.locator('#rift-motion-intensity')).toHaveValue('0');await expect(page.locator('#rift-damage-motion')).not.toBeChecked();
  await page.locator('#rift-display-preset').selectOption('cinematic');await expect(page.locator('#rift-preset-description')).toContainText('Full');await expect(page.locator('#rift-motion-intensity')).toHaveValue('0');
  await page.locator('#rift-apply-preset').click();await expect(page.locator('#rift-motion-intensity')).toHaveValue('1');await expect(page.locator('#rift-flash-intensity')).toHaveValue('1');
  await page.locator('#rift-reset-display').click();await expect(page.locator('#rift-hazard-labels')).toBeChecked();
});

test('live system reduced motion is followed until explicitly overridden and can be restored',async({page})=>{
  await page.emulateMedia({reducedMotion:'no-preference'});await page.goto('/abyss/rift');await page.locator('.rift-settings > summary').click();
  await page.emulateMedia({reducedMotion:'reduce'});await expect(page.locator('#rift-reduced')).toBeChecked();
  expect(await page.evaluate(()=>window.RiftRenderer.reduced)).toBe(true);
  await page.locator('#rift-reduced').uncheck();await page.emulateMedia({reducedMotion:'no-preference'});await page.emulateMedia({reducedMotion:'reduce'});await expect(page.locator('#rift-reduced')).not.toBeChecked();
  await page.locator('#rift-system-motion').click();await expect(page.locator('#rift-reduced')).toBeChecked();
  expect(await page.evaluate(()=>localStorage.getItem('riftReducedMotion'))).toBeNull();
  await page.emulateMedia({reducedMotion:'no-preference'});await expect(page.locator('#rift-reduced')).not.toBeChecked();
});

test('hiding hazard labels retains warning outlines and static damage retains its value',async({page})=>{
  await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');await expect(page.locator('#rift-start')).toHaveText('Resume expedition');
  const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
  await page.evaluate(async run=>{
    run.clock=0;run.paused=false;run.enemies=[];run.projectiles=[];run.drops=[];run.level.rooms[0].hazards=[{kind:'fire',x:300,y:300,w:80,h:50,offset:0,period:5,duration:2}];run.events=[{id:run.counter+1,kind:'hurt',value:37,x:400,y:200}];run.counter++;
    window.RiftDisplay.cameraSmooth=false;window.RiftDisplay.damageMotion=false;window.RiftDisplay.hazardLabels=false;
    const ctx=document.getElementById('rift-canvas').getContext('2d');window.paintCalls=[];
    for(const name of ['fillText','strokeRect']){const original=ctx[name].bind(ctx);ctx[name]=(...args)=>{window.paintCalls.push([name,...args]);if(window.paintCalls.length>500)window.paintCalls.shift();return original(...args);};}
    await window.RiftRenderer.prepareRun(run);window.RiftRenderer.snapshot(run,false);
  },run);
  await expect.poll(()=>page.evaluate(()=>window.paintCalls.filter(call=>call[0]==='fillText'&&call[1]==='37').length)).toBeGreaterThan(3);
  const calls=await page.evaluate(()=>window.paintCalls);expect(calls.some(call=>call[0]==='strokeRect')).toBe(true);expect(calls.some(call=>/^FIRE IN /.test(call[1]))).toBe(false);expect(calls.filter(call=>call[1]==='37').every(call=>call[3]===200)).toBe(true);
  await page.evaluate(()=>{window.RiftDisplay.hazardLabels=true;window.paintCalls=[];});await expect.poll(()=>page.evaluate(()=>window.paintCalls.some(call=>/^FIRE IN /.test(call[1])))).toBe(true);
});
