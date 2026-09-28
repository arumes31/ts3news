'use strict';
const crypto=require('node:crypto');
const {postCapEvidence}=require('./brawl-session-evidence.cjs');
const postMode=require('./brawl-session-options.cjs').sessionOptions({BRAWL_SESSION_POST_CAP:'1'}).mode;
const files=['tests/performance/rift-direct-memory.spec.js','scripts/brawl-direct-page.cjs','scripts/brawl-direct-cdp.cjs','scripts/brawl-direct-chromium.cjs','scripts/brawl-session-navigation.cjs'];
const integer=v=>Number.isSafeInteger(v)&&v>=0;
const canonical=v=>Array.isArray(v)?v.map(canonical):v&&typeof v==='object'?Object.fromEntries(Object.keys(v).sort().map(k=>[k,canonical(v[k])])):v;
const digest=v=>crypto.createHash('sha256').update(JSON.stringify(canonical(v))).digest('hex');
function directMemoryEvidence(reports){
 if(!Array.isArray(reports)||reports.length>2)throw Error('Expected at most two paired reports');
 const seen=new Set();
 const samples=reports.map(r=>{
  if(!r||typeof r.networkInspection!=='boolean'||seen.has(r.networkInspection))throw Error('Invalid or duplicate inspection variant');seen.add(r.networkInspection);
  const completeMode=r.mode==='direct browser paired instrumentation diagnostic';
  const base=postCapEvidence({...r,mode:completeMode?postMode:''});
  const driver=files.map(file=>{const matches=(Array.isArray(r.driver)?r.driver:[]).filter(d=>d?.file===file);return matches.length===1&&/^[a-f0-9]{64}$/.test(matches[0].sha256)?matches[0].sha256:null;});
  const driverValid=r.driver?.length===files.length&&driver.every(Boolean);
  const browser=typeof r.browser?.product==='string'&&/^Chrome\/\d+\.\d+\.\d+\.\d+$/.test(r.browser.product)?r.browser.product:null;
  const p=r.profile||{},profileValid=p.width===1280&&p.height===900&&p.dpr===1&&p.cpuSlowdown===4&&p.headless===true&&p.physicalMinimumDevice===false;
  const settingsValid=r.settings?.fps===30&&r.settings?.particles===false;
  const events=(Array.isArray(r.checkpoints)?r.checkpoints:[]).map(p=>integer(p?.networkEvents)?p.networkEvents:null);
  const eventProof=events.length>=5&&events.every((v,i)=>v!==null&&(i===0||v>=events[i-1]))&&(r.networkInspection?events.at(-1)>events[0]:events.every(v=>v===0));
  const points=base.checkpoints,first=points[0],last=points.at(-1),cap=points.find(p=>p.mission===49);
  const processProof=points.length>1&&points.every(p=>integer(p.rendererPrivateBytes));
  const rendererGrowthBytes=processProof?last.rendererPrivateBytes-first.rendererPrivateBytes:null;
  const postCapRendererGrowthBytes=processProof&&cap?last.rendererPrivateBytes-cap.rendererPrivateBytes:null;
  const comparison={browser,driverHash:driverValid?digest(driver):null,profileHash:profileValid?digest(p):null,settingsHash:settingsValid?digest(r.settings):null,graphicsHash:Array.isArray(r.graphics?.devices)?digest(r.graphics.devices):null};
  let status='incomplete';
  if(base.status==='runtime_failure'||base.status==='heap_limit_exceeded')status='failed';
  else if(completeMode&&r.status==='capture_complete_review_required'&&base.status==='post_cap_review_required'&&Object.values(comparison).every(Boolean)&&eventProof&&processProof)status='review_required';
  return {networkInspection:r.networkInspection,status,source:base.source,comparison,measuredDurationMS:base.measuredDurationMS,runtimeErrors:base.runtimeErrors,completedReplays:base.completedReplays,growthBytes:base.growthBytes,postCapGrowthBytes:base.postCapGrowthBytes,rendererGrowthBytes,postCapRendererGrowthBytes,networkEvents:events.at(-1)??null,checkpoints:points};
 }).sort((a,b)=>Number(a.networkInspection)-Number(b.networkInspection));
 let status='incomplete';
 if(samples.some(s=>s.status==='failed'))status='failed';
 else if(samples.length===2&&samples.every(s=>s.status==='review_required'))status=JSON.stringify([samples[0].source,samples[0].comparison])===JSON.stringify([samples[1].source,samples[1].comparison])?'paired_review_required':'comparison_mismatch';
 return {schema:'brawl-direct-memory-evidence-v1',status,releaseReady:false,physicalDeviceVerified:false,scope:'One sequential pair with Network inspection off/on. Retaining paths, duration differences, run order and other instrumentation require review; this is not proof of causality or a release pass.',rendererGrowthDifferenceBytes:status==='paired_review_required'?samples[1].rendererGrowthBytes-samples[0].rendererGrowthBytes:null,samples};
}
module.exports={directMemoryEvidence};
if(require.main===module){
 const fs=require('node:fs'),path=require('node:path');
 try{const root=process.argv[2];if(!root)throw Error('Directory required');const reports=fs.readdirSync(root,{withFileTypes:true}).filter(d=>d.isDirectory()).map(d=>path.join(root,d.name,'memory-report.json')).filter(p=>fs.existsSync(p)).map(p=>JSON.parse(fs.readFileSync(p,'utf8')));if(!reports.length)throw Error('No reports');const output=JSON.stringify(directMemoryEvidence(reports),null,2)+'\n';fs.writeFileSync(path.join(root,'public-direct-memory-evidence.json'),output);process.stdout.write(output);}
 catch(_){console.error('Direct memory export failed. Check the capture directory and unique inspection variants. Raw report contents were not printed.');process.exitCode=1;}
}
