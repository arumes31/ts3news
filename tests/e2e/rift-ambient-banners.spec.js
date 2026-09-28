const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test('rear regional banners stay outside combat and freeze, reduced='+reduced,async({page},info)=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(async({run,levels,reduced})=>{
  await RiftRenderer.ready;RiftRenderer.reduced=reduced;RiftDisplay.cameraSmooth=false;RiftDisplay.particles=false;
  run.status='fighting';run.paused=true;run.events=[];run.enemies=[];run.room=0;run.player.x=160;
  window.bannerRun=run;window.bannerLevels=levels;window.bannerBrackets=[];window.bannerVertices=[];
  const ctx=document.querySelector('#rift-canvas').getContext('2d'),fill=ctx.fillRect,line=ctx.lineTo;
  const colors=['#304a39','#633a29','#365561','#45435f','#464e2e','#285154','#57332f','#45374f','#343653','#514237'];
  ctx.fillRect=function(x,y,w,h){if(this.fillStyle==='#091914'&&w===960&&h===540){bannerBrackets=[];bannerVertices=[];}if(this.fillStyle==='#797366'&&w===56&&h===6)bannerBrackets.push({x,y,w,h});return fill.call(this,x,y,w,h);};
  ctx.lineTo=function(x,y){if(colors.includes(this.fillStyle))bannerVertices.push([x,y]);return line.call(this,x,y);};
  const sheet=document.createElement('canvas');sheet.id='banner-sheet';sheet.style.position='relative';sheet.style.zIndex='100000';sheet.style.background='#091914';sheet.width=600;sheet.height=700;document.body.append(sheet);
  document.querySelector('#rift-overlay').hidden=true;RiftRenderer.snapshot(run,true);
 },{...data,reduced});
 for(let region=0;region<10;region++){
  const before=await page.evaluate(region=>{bannerRun.level=structuredClone(bannerLevels[region*10]);RiftRenderer.snapshot(bannerRun,true);return {level:JSON.stringify(bannerRun.level),frame:RiftRenderer.frameCount};},region);
  await expect.poll(()=>page.evaluate(()=>RiftRenderer.frameCount)).toBeGreaterThan(before.frame);
  await expect.poll(()=>page.evaluate(()=>bannerBrackets.length)).toBe(2);
  const geometry=await page.evaluate(()=>({brackets:bannerBrackets,vertices:bannerVertices,level:JSON.stringify(bannerRun.level)}));
  expect(geometry.level).toBe(before.level);expect(geometry.vertices.length).toBeGreaterThan(0);
  expect(geometry.brackets.every(b=>b.y+b.h<315)).toBe(true);expect(geometry.vertices.every(v=>v[1]<300)).toBe(true);
  await page.evaluate(region=>{const source=document.querySelector('#rift-canvas'),target=document.querySelector('#banner-sheet').getContext('2d'),b=bannerBrackets[0];target.drawImage(source,b.x-3,178,62,122,(region%2)*300,Math.floor(region/2)*140,124,122);target.fillStyle='#fff';target.font='12px monospace';target.fillText(bannerRun.level.region_name,(region%2)*300,Math.floor(region/2)*140+135);},region);
 }
 await page.locator('#banner-sheet').screenshot({path:info.outputPath('regional-banners.png')});
 await page.locator('#rift-canvas').screenshot({path:info.outputPath('rear-banners-scene.png')});
 const frozen=await page.evaluate(()=>JSON.stringify(bannerVertices));await page.waitForTimeout(250);expect(await page.evaluate(()=>JSON.stringify(bannerVertices))).toBe(frozen);
 await page.evaluate(()=>{bannerRun.paused=false;RiftRenderer.snapshot(bannerRun,true);});await page.waitForTimeout(150);
 const moving=await page.evaluate(()=>JSON.stringify(bannerVertices));await page.waitForTimeout(250);
 if(reduced)expect(await page.evaluate(()=>JSON.stringify(bannerVertices))).toBe(moving);else expect(await page.evaluate(()=>JSON.stringify(bannerVertices))).not.toBe(moving);
 await page.evaluate(()=>{RiftDisplay.motionIntensity=0;});await page.waitForTimeout(100);
 const still=await page.evaluate(()=>JSON.stringify(bannerVertices));await page.waitForTimeout(200);expect(await page.evaluate(()=>JSON.stringify(bannerVertices))).toBe(still);
 await page.evaluate(()=>{bannerRun.paused=true;bannerRun.player.x=1500;RiftRenderer.snapshot(bannerRun,true);});
 await expect.poll(()=>page.evaluate(()=>bannerBrackets.length)).toBe(2);expect(await page.evaluate(()=>bannerBrackets.every(b=>b.x+56>=0&&b.x<=960))).toBe(true);
 await page.evaluate(()=>{bannerRun.practice={mode:'movement',arena:{}};RiftRenderer.snapshot(bannerRun,true);});await page.waitForTimeout(100);expect(await page.evaluate(()=>bannerBrackets)).toEqual([]);
 expect(errors).toEqual([]);
});
