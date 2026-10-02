const {test,expect}=require('@playwright/test');
const fs=require('node:fs');
for(const reduced of [false,true])for(const material of ['wood','stone','low','tall'])test('foreground cover fades only over player: '+material+', reduced='+reduced,async({page})=>{
 await page.emulateMedia({reducedMotion:reduced?'reduce':'no-preference'});
 await page.addInitScript(()=>{
  const sources=new WeakMap(),draw=CanvasRenderingContext2D.prototype.drawImage;
  CanvasRenderingContext2D.prototype.drawImage=function(image,...args){
   const source=sources.get(image)||(/rift_terrain_cover/.test(image.src||'')?'terrain':/rift_prop(?:s|_[0-9])/.test(image.src||'')?'prop':null);
   if(source){
    if(this.canvas.id==='rift-canvas'){
     if(window.coverOpacity){coverOpacity.push(this.globalAlpha);if(coverOpacity.length>20)coverOpacity.shift();}
     if(window.coverDraws&&args.length===8)coverDraws.add(JSON.stringify([source,args[4],args[7]]));
    }else sources.set(this.canvas,source);
   }
   return draw.call(this,image,...args);
  };
 });
 await page.goto('/abyss/rift?scenario=terrain-cover');await expect(page.locator('#rift-start')).toBeEnabled();
 const {run}=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(()=>{window.coverOpacity=[];});
 const show=(x,y,elevation=0)=>page.evaluate(async ({run,material,x,y,elevation})=>{run.paused=true;run.player.x=x;run.player.y=y;run.player.elevation=elevation;run.player.jump=elevation>0?.325:0;run.enemies=[];run.events=[];run.drops=[];const o={x:300,y:380,w:80,h:40};run.level.rooms[run.room]={obstacles:material==='low'?[o]:[],high_cover:material==='tall'?[o]:[],cover:['wood','stone'].includes(material)?[{...o,id:'fade-cover',material,hp:material==='wood'?60:0,max_hp:material==='wood'?60:0}]:[]};window.coverOpacity=[];await RiftRenderer.prepareRun(run);RiftRenderer.snapshot(run,true);},{run,material,x,y,elevation});
 await show(340,370);await expect.poll(()=>page.evaluate(()=>coverOpacity.at(-1))).toBeLessThan(.7);
 expect(await page.evaluate(()=>coverOpacity.at(-1))).toBeGreaterThanOrEqual(.3);
 if(!reduced&&['wood','tall'].includes(material)){const data=await page.locator('#rift-canvas').evaluate(c=>c.toDataURL('image/png'));fs.writeFileSync('test-results/cover-fade-'+material+'.png',Buffer.from(data.split(',')[1],'base64'));}
 await show(410,370);await expect.poll(()=>page.evaluate(()=>coverOpacity.at(-1))).toBeGreaterThan(.7);expect(await page.evaluate(()=>coverOpacity.at(-1))).toBeLessThan(1);
 await show(340,370,32);await expect.poll(()=>page.evaluate(()=>coverOpacity.at(-1))).toBe(1);
 await show(600,370);await expect.poll(()=>page.evaluate(()=>coverOpacity.at(-1))).toBe(1);
 await show(340,460);await expect.poll(()=>page.evaluate(()=>coverOpacity.at(-1))).toBe(1);
 await show(340,320);if(material==='low')await expect.poll(()=>page.evaluate(()=>coverOpacity.at(-1))).toBe(1);
});
