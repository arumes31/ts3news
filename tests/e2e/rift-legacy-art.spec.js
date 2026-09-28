const {test,expect}=require('@playwright/test');
const fs=require('node:fs'),path=require('node:path');

for(const missing of ['image','source','frame'])test('legacy monster stays visible with missing '+missing,async({page})=>{
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 const source=fs.readFileSync(path.resolve(__dirname,'../../internal/bot/webassets/rift_renderer.js'),'utf8');
 const marker='  function drawAtlas(img,sx,sy,sw,sh,dx,dy,dw,dh){';expect(source.includes(marker)).toBe(true);
 await page.route('**/static/rift_renderer.js*',route=>route.fulfill({contentType:'application/javascript',body:source.replace(marker,marker+'\n    window.legacyAtlasProbe?.(img,sx,sy,sw,sh,dx,dy,dw,dh);')}));
 await page.goto('/abyss/rift?scenario=checkpoint');
 await expect(page.locator('#rift-start')).toBeEnabled();
 const result=await page.evaluate(missing=>{
  const ctx=document.getElementById('rift-canvas').getContext('2d');
  const draw=ctx.drawImage,frame=window.RiftBestiary.frame,calls=[],sources=[];
  window.legacyAtlasProbe=(img,...args)=>sources.push({src:img.src,args,width:img.width,height:img.height});
  const unit={id:'legacy-monster',name:'Retired Dragon',art_key:'monster:Retired Dragon',kind:'boss',element:'fire',x:450,y:410,hp:100,max_hp:100,facing:-1,pose:'attack',pose_time:.2};
  const before=JSON.stringify(unit);
  window.RiftBestiary.frame=(...args)=>{const value=frame(...args);return missing==='frame'?null:missing==='image'?{...value,asset:'/static/retired-missing.png'}:{...value,source:null};};
  ctx.drawImage=function(img,...args){calls.push({src:img.src,args,width:img.width,height:img.height});return draw.call(this,img,...args);};
  try{window.RiftRenderer.renderActor(unit,performance.now());}
  finally{ctx.drawImage=draw;window.RiftBestiary.frame=frame;delete window.legacyAtlasProbe;}
  return {calls,sources,unchanged:before===JSON.stringify(unit)};
 },missing);
 expect(result.unchanged).toBe(true);
 expect(result.calls.length).toBeGreaterThan(0);
 // Grid coordinates belong to the original sheet, before cache substitution.
 const sprite=result.sources.find(call=>call.src?.includes('rift_mobs.png'));
 for(const call of result.calls.filter(call=>call.args.length===8)){expect(call.args[0]).toBeGreaterThanOrEqual(0);expect(call.args[1]).toBeGreaterThanOrEqual(0);expect(call.args[0]+call.args[2]).toBeLessThanOrEqual(call.width);expect(call.args[1]+call.args[3]).toBeLessThanOrEqual(call.height);}
 expect(sprite).toBeTruthy();
 expect(sprite.args[0]).toBe(sprite.width/16*9);
 expect(sprite.args[1]).toBe(sprite.height/6*3);
 expect(sprite.args[1]).toBeGreaterThanOrEqual(0);
 expect(sprite.args[0]+sprite.args[2]).toBeLessThanOrEqual(sprite.width);
 expect(sprite.args[1]+sprite.args[3]).toBeLessThanOrEqual(sprite.height);
 expect(errors).toEqual([]);
});
