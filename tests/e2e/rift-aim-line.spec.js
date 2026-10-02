const {test,expect}=require('@playwright/test');
for(const mode of ['archer','boss-volley','boss-slam','idle','dead','clean','reduced'])test('ranged aim line: '+mode,async({page},testInfo)=>{
 await page.emulateMedia({reducedMotion:mode==='reduced'?'reduce':'no-preference'});
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.evaluate(async({run,mode})=>{
  await window.RiftRenderer.ready;document.getElementById('rift-overlay').hidden=true;
  window.aimSegments=[];window.slamEllipses=0;
  const ctx=document.getElementById('rift-canvas').getContext('2d'),move=ctx.moveTo.bind(ctx),line=ctx.lineTo.bind(ctx),ellipse=ctx.ellipse.bind(ctx);
  let start;ctx.moveTo=(x,y)=>{if(ctx.strokeStyle==='#ffbd81')start={x,y};return move(x,y);};
  ctx.lineTo=(x,y)=>{if(ctx.strokeStyle==='#ffbd81'&&start)window.aimSegments.push({dx:x-start.x,dy:y-start.y,dash:ctx.getLineDash()});return line(x,y);};
  ctx.ellipse=(...args)=>{if(ctx.strokeStyle==='#ffce7d'&&args[2]===125)window.slamEllipses++;return ellipse(...args);};
  window.RiftDisplay.cleanScreenshot=mode==='clean';run.paused=true;run.status='fighting';run.events=[];
  run.enemies=[{id:'aiming',kind:mode.startsWith('boss')?'boss':'archer',art_key:mode.startsWith('boss')?'monster:Test':'',attacks:mode==='boss-volley'?1:0,x:run.player.x+200,y:run.player.y+25,hp:mode==='dead'?0:100,max_hp:100,windup:mode==='idle'?0:.4,pose:'windup',facing:-1,target_x:run.player.x,target_y:run.player.y}];
  window.RiftRenderer.snapshot(run,false);
 },{run,mode});
 if(['archer','boss-volley','reduced'].includes(mode)){
  await expect.poll(()=>page.evaluate(()=>window.aimSegments.length)).toBeGreaterThan(0);
  expect(await page.evaluate(()=>window.aimSegments.every(s=>s.dx===-200&&s.dy===-25&&s.dash.length===2))).toBe(true);
  expect(await page.evaluate(()=>window.slamEllipses)).toBe(0);
  if(mode==='archer')await page.locator('#rift-canvas').screenshot({path:testInfo.outputPath('aim-line.png')});
 }else{
  await page.waitForTimeout(200);expect(await page.evaluate(()=>window.aimSegments.length)).toBe(0);
  if(mode==='boss-slam')expect(await page.evaluate(()=>window.slamEllipses)).toBeGreaterThan(0);
 }
});
