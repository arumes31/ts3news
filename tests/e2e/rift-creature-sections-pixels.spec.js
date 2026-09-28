const {test,expect}=require('@playwright/test');

test('reconstructed creature rows preserve every shared frame under actor transforms',async({page})=>{
 test.setTimeout(180000);
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.addScriptTag({url:'/static/rift_creature_sections.js'});
 const result=await page.evaluate(async()=>{
  const art=AbyssCombatArt,manifest=RiftCreatureSections,images=new Map();
  const load=async src=>{if(images.has(src))return images.get(src);const image=new Image();image.src=src;await image.decode();images.set(src,image);return image;};
  const outputs=[0,1].map(()=>{const canvas=document.createElement('canvas');canvas.width=1280;canvas.height=192;return canvas;});
  const names={scribe:'Scribe Without Eyes',remembers:'Abyss That Remembers'};
  let cases=0,renderedFrames=0,mismatchCount=0;const mismatches=[],sourceCells=new Set();
  const paint=(canvas,image,rects,size,flip,filter,alpha,angle)=>{
   const ctx=canvas.getContext('2d');ctx.setTransform(1,0,0,1,0,0);ctx.globalAlpha=1;ctx.filter='none';ctx.clearRect(0,0,1280,192);ctx.fillStyle='#253d43';ctx.fillRect(0,0,1280,192);ctx.imageSmoothingEnabled=false;
   rects.forEach((rect,index)=>{ctx.save();ctx.beginPath();ctx.rect(index*160,0,160,192);ctx.clip();ctx.translate(index*160+80,96);ctx.rotate(angle);ctx.scale(flip?-1:1,1);ctx.globalAlpha=alpha;ctx.filter=filter;ctx.drawImage(image,...rect,-size/2,-size/2,size,size);ctx.restore();});
   return ctx.getImageData(0,0,1280,192).data;
  };
  for(const rig of art.rigs){
   const unit={name:names[rig]||rig,art_key:'bounds-probe:'+rig,kind:'goblin',element:'physical'},panel=manifest.rigs[rig];
   const frames=Object.keys(art.poses).flatMap(pose=>art.poses[pose].map((_,index)=>art.actorFrame(unit,pose,index)));
   if(frames.length!==8||frames.some(frame=>frame.rig!==rig||frame.asset!==panel.sourceAsset))throw Error('Shared provider and transport rig differ: '+rig);
   const original=await load(panel.sourceAsset),section=await load(panel.url);
   if(section.width!==panel.width||section.height!==panel.height)throw Error('Creature row dimensions differ');
   const source=frames.map(frame=>{const s=frame.source,rect=[s.x*original.width,s.y*original.height,s.width*original.width,s.height*original.height];sourceCells.add(frame.asset+JSON.stringify(rect));return rect;});
   // Keep the original atlas origin: shifted fractional sampling changes rotated pixels.
   const reconstructed=document.createElement('canvas');reconstructed.width=original.width;reconstructed.height=original.height;
   const copy=reconstructed.getContext('2d');copy.imageSmoothingEnabled=false;copy.drawImage(section,0,panel.top);
   for(const size of [61,113,168])for(const flip of [false,true])for(const filter of ['none','brightness(1.3)'])for(const alpha of [1,.73])for(const angle of [0,.07]){
    const a=paint(outputs[0],original,source,size,flip,filter,alpha,angle),b=paint(outputs[1],reconstructed,source,size,flip,filter,alpha,angle);let different=0;
    for(let i=0;i<a.length;i++)if(a[i]!==b[i])different++;
    if(different){mismatchCount++;if(mismatches.length<12)mismatches.push({rig,size,flip,filter,alpha,angle,different});}cases++;renderedFrames+=frames.length;
   }
  }
  let reconstructedSheets=0,reconstructionDifferences=0;
  for(const asset of art.atlasAssets){
   const original=await load(asset),merged=document.createElement('canvas'),reference=document.createElement('canvas');
   merged.width=reference.width=original.width;merged.height=reference.height=original.height;
   const target=merged.getContext('2d'),baseline=reference.getContext('2d');target.imageSmoothingEnabled=false;baseline.drawImage(original,0,0);
   // Reverse loading order and clear overlap before copying to avoid blending translucent borders twice.
   for(const panel of Object.values(manifest.rigs).filter(panel=>panel.sourceAsset===asset).reverse()){
    const section=await load(panel.url);target.clearRect(0,panel.top,panel.width,panel.height);target.drawImage(section,0,panel.top);
   }
   const a=baseline.getImageData(0,0,original.width,original.height).data,b=target.getImageData(0,0,original.width,original.height).data;
   for(let i=0;i<a.length;i++)if(a[i]!==b[i])reconstructionDifferences++;
   reconstructedSheets++;
  }
  return {cases,renderedFrames,sourceCells:sourceCells.size,mismatchCount,mismatches,reconstructedSheets,reconstructionDifferences};
 });
 console.log(JSON.stringify(result));
 expect(result.sourceCells).toBe(256);expect(result.cases).toBe(1536);expect(result.renderedFrames).toBe(12288);expect(result.mismatchCount).toBe(0);expect(result.mismatches).toEqual([]);expect(result.reconstructedSheets).toBe(4);expect(result.reconstructionDifferences).toBe(0);
});
