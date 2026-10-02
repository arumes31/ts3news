'use strict';
const {setTimeout:delay}=require('node:timers/promises');
async function attachDirectPage(cdp,{url,networkInspection=false}){
 const {targetId}=await cdp.send('Target.createTarget',{url:'about:blank'});
 const {sessionId}=await cdp.send('Target.attachToTarget',{targetId,flatten:true});
 const send=(method,params={},timeout)=>cdp.send(method,params,sessionId,timeout);
 const errors=[];let networkEvents=0;
 cdp.on('Runtime.exceptionThrown',()=>errors.push({kind:'runtime'}),sessionId);
 cdp.on('Runtime.consoleAPICalled',event=>{if(event.type==='error')errors.push({kind:'console'});},sessionId);
 cdp.on('Network.requestWillBeSent',()=>networkEvents++,sessionId);
 await send('Page.enable');await send('Runtime.enable');
 if(networkInspection)await send('Network.enable');
 await send('Emulation.setDeviceMetricsOverride',{width:1280,height:900,deviceScaleFactor:1,mobile:false});
 await send('Emulation.setCPUThrottlingRate',{rate:4});
 await send('Page.addScriptToEvaluateOnNewDocument',{source:`window.directCaptureErrors=[];const originalFetch=window.fetch;window.fetch=async function(...args){try{const response=await originalFetch.apply(this,args);if(response.status>=400)directCaptureErrors.push('http');return response;}catch(error){directCaptureErrors.push('fetch');throw error;}};window.addEventListener('error',event=>{if(event.target!==window)directCaptureErrors.push('resource');},true);`});
 const evaluate=async(fn,arg)=>{const result=await send('Runtime.evaluate',{expression:'('+fn.toString()+')('+JSON.stringify(arg??null)+')',awaitPromise:true,returnByValue:true},120000);if(result.exceptionDetails)throw Error('Direct page evaluation failed');return result.result.value;};
 const wait=async(predicate,timeout=120000)=>{const deadline=Date.now()+timeout;while(Date.now()<deadline){if(await predicate())return;await delay(100);}throw Error('Direct page condition timed out');};
 await send('Page.navigate',{url});
 await wait(()=>evaluate(()=>!!document.querySelector('#rift-start')&&!document.querySelector('#rift-start').disabled));
 const click=async(selector)=>{
  let point;await wait(async()=>{point=await evaluate(selector=>{const el=document.querySelector(selector);if(!el||el.disabled||!el.getClientRects().length)return null;el.scrollIntoView({block:'center'});const r=el.getBoundingClientRect();return {x:r.x+r.width/2,y:r.y+r.height/2};},selector);return !!point;});
  await send('Input.dispatchMouseEvent',{type:'mousePressed',button:'left',clickCount:1,...point});await send('Input.dispatchMouseEvent',{type:'mouseReleased',button:'left',clickCount:1,...point});
 };
 const held=new Set();
 const key=async(name,down)=>{const space=name==='Space';await send('Input.dispatchKeyEvent',{type:down?'keyDown':'keyUp',key:space?' ':name,code:space?'Space':'Key'+name.toUpperCase(),windowsVirtualKeyCode:space?32:name.toUpperCase().charCodeAt(0),...(down?{text:space?' ':name}:{})});};
 const controls=async wanted=>{for(const name of [...held])if(!wanted.has(name)){await key(name,false);held.delete(name);}for(const name of wanted)if(!held.has(name)){await key(name,true);held.add(name);}};
 const cookies=await cdp.send('Storage.getCookies');const cookie=cookies.cookies.map(c=>c.name+'='+c.value).join('; ');
 const read=async()=>{const response=await fetch(new URL('/api/abyss/rift',url),{headers:{cookie}});if(!response.ok)throw Error('Direct fixture API failed');return (await response.json()).run;};
 return {send,evaluate,wait,click,controls,read,errors,sessionId,get networkEvents(){return networkEvents;},async checkErrors(){const pageErrors=await evaluate(()=>directCaptureErrors);if(errors.length||pageErrors.length)throw Error('Direct capture runtime or HTTP failure');}};
}
async function startDirectExpedition(page,existing){
 // Mouse dispatch does not await the async begin handler or its server response.
 await page.wait(()=>page.evaluate(()=>{const button=document.getElementById('rift-start');return !!button&&!button.disabled;}));
 const replay=existing?.status==='complete';
 await page.click(replay?'#rift-replay':'#rift-start');
 await page.wait(async()=>{
  await page.checkErrors();const run=await page.read();
  return typeof run?.id==='string'&&run.id.length>0&&run.status==='fighting'&&run.paused===false&&(!replay||run.id!==existing.id);
 });
}
module.exports={attachDirectPage,startDirectExpedition};
