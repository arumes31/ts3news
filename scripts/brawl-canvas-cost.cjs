'use strict';
// Serialized into a synthetic fixture page by the performance harness only.
function installCanvasCostProbe(options={}){
 const ctx=options.context||document.querySelector('#rift-canvas').getContext('2d');
 const clock=options.clock||(()=>performance.now());
 const identify=options.identify||(image=>{const match=String(image?.src||'').match(/\/static\/([a-zA-Z0-9_-]+\.(?:png|webp))/);return match?match[1]:'canvas '+image?.width+'x'+image?.height;});
 const originals={},descriptors={},rows=new Map();let lastImage='none',lastDrawState={filter:'none',alpha:1,composite:'source-over'},stopped=false;
 for(const operation of ['drawImage','save','restore']){
  const original=ctx[operation];originals[operation]=original;descriptors[operation]=Object.getOwnPropertyDescriptor(ctx,operation);
  ctx[operation]=function(...args){
   if(operation==='drawImage'){lastImage=identify(args[0]);lastDrawState={filter:this.filter||'none',alpha:this.globalAlpha??1,composite:this.globalCompositeOperation||'source-over'};}
   const image=lastImage,drawState=lastDrawState,key=operation+'|'+image+'|'+JSON.stringify(drawState),started=clock();
   try{return original.apply(this,args);}finally{
    const ms=clock()-started;let row=rows.get(key);
    if(!row){row={operation,image,drawState,calls:0,totalMS:0,maxMS:0,slowCalls:0};rows.set(key,row);}
    row.calls++;row.totalMS+=ms;row.maxMS=Math.max(row.maxMS,ms);if(ms>=8)row.slowCalls++;
   }
  };
 }
 const probe={stop(){
  if(!stopped){for(const operation of Object.keys(originals)){if(descriptors[operation])Object.defineProperty(ctx,operation,descriptors[operation]);else delete ctx[operation];}stopped=true;}
  return [...rows.values()].map(row=>({...row})).sort((a,b)=>b.totalMS-a.totalMS);
 }};
 if(typeof window!=='undefined')window.brawlCanvasCost=probe;
 return options.context?probe:undefined;
}
module.exports={installCanvasCostProbe};
