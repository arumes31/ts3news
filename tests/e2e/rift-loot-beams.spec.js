const {test,expect}=require('@playwright/test');

test('rare loot beams follow canonical rarities, delivery state and saved intensity',async({page},testInfo)=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.locator('.rift-settings > summary').click();
 const intensity=page.locator('#rift-loot-beam-intensity');await expect(intensity).toHaveValue('1');
 await page.locator('#rift-loot-sparkle').uncheck();await page.locator('#rift-loot-motion').uncheck();
 await page.emulateMedia({reducedMotion:'reduce'});
 await page.evaluate(async()=>{
  await window.RiftRenderer.ready;
  const data=await(await fetch('/api/abyss/rift')).json();window.beamRun=data.run;window.beamRarities=data.rarities;
  const run=window.beamRun;run.paused=true;run.drops=[run.drops.find(d=>d.gear)];
  Object.assign(run.drops[0],{x:run.player.x+300,y:410,elevation:0,collected:false,banked:false});
  document.querySelector('#rift-overlay').hidden=true;
  window.beamDraws=[];const ctx=document.querySelector('#rift-canvas').getContext('2d'),original=ctx.fillRect;
  const colors=new WeakMap(),linear=ctx.createLinearGradient;
  ctx.createLinearGradient=function(...args){const gradient=linear.apply(this,args),stop=gradient.addColorStop;gradient.addColorStop=function(offset,color){if(offset===1)colors.set(this,color);return stop.call(this,offset,color);};return gradient;};
  ctx.fillRect=function(x,y,w,h){if((w===24&&h===112)||(w===32&&h===144))window.beamDraws.push({x,y,w,h,alpha:this.globalAlpha,color:colors.get(this.fillStyle)});return original.call(this,x,y,w,h);};
 });
 async function render(change){
  await page.evaluate(change);
  await page.evaluate(()=>{window.beamDraws=[];window.RiftRenderer.snapshot(window.beamRun,true);});
  await page.waitForTimeout(180);
  return page.evaluate(()=>window.beamDraws);
 }
 const rarities=await page.evaluate(()=>window.beamRarities);
 for(const rarity of rarities){
  await page.evaluate(value=>window.beamRun.drops[0].gear.Rarity=value,rarity.value);
  const draws=await render(()=>{});
  expect(draws.length>0,rarity.name).toBe(rarity.rare_or_better===true);
  if(draws.length){expect(draws[0].h).toBe(rarity.legendary?144:112);expect(draws[0].alpha).toBeCloseTo(.6);expect(draws[0].color).toBe(rarity.color+'cc');expect(draws[0].y+draws[0].h).toBe(410);}
 }
 await page.evaluate(()=>window.beamRun.drops[0].gear.Rarity=window.beamRarities.find(r=>r.legendary).value);
 await render(()=>{});await page.locator('#rift-canvas').screenshot({path:testInfo.outputPath('rare-loot-full-beam.png')});
 await intensity.selectOption('0.5');let draws=await render(()=>{});expect(draws.length).toBeGreaterThan(0);expect(draws[0].alpha).toBeCloseTo(.3);
 expect(new Set(draws.map(d=>JSON.stringify(d))).size).toBe(1);
 await page.locator('#rift-canvas').screenshot({path:testInfo.outputPath('rare-loot-soft-beam.png')});
 await intensity.selectOption('0');expect(await render(()=>{})).toHaveLength(0);
 await intensity.selectOption('1');
 expect(await render(()=>window.beamRun.drops[0].gear.Rarity=-123)).toHaveLength(0);
 await page.evaluate(()=>{window.beamGear=window.beamRun.drops[0].gear;window.beamRun.drops[0].gear=null;});
 expect(await render(()=>{})).toHaveLength(0);
 await page.evaluate(()=>{window.beamRun.drops[0].gear=window.beamGear;window.beamGear.Rarity=window.beamRarities.find(r=>r.legendary).value;});
 expect(await render(()=>window.beamRun.drops[0].collected=true)).toHaveLength(0);
 expect(await render(()=>Object.assign(window.beamRun.drops[0],{collected:false,banked:true}))).toHaveLength(0);
 expect(await render(()=>{window.beamRun.drops[0].banked=false;window.beamRun.practice={mode:'pickup'};})).toHaveLength(0);
 expect(await render(()=>window.beamRun.practice.mode='banking')).toHaveLength(0);
 for(const preset of ['minimal','accessible','lowPower']){
  await page.locator('#rift-display-preset').selectOption(preset);await page.locator('#rift-apply-preset').click();await expect(intensity).toHaveValue('0');
 }
 await intensity.selectOption('0.5');await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await expect(intensity).toHaveValue('0.5');
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
