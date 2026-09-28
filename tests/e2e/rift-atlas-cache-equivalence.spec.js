const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path'),crypto=require('node:crypto');
test('cached atlas frames preserve pixels state and memory bounds',async({page},info)=>{
 test.setTimeout(120000);
 await page.addInitScript(()=>{const request=requestAnimationFrame;window.requestAnimationFrame=cb=>cb.name==='render'?1:request(cb);});
 const candidate=fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift_renderer.js'),'utf8').replace(/\r\n/g,'\n');
 const cached='    const frame=cachedAtlasFrame(img,sx,sy,sw,sh,dw,dh);\n    if(frame)ctx.drawImage(frame.canvas,sx-frame.left,sy-frame.top,sw,sh,dx,dy,dw,dh);\n    else ctx.drawImage(img,sx,sy,sw,sh,dx,dy,dw,dh);';
 expect(candidate.includes(cached)).toBe(true);
 const baseline=candidate.replace(cached,'    ctx.drawImage(img,sx,sy,sw,sh,dx,dy,dw,dh);');
 const hook=`renderer.stateProbe=function(units){
  animationTime=120;decorationTime=120;renderer.reduced=false;motion=1;
  const outputs=[];
  for(const filter of ['none','brightness(1.3)'])for(const flipped of [false,true]){
   ctx.setTransform(1,0,0,1,0,0);ctx.imageSmoothingEnabled=false;ctx.globalAlpha=1;ctx.filter='none';ctx.clearRect(0,0,960,540);ctx.fillStyle='#253d43';ctx.fillRect(0,0,960,540);
   ctx.save();ctx.globalAlpha=.71;ctx.filter=filter;ctx.translate(9.5,12.5);ctx.rotate(.07);ctx.beginPath();ctx.rect(0,0,850,490);ctx.clip();
   const state=()=>({alpha:ctx.globalAlpha,filter:ctx.filter,transform:Array.from(ctx.getTransform().toFloat64Array())});const before=state();
   sprite(0,3,120,220,101,flipped?-1:1,.85,'heroesA');fx(1,2,280,200,120,.7);
   for(let i=0;i<Math.min(3,units.length);i++)catalogActor({...units[i],facing:flipped?-1:1},i===0?'stagger':'knockdown',430+i*145,260,140,.8);
   const after=state();ctx.restore();outputs.push({before,after,png:canvas.toDataURL()});
  }
  for(const sheet of ['heroesA','effects']){
   ctx.setTransform(1,0,0,1,0,0);ctx.globalAlpha=1;ctx.filter='none';ctx.clearRect(0,0,960,540);ctx.fillStyle='#253d43';ctx.fillRect(0,0,960,540);
   const before={alpha:ctx.globalAlpha,filter:ctx.filter};
   for(let row=0;row<6;row++)for(let col=0;col<(sheet==='effects'?6:16);col++){
    if(sheet==='effects')fx(row,col,90+col*140,65+row*85,75,.73);
    else sprite(row,col,35+col*59,80+row*85,75,col%2?-1:1,.73,'heroesA');
   }
   outputs.push({before,after:{alpha:ctx.globalAlpha,filter:ctx.filter},png:canvas.toDataURL()});
  }
  // Exercise eviction with valid distinct rectangles, then draw again after reuse.
  const img=images.heroesA,retired=atlasFrames.values().next().value;
  for(let i=0;i<200;i++)drawAtlas(img,i,0,128,128,0,0,64,64);
  for(let i=0;i<40;i++)drawAtlas(img,i,0,512,512,0,0,64,64);
  return {outputs,cache:renderer.atlasCacheStats(),retired:retired?[retired.canvas.width,retired.canvas.height]:null};
 };`;
 const results=[];
 for(const source of [baseline,candidate]){
  await page.route('**/static/rift_renderer.js*',r=>r.fulfill({contentType:'application/javascript',body:source.replace('window.RiftRenderer=renderer;',hook+'window.RiftRenderer=renderer;')}));
  await page.goto('/abyss/rift?scenario=visual&seed=state-pixels&level=100&room=2&subclass=vanguard');await expect(page.locator('#rift-start')).toBeEnabled();
  const data=await(await page.request.get('/api/abyss/rift')).json();
  const output=await page.evaluate(async run=>{await RiftRenderer.ready;return RiftRenderer.stateProbe(run.enemies);},data.run);
  output.outputs.forEach((item,index)=>fs.writeFileSync(info.outputPath('variant-'+results.length+'-'+index+'.png'),Buffer.from(item.png.split(',')[1],'base64')));
  results.push({outputs:output.outputs.map(item=>({...item,png:crypto.createHash('sha256').update(item.png).digest('hex')})),cache:output.cache,retired:output.retired});
  await page.unroute('**/static/rift_renderer.js*');
 }
 expect(results[1].outputs).toEqual(results[0].outputs);
 expect(results[1].cache.entries).toBeLessThanOrEqual(64);expect(results[1].cache.bytes).toBeLessThanOrEqual(8*1024*1024);expect(results[1].cache.hits).toBeGreaterThan(0);expect(results[1].cache.misses).toBeGreaterThan(64);expect(results[1].retired).toEqual([0,0]);
});
