const {test,expect}=require('@playwright/test');

test('screen shake persists valid settings and resets to off without touching a run',async({page})=>{
 await page.addInitScript(()=>{if(!sessionStorage.getItem('shakeSeed')){localStorage.setItem('riftDisplay',JSON.stringify({version:1,shakeIntensity:'bad',textScale:1.25}));sessionStorage.setItem('shakeSeed','1');}});
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('.rift-settings > summary').click();await expect(page.locator('#rift-shake-intensity')).toHaveValue('0');await expect(page.locator('#rift-text-scale')).toHaveValue('1.25');
 await page.locator('#rift-shake-intensity').selectOption('0.5');await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('.rift-settings > summary').click();await expect(page.locator('#rift-shake-intensity')).toHaveValue('0.5');await page.locator('#rift-reset-display').click();await expect(page.locator('#rift-shake-intensity')).toHaveValue('0');expect((await(await page.request.get('/api/abyss/rift')).json()).run).toBeNull();
});

test('screen shake moves rendered impacts only when enabled and motion is allowed',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.locator('.rift-settings > summary').click();await page.locator('#rift-shake-intensity').selectOption('1');
 let counter=run.counter;
 const sample=async()=>page.evaluate(async({run,id})=>{const renderer=window.RiftRenderer;renderer.snapshot({...run,paused:false,counter:id,events:[{id,kind:'hurt',value:8,x:run.player.x,y:run.player.y}]},false);const offsets=[];for(let i=0;i<8;i++){await new Promise(requestAnimationFrame);const m=document.getElementById('rift-canvas').getContext('2d').getTransform();offsets.push(Math.abs(m.e)+Math.abs(m.f));}return offsets;},{run,id:++counter});
 expect(Math.max(...await sample())).toBeGreaterThan(0);await page.locator('#rift-shake-intensity').selectOption('0');expect(Math.max(...await sample())).toBe(0);
 await page.locator('#rift-shake-intensity').selectOption('1');await page.locator('#rift-reduced').check();expect(Math.max(...await sample())).toBe(0);
 await page.locator('#rift-reduced').uncheck();await page.locator('#rift-motion-intensity').selectOption('0');expect(Math.max(...await sample())).toBe(0);
 expect((await(await page.request.get('/api/abyss/rift')).json()).run.paused).toBe(true);
});
