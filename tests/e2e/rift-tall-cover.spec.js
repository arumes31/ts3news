const {test,expect}=require('@playwright/test');
test('tall cover has a taller silhouette and an explained combat rule',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 await expect(page.locator('#rift-tall-cover-rule')).toContainText('projectiles from either side');
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.evaluate(async run=>{
  await window.RiftRenderer.ready;window.coverHeights=[];
  const ctx=document.getElementById('rift-canvas').getContext('2d'),draw=ctx.drawImage.bind(ctx);
  ctx.drawImage=(...args)=>{if(args.length===9&&args[7]===56)window.coverHeights.push(args[8]);return draw(...args);};
  run.level.rooms[run.room].obstacles=[{x:350,y:380,w:42,h:32}];
  run.level.rooms[run.room].high_cover=[{x:450,y:380,w:42,h:32}];run.paused=true;window.RiftRenderer.snapshot(run,true);
 },run);
 await expect.poll(()=>page.evaluate(()=>window.coverHeights.includes(70)&&window.coverHeights.includes(132))).toBe(true);
});
