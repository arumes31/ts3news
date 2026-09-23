const {test,expect}=require('@playwright/test');
const fs=require('node:fs');
for(const reduced of [false,true])test('spike beds extend and retract with saved hazard phase, reduced='+reduced,async({page})=>{
 await page.emulateMedia({reducedMotion:reduced?'reduce':'no-preference'});await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();const {run}=await(await page.request.get('/api/abyss/rift')).json();
 await page.evaluate(()=>{window.spikeHeights=[];const move=CanvasRenderingContext2D.prototype.moveTo,line=CanvasRenderingContext2D.prototype.lineTo;CanvasRenderingContext2D.prototype.moveTo=function(x,y){if(this.fillStyle==='#cbd3dd')this.spikeBase=y;return move.call(this,x,y);};CanvasRenderingContext2D.prototype.lineTo=function(x,y){if(this.fillStyle==='#cbd3dd'&&this.spikeBase>y){if(spikeHeights[0]?.frame!==RiftRenderer.frameCount)spikeHeights=[];spikeHeights.push({height:this.spikeBase-y,frame:RiftRenderer.frameCount});}return line.call(this,x,y);};});
 const show=clock=>page.evaluate(({run,clock})=>{run.clock=clock;run.paused=true;run.status='fighting';run.player.x=160;run.enemies=[];run.events=[];run.level.rooms[run.room].hazards=[{kind:'spikes',x:420,y:370,w:180,h:60,period:7,offset:0,duration:1,jumpable:true}];spikeHeights=[];RiftRenderer.snapshot(run,true);},{run,clock});
 await show(1.4);await expect.poll(()=>page.evaluate(()=>spikeHeights[0]?.height)).toBe(16);
 const png=await page.locator('#rift-canvas').evaluate(c=>c.toDataURL('image/png'));fs.writeFileSync('test-results/spike-beds'+(reduced?'-reduced':'')+'.png',Buffer.from(png.split(',')[1],'base64'));
 for(const clock of [1.11,2.29]){await show(clock);if(reduced){await page.waitForTimeout(100);expect(await page.evaluate(()=>spikeHeights.length)).toBe(0);}else{await expect.poll(()=>page.evaluate(()=>spikeHeights[0]?.height)).toBeCloseTo(8,4);await page.waitForTimeout(100);expect(await page.evaluate(()=>spikeHeights[0].height)).toBeCloseTo(8,4);}}
 await show(2.5);await page.waitForTimeout(100);expect(await page.evaluate(()=>spikeHeights.length)).toBe(0);
});

test('spike impact emits metal tone and noise and stops on pause',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();const result=await page.evaluate(async()=>{const a=RiftAudio;await a.setActive(true,0);const before=a.voices;a.play('spikes',0);const added=a.voices-before;await a.setActive(false);return {added,remaining:a.voices};});expect(result).toEqual({added:2,remaining:0});
});
