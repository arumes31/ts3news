const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test('boss jump example teaches timing without changing practice, reduced='+reduced,async({page},testInfo)=>{
 await page.emulateMedia({reducedMotion:reduced?'reduce':'no-preference'});
 const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto('/abyss/rift?practice=boss');await expect(page.locator('#rift-start')).toBeEnabled();
 const posts=[];page.on('request',r=>{if(r.method()==='POST'&&r.url().includes('/api/abyss/rift'))posts.push(r.postDataJSON());});
 const example=page.locator('#rift-boss-jump-example'),canvas=page.locator('#rift-boss-jump-canvas'),caption=page.locator('#rift-boss-jump-status');
 await example.locator('summary').click();
 await page.evaluate(()=>{window.jumpStages=new Set();new MutationObserver(()=>jumpStages.add(document.querySelector('#rift-boss-jump-canvas').dataset.stage)).observe(document.querySelector('#rift-boss-jump-canvas'),{attributes:true,attributeFilter:['data-stage']});});
 await page.locator('#rift-boss-jump-play').click();
 if(reduced){await expect(caption).toContainText('Reduced motion');await expect(canvas).toHaveAttribute('data-playing','false');}
 else {await expect(canvas).toHaveAttribute('data-playing','true');await expect(canvas).toHaveAttribute('data-playing','false',{timeout:5000});expect(await page.evaluate(()=>Array.from(jumpStages))).toEqual(['warning','jump','impact','landing']);}
 // Restart the static sequence from the warning in either motion mode.
 if(reduced)for(let i=0;i<3;i++)await page.locator('#rift-boss-jump-step').click();
 await page.locator('#rift-boss-jump-step').click();await expect(canvas).toHaveAttribute('data-stage','warning');
 await page.locator('#rift-boss-jump-step').click();await expect(canvas).toHaveAttribute('data-stage','jump');
 const key=await page.evaluate(()=>RiftControls.label('jump'));await expect(caption).toHaveText('Press '+key+' now.');
 await page.locator('#rift-boss-jump-step').click();await expect(canvas).toHaveAttribute('data-stage','impact');
 expect(Number(await canvas.getAttribute('data-lift'))).toBeGreaterThan(40);
 await canvas.screenshot({path:testInfo.outputPath('jump-at-impact.png')});
 await page.locator('#rift-boss-jump-step').click();await expect(canvas).toHaveAttribute('data-stage','landing');await expect(canvas).toHaveAttribute('data-lift','0');
 expect(posts).toEqual([]);
 await page.locator('#rift-boss-jump-play').click();await example.locator('summary').click();await expect(canvas).toHaveAttribute('data-playing','false');
 await page.setViewportSize({width:390,height:844});await example.locator('summary').click();expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await page.locator('#rift-boss-jump-sound').click();await expect(caption).toContainText('slam');await example.locator('summary').click();await expect.poll(()=>page.evaluate(()=>RiftAudio.voices)).toBe(0);
 expect(errors).toEqual([]);
});

test('playing the jump example pauses an active boss drill',async({page})=>{
 await page.goto('/abyss/rift?practice=boss');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.locator('#rift-boss-jump-example summary').click();await page.locator('#rift-boss-jump-play').click();await expect(page.locator('#rift-paused-badge')).toHaveText('Paused');
 const r=(await(await page.request.get('/api/abyss/rift?practice=boss')).json()).run;expect(r.paused).toBe(true);
});
