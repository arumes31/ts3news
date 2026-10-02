const {test,expect}=require('@playwright/test');

test('region panels preserve fractional crops and parallax pixels',async({page})=>{
 test.setTimeout(120000);
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.addScriptTag({url:'/static/rift_region_sections.js'});
 const result=await page.evaluate(async()=>{
  const load=async src=>{const img=new Image();img.src=src;await img.decode();return img;};
  const original=await load(document.getElementById('rift-app').dataset.regions),rows=[0,.179,.363,.559,.755,1];
  const outputs=[0,1].map(()=>{const c=document.createElement('canvas');c.width=960;c.height=540;return c;});
  let cases=0;const mismatches=[];
  const paint=(canvas,image,rect,x,flip,filter,alpha)=>{
   const ctx=canvas.getContext('2d');ctx.setTransform(1,0,0,1,0,0);ctx.globalAlpha=1;ctx.filter='none';ctx.clearRect(0,0,960,540);ctx.fillStyle='#091914';ctx.fillRect(0,0,960,540);ctx.imageSmoothingEnabled=false;
   ctx.save();ctx.beginPath();ctx.rect(0,0,960,540);ctx.clip();if(flip){ctx.translate(960,0);ctx.scale(-1,1);}ctx.globalAlpha=alpha;ctx.filter=filter;
   ctx.drawImage(image,...rect,x,0,Math.max(1184,960-x),540);ctx.restore();return ctx.getImageData(0,0,960,540).data;
  };
  for(let region=0;region<10;region++){
   const panel=RiftRegionSections.regions[region],section=await load(panel.url),row=Math.floor(region/2);
   const source=[region%2*original.width/2+2,rows[row]*original.height+2,original.width/2-4,(rows[row+1]-rows[row])*original.height-4];
   if(section.width!==panel.width||section.height!==panel.height)throw Error('Panel dimensions differ');
   for(const x of [0,-.35,-123.45,-739.2])for(const flip of [false,true])for(const filter of ['none','brightness(1.3)'])for(const alpha of [1,.73]){
    const a=paint(outputs[0],original,source,x,flip,filter,alpha),b=paint(outputs[1],section,panel.source,x,flip,filter,alpha);let different=0;
    for(let i=0;i<a.length;i++)if(a[i]!==b[i])different++;
    if(different)mismatches.push({region,x,flip,filter,alpha,different});cases++;
   }
  }
  return {cases,mismatches};
 });
 expect(result.cases).toBe(320);expect(result.mismatches).toEqual([]);
 console.log(JSON.stringify({cases:result.cases,pixelMismatches:result.mismatches.length}));
});
