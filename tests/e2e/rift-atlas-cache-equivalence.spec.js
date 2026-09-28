const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path'),crypto=require('node:crypto');
test('cached atlas frames preserve pixels state and memory bounds',async({page},info)=>{
 test.setTimeout(120000);
 await page.addInitScript(()=>{const request=requestAnimationFrame;window.requestAnimationFrame=cb=>cb.name==='render'?1:request(cb);});
 let candidate=fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift_renderer.js'),'utf8').replace(/\r\n/g,'\n');
 const cached='    const frame=cachedAtlasFrame(img,sx,sy,sw,sh,dw,dh);\n    if(frame)ctx.drawImage(frame.canvas,sx-frame.left,sy-frame.top,sw,sh,dx,dy,dw,dh);\n    else ctx.drawImage(img,sx,sy,sw,sh,dx,dy,dw,dh);';
 expect(candidate.includes(cached)).toBe(true);
 const sharedOrigin=process.env.BRAWL_SHARED_ORIGIN_EXPERIMENT==='1';
 const opaque=process.env.BRAWL_OPAQUE_CANVAS_EXPERIMENT==='1';
 if(sharedOrigin&&opaque)throw Error('Choose one rendering experiment');
 const baseline=opaque||sharedOrigin?candidate:candidate.replace(cached,'    ctx.drawImage(img,sx,sy,sw,sh,dx,dy,dw,dh);');
 if(opaque){expect(candidate.split("canvas.getContext('2d')").length-1).toBe(1);candidate=candidate.replace("canvas.getContext('2d')","canvas.getContext('2d',{alpha:false})");}
 if(sharedOrigin)candidate=require('../../scripts/brawl-shared-origin-experiment.cjs').sharedOriginCandidate(candidate);
 const hook=`renderer.stateProbe=function(units){
  animationTime=120;decorationTime=120;renderer.reduced=false;motion=1;
  const outputs=[];
  const prefix=cachedAtlasFrame(catalogImages[bestiary.assets[0]],0,0,156.75,158,80,80);
  const sharedPrefix=prefix?{left:prefix.left,top:prefix.top,bytes:prefix.bytes}:null;
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
  for(const flipped of [false,true]){
   ctx.setTransform(1,0,0,1,0,0);ctx.globalAlpha=1;ctx.filter='none';ctx.clearRect(0,0,960,540);ctx.fillStyle='#253d43';ctx.fillRect(0,0,960,540);
   ctx.save();ctx.globalAlpha=.35;ctx.filter='brightness(1.3)';ctx.translate(9.5,12.5);ctx.rotate(.07);ctx.scale(flipped?-1:1,1);if(flipped)ctx.translate(-900,0);
   ctx.beginPath();ctx.rect(0,0,850,490);ctx.clip();
   const state=()=>({alpha:ctx.globalAlpha,filter:ctx.filter,transform:Array.from(ctx.getTransform().toFloat64Array())}),before=state();
   const img=images.props,frames=Array.from({length:8},(_,i)=>[i%4*img.width/4,Math.floor(i/4)*img.height/2,img.width/4,img.height/2]);
   for(let i=0;i<frames.length;i++)for(const size of [64,128,256])drawAtlas(img,...frames[i],30+(i%4)*190,20+Math.floor(i/4)*220+size/8,size,Math.min(size,140));
   const after=state();ctx.restore();outputs.push({before,after,png:canvas.toDataURL()});
  }
  const names={scribe:'Scribe Without Eyes',remembers:'Abyss That Remembers'};
  const cells=new Map();
  for(const rig of AbyssCombatArt.rigs)for(const pose of Object.keys(AbyssCombatArt.poses))for(const i of [0,1]){
   const frame=bestiary.frame({name:names[rig]||rig,art_key:'shared-origin:'+rig,kind:'goblin'},pose,i);
   cells.set(frame.asset+':'+frame.row+':'+frame.column,frame);
  }
  for(const asset of bestiary.assets)for(const flip of [false,true]){
   ctx.setTransform(1,0,0,1,0,0);ctx.globalAlpha=1;ctx.filter='none';ctx.clearRect(0,0,960,540);ctx.fillStyle='#253d43';ctx.fillRect(0,0,960,540);
   ctx.save();ctx.translate(.5,.5);ctx.globalAlpha=.73;ctx.filter='brightness(1.3)';
   if(flip){ctx.translate(960,0);ctx.scale(-1,1);}
   for(const frame of cells.values())if(frame.asset===asset){const img=catalogImages[asset],s=frame.source;drawAtlas(img,s.x*img.width,s.y*img.height,s.width*img.width,s.height*img.height,20+frame.column*115,10+frame.row*65,58,58);}
   ctx.restore();outputs.push({png:canvas.toDataURL(),sharedCells:[...cells.values()].filter(f=>f.asset===asset).length});
  }
  // Prop prefixes preserve fractional coordinates; other fractional crops stay native.
  const propFrames=Array.from({length:8},(_,i)=>{
   const img=images.props,frame=cachedAtlasFrame(img,i%4*img.width/4,Math.floor(i/4)*img.height/2,img.width/4,img.height/2,128,128);
   return frame?{left:frame.left,top:frame.top,bytes:frame.bytes}:null;
  });
  const fractionalCached=!!cachedAtlasFrame(images.heroesA,20.25,20.125,128.5,128.25,100,100);
  for(const flipped of [false,true]){
   ctx.setTransform(1,0,0,1,0,0);ctx.globalAlpha=1;ctx.filter='none';ctx.clearRect(0,0,960,540);ctx.fillStyle='#253d43';ctx.fillRect(0,0,960,540);
   ctx.save();ctx.translate(480,240);ctx.rotate(.07);ctx.scale(flipped?-1:1,1);ctx.globalAlpha=.71;ctx.filter='brightness(1.3)';
   const before={alpha:ctx.globalAlpha,filter:ctx.filter};
   for(let i=0;i<6;i++)drawAtlas(images.heroesA,20.25+i*128,20.125,128.5,128.25,-430+i*145,-100,100,100);
   const after={alpha:ctx.globalAlpha,filter:ctx.filter};ctx.restore();outputs.push({before,after,png:canvas.toDataURL()});
  }
  // Exercise eviction with valid distinct rectangles, then draw again after reuse.
  const img=images.heroesA,retired=atlasFrames.values().next().value;
  for(let i=0;i<200;i++)drawAtlas(img,i,0,128,128,0,0,64,64);
  for(let i=0;i<40;i++)drawAtlas(img,i,0,512,512,0,0,64,64);
  const sharedFullRejected=cachedAtlasFrame(catalogImages[bestiary.assets[0]],0,0,156.75,158,80,80)===null;
  return {outputs,sharedPrefix,sharedFullRejected,propFrames,fractionalCached,cache:renderer.atlasCacheStats(),retired:retired?[retired.canvas.width,retired.canvas.height]:null};
 };`;
 const results=[];
 for(const source of [baseline,candidate]){
  await page.route('**/static/rift_renderer.js*',r=>r.fulfill({contentType:'application/javascript',body:source.replace('window.RiftRenderer=renderer;',hook+'window.RiftRenderer=renderer;')}));
  await page.goto('/abyss/rift?scenario=visual&seed=state-pixels&level=100&room=2&subclass=vanguard');await expect(page.locator('#rift-start')).toBeEnabled();
  expect(await page.evaluate(()=>document.getElementById('rift-canvas').getContext('2d').getContextAttributes().alpha)).toBe(!(opaque&&source===candidate));
  const data=await(await page.request.get('/api/abyss/rift')).json();
  const output=await page.evaluate(async run=>{await RiftRenderer.ready;return RiftRenderer.stateProbe(run.enemies);},data.run);
  output.outputs.forEach((item,index)=>fs.writeFileSync(info.outputPath('variant-'+results.length+'-'+index+'.png'),Buffer.from(item.png.split(',')[1],'base64')));
  results.push({outputs:output.outputs.map(item=>({...item,png:crypto.createHash('sha256').update(item.png).digest('hex')})),cache:output.cache,sharedPrefix:output.sharedPrefix,sharedFullRejected:output.sharedFullRejected,retired:output.retired,propFrames:output.propFrames,fractionalCached:output.fractionalCached});
  await page.unroute('**/static/rift_renderer.js*');
 }
 expect(results[1].outputs).toEqual(results[0].outputs);
 expect(results[1].outputs.filter(p=>p.sharedCells!==undefined).map(p=>p.sharedCells)).toEqual(Array(8).fill(64));
 expect(results[0].sharedPrefix).toBe(null);
 expect(results[1].sharedPrefix).toEqual(sharedOrigin?{left:0,top:0,bytes:315*1254*4}:null);
 expect(results[1].sharedFullRejected).toBe(true);
 expect(results[1].fractionalCached).toBe(false);
 expect(results[1].propFrames[0]).toEqual({left:0,top:0,bytes:445*445*4});
 expect(results[1].propFrames.some(frame=>frame===null)).toBe(true);
 for(const frame of results[1].propFrames.filter(Boolean)){
  expect(frame.left).toBe(0);expect(frame.top).toBe(0);expect(frame.bytes).toBeLessThanOrEqual(2*1024*1024);
 }
 expect(results[1].cache.entries).toBeLessThanOrEqual(64);expect(results[1].cache.bytes).toBeLessThanOrEqual(8*1024*1024);expect(results[1].cache.hits).toBeGreaterThan(0);expect(results[1].cache.misses).toBeGreaterThan(64);expect(results[1].retired).toEqual([0,0]);
});
