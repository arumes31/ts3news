const {test,expect}=require('@playwright/test');
for(const [scenario,pattern] of [['circle','HOLD '],['rescue','BREAK CAGE'],['terrain-cover','WOOD ']])test('interaction prompts remain above sprites: '+scenario,async({page})=>{
 await page.goto('/abyss/rift?scenario='+scenario);await expect(page.locator('#rift-start')).toBeEnabled();
 const {run}=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(({run,scenario})=>{
  run.paused=true;run.events=[];if(scenario==='terrain-cover'){run.player.x=260;run.player.y=400;run.player.facing=1;}
  window.promptDraws=[];
  for(const method of ['drawImage','fillText']){const original=CanvasRenderingContext2D.prototype[method];CanvasRenderingContext2D.prototype[method]=function(...args){if(this.canvas.id==='rift-canvas'){const frame=RiftRenderer.frameCount;if(promptDraws[0]?.frame!==frame)promptDraws=[];promptDraws.push({frame,method,text:method==='fillText'?String(args[0]):''});}return original.apply(this,args);};}
  RiftRenderer.snapshot(run,true);
 },{run,scenario});
 await expect.poll(()=>page.evaluate(pattern=>{const a=promptDraws,labels=a.map((v,i)=>v.text.includes(pattern)?i:-1).filter(i=>i>=0);return labels.length>0&&Math.min(...labels)>a.findLastIndex(v=>v.method==='drawImage');},pattern)).toBe(true);
 await page.evaluate(()=>RiftDisplay.cleanScreenshot=true);await page.waitForTimeout(150);
 expect(await page.evaluate(pattern=>promptDraws.some(v=>v.text.includes(pattern)),pattern)).toBe(false);
});
