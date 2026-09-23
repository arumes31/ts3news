const {test,expect}=require('@playwright/test');
const fs=require('node:fs');
for(const reduced of [false,true])test('forge steam stays behind combat rendering, reduced='+reduced,async({page})=>{
 await page.emulateMedia({reducedMotion:reduced?'reduce':'no-preference'});await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();const data=await(await page.request.get('/api/abyss/rift')).json();const level=data.levels.find(l=>l.id===11);expect(level.rooms[0].steam_vents).toHaveLength(2);
 await page.evaluate(()=>{window.steamDraws=[];for(const method of ['ellipse','drawImage','fillText']){const original=CanvasRenderingContext2D.prototype[method];CanvasRenderingContext2D.prototype[method]=function(...args){if(this.canvas.id==='rift-canvas'){if(steamDraws[0]?.frame!==RiftRenderer.frameCount)steamDraws=[];steamDraws.push({frame:RiftRenderer.frameCount,method,steam:this.fillStyle==='#ced8c8'&&method==='ellipse',y:args[1]});}return original.apply(this,args);};}});
 await page.evaluate(({run,level})=>{run.level=level;run.room=0;run.player.x=395;run.paused=true;run.status='fighting';run.enemies=[];run.events=[];RiftRenderer.snapshot(run,true);},{run:data.run,level});
 await expect.poll(()=>page.evaluate(()=>steamDraws.filter(d=>d.steam).length)).toBe(10);
 expect(await page.evaluate(()=>{const last=steamDraws.map(d=>d.steam).lastIndexOf(true);return steamDraws.slice(last+1).some(d=>d.method==='drawImage');})).toBe(true);
 expect(await page.evaluate(()=>steamDraws.filter(d=>d.steam).every(d=>d.y<=300))).toBe(true);
 const positions=await page.evaluate(()=>steamDraws.filter(d=>d.steam).map(d=>d.y));await page.waitForTimeout(150);expect(await page.evaluate(()=>steamDraws.filter(d=>d.steam).map(d=>d.y))).toEqual(positions);
 const png=await page.locator('#rift-canvas').evaluate(c=>c.toDataURL('image/png'));fs.writeFileSync('test-results/steam-vents'+(reduced?'-reduced':'')+'.png',Buffer.from(png.split(',')[1],'base64'));
});
