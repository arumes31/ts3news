const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])for(const boss of [false,true])test('knockback holds camera and recovers without a direct-camera snap, boss='+boss+', reduced='+reduced,async({page})=>{
 await page.emulateMedia({reducedMotion:reduced?'reduce':'no-preference'});
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const {run}=await(await page.request.get('/api/abyss/rift')).json();
 const feed=(x,hit,paused=false)=>page.evaluate(({run,x,hit,paused,boss})=>{
  window.RiftDisplay.cameraSmooth=false;run.paused=paused;run.player.x=x;run.player.pose=hit?'hit':'idle';run.player.pose_time=hit?.2:0;run.player.recoil_x=hit?-10:0;run.player.knockdown=0;
  run.enemies=boss?[{id:'camera-boss',kind:'boss',x:900,y:410,hp:100,max_hp:100,facing:1,pose:'idle'}]:[];run.events=[];
  window.RiftRenderer.snapshot(run,true);
 },{run,x,hit,paused,boss});
 const camera=()=>page.evaluate(()=>window.RiftRenderer.cameraFraming.x);
 await feed(850,false,true);await expect.poll(camera).toBe(500);
 await feed(770,true);await page.waitForTimeout(250);expect(await camera()).toBe(500);
 // Sample every rendered frame during recovery, including when smoothing is disabled.
 await page.evaluate(()=>{window.cameraSamples=[RiftRenderer.cameraFraming.x];let left=100;const sample=()=>{cameraSamples.push(RiftRenderer.cameraFraming.x);if(--left>0)requestAnimationFrame(sample);};requestAnimationFrame(sample);});
 await feed(770,false);await expect.poll(camera).toBeLessThan(470);
 await expect.poll(camera).toBeLessThan(boss?421:469);
 const samples=await page.evaluate(()=>cameraSamples);
 expect(samples.length).toBeGreaterThan(2);
 expect(Math.max(...samples.slice(1).map((x,i)=>Math.abs(x-samples[i])))).toBeLessThan(33);
 await expect.poll(camera).toBe(boss?420:468);
 if(boss){const x=await camera();expect(900-x-84).toBeGreaterThanOrEqual(0);expect(900-x+84).toBeLessThanOrEqual(960);}
});
