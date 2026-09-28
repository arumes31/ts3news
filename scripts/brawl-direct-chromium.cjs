'use strict';
const fs=require('node:fs'),path=require('node:path');
const {spawn}=require('node:child_process');
const {setTimeout:delay}=require('node:timers/promises');
const {connectCDP}=require('./brawl-direct-cdp.cjs');
async function launchDirectChromium({executablePath,outputDirectory}){
 fs.mkdirSync(outputDirectory,{recursive:true});
 const profile=fs.mkdtempSync(path.join(path.resolve(outputDirectory),'chromium-'));
 const args=['--headless=new','--remote-debugging-port=0','--remote-debugging-address=127.0.0.1','--user-data-dir='+profile,'--no-first-run','--no-default-browser-check','--disable-background-networking','--disable-extensions','--disable-component-update','--window-size=1280,900','about:blank'];
 const child=spawn(executablePath,args,{windowsHide:true,stdio:['ignore','ignore','pipe']});
 let launchError;child.on('error',error=>launchError=error);
 let stderr='';child.stderr.on('data',chunk=>{stderr=(stderr+chunk).slice(-16384);});
 let cdp;
 try{
  const active=path.join(profile,'DevToolsActivePort'),deadline=Date.now()+30000;
  while(!fs.existsSync(active)){if(launchError)throw launchError;if(child.exitCode!==null)throw Error('Isolated Chromium exited during launch');if(Date.now()>deadline)throw Error('Isolated Chromium launch timed out');await delay(100);}
  const [port,socketPath]=fs.readFileSync(active,'utf8').trim().split(/\r?\n/);
  if(!/^\d+$/.test(port)||!socketPath?.startsWith('/devtools/browser/'))throw Error('Invalid isolated Chromium endpoint');
  cdp=await connectCDP('ws://127.0.0.1:'+port+socketPath);
  const version=await cdp.send('Browser.getVersion');
  let closed=false;
  return {cdp,version,profile,args,async close(){
   if(closed)return;closed=true;
   try{await cdp.send('Browser.close',{},undefined,5000);}catch{}finally{cdp.close();}
   const deadline=Date.now()+5000;while(child.exitCode===null&&Date.now()<deadline)await delay(50);
   if(child.exitCode===null)child.kill();
   fs.writeFileSync(path.join(profile,'launcher-stderr.txt'),stderr);
  }};
 }catch(error){cdp?.close();if(child.exitCode===null)child.kill();throw error;}
}
module.exports={launchDirectChromium};
