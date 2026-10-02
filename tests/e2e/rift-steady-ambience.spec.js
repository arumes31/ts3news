const {test,expect}=require('@playwright/test');

test('steady ambience persists and resets without changing channel levels',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('.rift-settings > summary').click();const steady=page.locator('#rift-steady-ambience');await expect(steady).not.toBeChecked();await steady.check();await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('.rift-settings > summary').click();await expect(steady).toBeChecked();expect(await page.evaluate(()=>window.RiftAudio.ambience)).toBe(.35);await page.locator('#rift-reset-audio').click();await expect(steady).not.toBeChecked();expect((await(await page.request.get('/api/abyss/rift')).json()).run).toBeNull();
});

test('steady ambience retains background sound while suppressing periodic accents',async({page})=>{
 await page.addInitScript(()=>{window.audioEdges=[];const original=AudioNode.prototype.connect;AudioNode.prototype.connect=function(target,...args){window.audioEdges.push([this,target]);return original.call(this,target,...args);};});await page.goto('/abyss/rift');
 const background=await page.evaluate(async()=>{const a=window.RiftAudio;a.set('steadyAmbience',true);await a.setActive(true,0);return window.audioEdges.some(([source])=>source instanceof AudioBufferSourceNode&&source.loop);});expect(background).toBe(true);await page.waitForTimeout(2300);
 expect(await page.evaluate(()=>{window.RiftAudio.tick();return window.audioEdges.filter(([source])=>source instanceof StereoPannerNode).length;})).toBe(0);
 await page.evaluate(()=>{window.RiftAudio.set('steadyAmbience',false);window.RiftAudio.tick();});await expect.poll(()=>page.evaluate(()=>window.audioEdges.filter(([source])=>source instanceof StereoPannerNode).length)).toBeGreaterThan(0);await page.evaluate(()=>window.RiftAudio.setActive(false));
});
