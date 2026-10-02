const {test,expect}=require('@playwright/test');
const fs=require('node:fs');
for(const reduced of [false,true])test('checkerboard leaves four safe tiles and alternates, reduced='+reduced,async({page})=>{
 await page.emulateMedia({reducedMotion:reduced?'reduce':'no-preference'});await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();
 const level=data.levels.find(l=>l.id===9);expect(level.tactic).toContain('safe group alternates');const hazards=level.rooms[2].hazards;expect(hazards).toHaveLength(8);
 await page.evaluate(()=>{window.tileLabels=[];const fill=CanvasRenderingContext2D.prototype.fillText;CanvasRenderingContext2D.prototype.fillText=function(text,x,y,...args){if(this.canvas.id==='rift-canvas'&&/^(JUMP|SAFE)/.test(String(text))){if(tileLabels[0]?.frame!==RiftRenderer.frameCount)tileLabels=[];tileLabels.push({text,x,y,frame:RiftRenderer.frameCount});}return fill.call(this,text,x,y,...args);};});
 for(const clock of [1.3,4.3]){
  await page.evaluate(({run,level,clock})=>{run.level=level;run.room=2;run.player.x=160;run.clock=clock;run.paused=true;run.status='fighting';run.events=[];run.enemies=[];tileLabels=[];RiftRenderer.snapshot(run,true);},{run:data.run,level,clock});
  const active=h=>{const phase=(clock+h.offset)%h.period;return phase>=1.2&&phase<1.2+h.duration;};
  for(const [prefix,wanted] of [['JUMP',true],['SAFE',false]]){
   const expected=hazards.filter(h=>active(h)===wanted).map(h=>({x:h.x+h.w/2,y:h.y-6}));expect(expected).toHaveLength(4);
   await expect.poll(()=>page.evaluate(prefix=>tileLabels.filter(l=>l.text.startsWith(prefix)).map(({x,y})=>({x,y})),prefix)).toEqual(expected);
  }
 }
 const png=await page.locator('#rift-canvas').evaluate(c=>c.toDataURL('image/png'));fs.writeFileSync('test-results/checkerboard'+(reduced?'-reduced':'')+'.png',Buffer.from(png.split(',')[1],'base64'));
});
