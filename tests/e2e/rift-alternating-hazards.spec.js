const {test,expect}=require('@playwright/test');
for(const reduced of [false,true])test('Crossroads hazard groups activate in spatial order, reduced='+reduced,async({page})=>{
 await page.emulateMedia({reducedMotion:reduced?'reduce':'no-preference'});await page.goto('/abyss/rift?scenario=checkpoint&mission=4');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();
 const level=data.levels.find(l=>l.id===4);expect(level.tactic).toContain('linked hazards pulse one at a time from left to right');
 await page.evaluate(()=>{window.linkedLabels=[];const fill=CanvasRenderingContext2D.prototype.fillText;CanvasRenderingContext2D.prototype.fillText=function(text,x,y,...args){if(this.canvas.id==='rift-canvas'){if(linkedLabels[0]?.frame!==RiftRenderer.frameCount)linkedLabels=[];if(/^(JUMP|SAFE|[A-Z]+ IN)/.test(String(text)))linkedLabels.push({text,x,frame:RiftRenderer.frameCount});}return fill.call(this,text,x,y,...args);};});
 const hazards=level.rooms[2].hazards;
 for(let index=0;index<3;index++){
  const clock=1.3+index*hazards[0].period/3;
  await page.evaluate(({run,level,clock})=>{run.level=level;run.room=2;run.player.x=160;run.clock=clock;run.paused=true;run.status='fighting';run.events=[];run.enemies=[];linkedLabels=[];RiftRenderer.snapshot(run,true);},{run:data.run,level,clock});
  await expect.poll(()=>page.evaluate(()=>linkedLabels.filter(l=>l.text.startsWith('JUMP')).map(l=>l.x))).toEqual([hazards[index].x+hazards[index].w/2]);
 }
});
