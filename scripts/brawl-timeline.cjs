"use strict";
const fs=require('node:fs');
const categories=['devtools.timeline','disabled-by-default-devtools.timeline','disabled-by-default-devtools.timeline.frame','toplevel','cc','gpu','v8','blink','disabled-by-default-blink.image_decoding','skia','disabled-by-default-skia','cc.debug'];
async function startTimeline(cdp,output,{timeoutMS=30000,maxBytes=256*1024*1024}={}){
 await cdp.send('Tracing.start',{transferMode:'ReturnAsStream',streamFormat:'json',traceConfig:{recordMode:'recordUntilFull',traceBufferSizeInKb:65536,includedCategories:categories}});
 let stopped;
 return {stop(){
  if(stopped)return stopped;
  stopped=(async()=>{
   let listener,timer,handle,fd;
   try{
    const complete=new Promise((resolve,reject)=>{
     listener=resolve;cdp.on('Tracing.tracingComplete',listener);
     timer=setTimeout(()=>reject(Error('Timeline completion timed out')),timeoutMS);
    });
    // Observe completion before ending: Chromium can emit the event immediately.
    const [,result]=await Promise.all([cdp.send('Tracing.end'),complete]);
    handle=result.stream;if(!handle)throw Error('Timeline stream missing');
    fd=fs.openSync(output,'w');let bytes=0;
    for(;;){
     const chunk=await cdp.send('IO.read',{handle,size:1024*1024});
     const data=Buffer.from(chunk.data,chunk.base64Encoded?'base64':'utf8');bytes+=data.length;
     if(bytes>maxBytes)throw Error('Timeline exceeds diagnostic size limit');
     fs.writeSync(fd,data);if(chunk.eof)break;
    }
    return {bytes,dataLossOccurred:result.dataLossOccurred,categories};
   }finally{
    clearTimeout(timer);if(listener)cdp.off('Tracing.tracingComplete',listener);
    if(fd!==undefined)fs.closeSync(fd);
    if(handle)await cdp.send('IO.close',{handle});
   }
  })();
  return stopped;
 }};
}
module.exports={startTimeline};
