const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])for(const smooth of [true,false])test('camera holds a tracking band at both arena boundaries, smooth='+smooth+', reduced='+reduced,async({page})=>{
 await page.emulateMedia({reducedMotion:reduced?'reduce':'no-preference'});
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const {run}=await(await page.request.get('/api/abyss/rift')).json();
 const feed=async(x,paused=false)=>page.evaluate(({run,x,paused,smooth})=>{window.RiftDisplay.cameraSmooth=smooth;run.player.x=x;run.paused=paused;run.enemies=[];run.events=[];window.RiftRenderer.snapshot(run,true);return window.RiftRenderer.cameraFraming?.x;},{run,x,paused,smooth});
 const camera=()=>page.evaluate(()=>window.RiftRenderer.cameraFraming.x);
 await feed(160,true);await expect.poll(camera).toBe(0);
 await feed(390);await page.waitForTimeout(300);expect(await camera()).toBe(0);
 await feed(450);await expect.poll(camera).toBeGreaterThan(50);expect(await camera()).toBeLessThanOrEqual(52);
 const held=await feed(425);await page.waitForTimeout(300);expect(Math.abs(await camera()-held)).toBeLessThan(.1);
 await feed(330);await expect.poll(camera).toBeLessThan(29);
 await feed(1450,true);await expect.poll(camera).toBe(640);
 await feed(950);await page.waitForTimeout(300);expect(await camera()).toBe(640);
 await feed(900);await expect.poll(camera).toBeLessThan(600);expect(await camera()).toBeGreaterThanOrEqual(598);
 const right=await feed(925);await page.waitForTimeout(300);expect(Math.abs(await camera()-right)).toBeLessThan(.1);
 await feed(160,true);await expect.poll(camera).toBe(0);
});
