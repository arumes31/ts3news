const {test,expect}=require('@playwright/test');

test('local mob transport rows preserve every frame under drawing transforms',async({page})=>{
 test.setTimeout(120000);
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.addScriptTag({url:'/static/rift_mob_sections.js'});
 const result=await page.evaluate(async()=>{
  const load=async src=>{const image=new Image();image.src=src;await image.decode();return image;};
  const make=()=>{const c=document.createElement('canvas');c.width=3072;c.height=1152;return c;};
  const originalOutput=make(),sectionOutput=make();let cases=0,frames=0;const mismatches=[];
  for(const key of ['mobs']){
   const atlas=RiftMobSections,original=await load(document.getElementById('rift-app').dataset[key]);
   const combined=document.createElement('canvas');combined.width=atlas.width;combined.height=atlas.height;
   const combine=combined.getContext('2d'),rows=[];
   for(const section of atlas.rows){const image=await load(section.url);if(image.width!==section.width||image.height!==section.height)throw Error('Section dimensions differ');combine.drawImage(image,0,section.y);rows.push(image);}
   const paint=(output,image,size,flip,filter,alpha)=>{
    const ctx=output.getContext('2d');ctx.setTransform(1,0,0,1,0,0);ctx.globalAlpha=1;ctx.filter='none';ctx.clearRect(0,0,output.width,output.height);ctx.fillStyle='#21362c';ctx.fillRect(0,0,output.width,output.height);ctx.imageSmoothingEnabled=false;
    for(let row=0;row<6;row++)for(let col=0;col<16;col++){
     ctx.save();ctx.beginPath();ctx.rect(col*192,row*192,192,192);ctx.clip();ctx.translate(col*192+96,row*192+96);ctx.rotate(.07);ctx.scale(flip?-1:1,1);ctx.globalAlpha=alpha;ctx.filter=filter;
     ctx.drawImage(Array.isArray(image)?image[row]:image,col*128,Array.isArray(image)?0:row*128,128,128,-size/2,-size/2,size,size);ctx.restore();
    }
    return ctx.getImageData(0,0,output.width,output.height).data;
   };
   for(const mode of ['combined','rows'])for(const size of [63,80,101,168])for(const flip of [false,true])for(const filter of ['none','brightness(1.3)'])for(const alpha of [1,.73]){
    const a=paint(originalOutput,original,size,flip,filter,alpha),b=paint(sectionOutput,mode==='combined'?combined:rows,size,flip,filter,alpha);let different=0;
    for(let i=0;i<a.length;i++)if(a[i]!==b[i])different++;
    if(different)mismatches.push({key,mode,size,flip,filter,alpha,different});cases++;frames+=96;
   }
  }
  return {cases,frames,mismatches};
 });
 expect(result.cases).toBe(64);expect(result.frames).toBe(6144);expect(result.mismatches).toEqual([]);
 console.log(JSON.stringify({cases:result.cases,frames:result.frames,pixelMismatches:result.mismatches.length}));
});
