const {test,expect}=require('@playwright/test');
test('canvas stays bounded through mobile resize, large displays and DPR changes',async({page,context})=>{
 const images=[];page.on('request',request=>{if(request.resourceType()==='image'&&!/\/rift_region_[1-9]\.png/.test(request.url()))images.push(request.url());});
 await page.goto('/abyss/rift?scenario=checkpoint');
 await expect(page.locator('#rift-start')).toBeEnabled();
 await page.evaluate(async()=>{await window.RiftRenderer.ready;});
 await page.evaluate(()=>{
  window.canvasSizeWrites=0;
  for(const key of ['width','height']){
   const descriptor=Object.getOwnPropertyDescriptor(HTMLCanvasElement.prototype,key);
   Object.defineProperty(HTMLCanvasElement.prototype,key,{...descriptor,set(value){if(this.id==='rift-canvas')window.canvasSizeWrites++;return descriptor.set.call(this,value);}});
  }
  window.canvasDocumentToken='same-document';
 });
 const loaded=images.length;expect(loaded).toBeGreaterThan(0);
 const session=await context.newCDPSession(page);
 const cases=[{width:390,height:844,dpr:1},{width:390,height:700,dpr:1},{width:390,height:844,dpr:1},{width:390,height:650,dpr:1},{width:3840,height:2160,dpr:1},{width:1280,height:900,dpr:2},{width:1280,height:900,dpr:3},{width:1280,height:900,dpr:1}];
 for(const size of cases){
  const before=await page.evaluate(()=>window.RiftRenderer.frameCount);
  await session.send('Emulation.setDeviceMetricsOverride',{width:size.width,height:size.height,deviceScaleFactor:size.dpr,mobile:false});
  await expect.poll(()=>page.evaluate(()=>window.RiftRenderer.frameCount)).toBeGreaterThan(before);
  const state=await page.locator('#rift-canvas').evaluate(canvas=>{
   const bounds=canvas.getBoundingClientRect();
   return {width:canvas.width,height:canvas.height,left:bounds.left,right:bounds.right,ratio:bounds.width/bounds.height,viewport:innerWidth,dpr:devicePixelRatio,writes:window.canvasSizeWrites,token:window.canvasDocumentToken,smoothing:canvas.getContext('2d').imageSmoothingEnabled};
  });
  expect(state.width).toBe(960);expect(state.height).toBe(540);
  expect(state.viewport).toBe(size.width);expect(state.dpr).toBe(size.dpr);
  expect(state.writes).toBe(0);expect(state.token).toBe('same-document');
  expect(state.left).toBeGreaterThanOrEqual(0);expect(state.right).toBeLessThanOrEqual(size.width);
  expect(state.ratio).toBeCloseTo(16/9,2);expect(state.smoothing).toBe(false);
  expect(images.length).toBe(loaded);
 }
 await session.send('Emulation.clearDeviceMetricsOverride');
});
