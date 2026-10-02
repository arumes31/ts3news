const {test,expect}=require('@playwright/test');

test('prop sections preserve fractional source cells independently and after overlapping reconstruction',async({page})=>{
 test.setTimeout(120000);
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.addScriptTag({url:'/static/rift_prop_sections.js'});
 const result=await page.evaluate(async()=>{
  const load=async src=>{const img=new Image();img.src=src;await img.decode();return img;};
  const manifest=RiftPropSections,original=await load(document.getElementById('rift-app').dataset.props);
  const make=(width,height)=>{const c=document.createElement('canvas');c.width=width;c.height=height;return c;};
  const outputs=[make(400,300),make(400,300)],merged=make(manifest.width,manifest.height),images=[];
  let cases=0,differences=0,reconstructionDifferences=0;
  const mismatches=[];
  const paint=(canvas,image,index,width,height,flip,alpha,filter,angle)=>{
   const ctx=canvas.getContext('2d');ctx.setTransform(1,0,0,1,0,0);ctx.globalAlpha=1;ctx.filter='none';ctx.clearRect(0,0,400,300);ctx.fillStyle='#21362c';ctx.fillRect(0,0,400,300);ctx.imageSmoothingEnabled=false;
   ctx.save();ctx.translate(200.3,150.7);ctx.rotate(angle);ctx.scale(flip?-1:1,1);ctx.globalAlpha=alpha;ctx.filter=filter;
   ctx.drawImage(image,index%4*original.width/4,Math.floor(index/4)*original.height/2,original.width/4,original.height/2,-width/2,-height/2,width,height);ctx.restore();
   return ctx.getImageData(0,0,400,300).data;
  };
  for(let index=0;index<8;index++){
   const section=manifest.panels[index],img=await load(section.url);images.push(img);
   if(img.width!==section.width||img.height!==section.height)throw Error('Section dimensions differ');
   const isolated=make(manifest.width,manifest.height);isolated.getContext('2d').drawImage(img,section.left,section.top);
   for(const width of [58,90,134])for(const height of [76,138,200])for(const flip of [false,true])for(const alpha of [1,.37])for(const filter of ['none','brightness(1.3)'])for(const angle of [0,.07]){
    const a=paint(outputs[0],original,index,width,height,flip,alpha,filter,angle),b=paint(outputs[1],isolated,index,width,height,flip,alpha,filter,angle);let changed=0;
    for(let i=0;i<a.length;i++)if(a[i]!==b[i])changed++;
    if(changed){differences+=changed;if(mismatches.length<8)mismatches.push({index,width,height,flip,alpha,filter,angle,changed});}cases++;
   }
  }
  const reference=make(manifest.width,manifest.height),baseline=reference.getContext('2d');baseline.drawImage(original,0,0);
  const expected=baseline.getImageData(0,0,manifest.width,manifest.height).data,ctx=merged.getContext('2d');
  for(const order of [[0,1,2,3,4,5,6,7],[7,6,5,4,3,2,1,0]]){
   ctx.clearRect(0,0,manifest.width,manifest.height);
   for(const index of order){const s=manifest.panels[index];ctx.clearRect(s.left,s.top,s.width,s.height);ctx.drawImage(images[index],s.left,s.top);}
   const actual=ctx.getImageData(0,0,manifest.width,manifest.height).data;
   for(let i=0;i<expected.length;i++)if(expected[i]!==actual[i])reconstructionDifferences++;
  }
  return {cases,differences,mismatches,reconstructionDifferences};
 });
 console.log(JSON.stringify(result));expect(result.cases).toBe(1152);expect(result.differences).toBe(0);expect(result.reconstructionDifferences).toBe(0);
});
