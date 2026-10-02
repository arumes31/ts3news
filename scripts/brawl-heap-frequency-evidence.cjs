"use strict";
const crypto=require('node:crypto');
const {postCapEvidence}=require('./brawl-session-evidence.cjs');
const postMode=require('./brawl-session-options.cjs').sessionOptions({BRAWL_SESSION_POST_CAP:'1'}).mode;
const files=['tests/performance/rift-direct-memory.spec.js','scripts/brawl-direct-page.cjs','scripts/brawl-direct-cdp.cjs','scripts/brawl-direct-chromium.cjs','scripts/brawl-session-navigation.cjs'];
const integer=v=>Number.isSafeInteger(v)&&v>=0;
const canonical=v=>Array.isArray(v)?v.map(canonical):v&&typeof v==='object'?Object.fromEntries(Object.keys(v).sort().map(k=>[k,canonical(v[k])])):v;
const digest=v=>crypto.createHash('sha256').update(JSON.stringify(canonical(v))).digest('hex');
function heapFrequencyEvidence(reports){
 if(!Array.isArray(reports)||reports.length>2)throw Error('Expected up to two reports');
 const seen=new Set();
 const samples=reports.map(r=>{
  const policy=r?.heapSnapshotPolicy;
  if(!['all','endpoints'].includes(policy)||seen.has(policy))throw Error('Invalid or duplicate snapshot policy');seen.add(policy);
  // Reuse the complete gameplay/counter validator; verify this diagnostic's own mode separately.
  const base=postCapEvidence({...r,mode:postMode});
  const mode=policy==='all'?'direct browser paired instrumentation diagnostic':'direct browser endpoint-snapshot instrumentation diagnostic';
  const points=Array.isArray(r.checkpoints)?r.checkpoints:[];
  const snapshotsValid=points.length>=5&&points.every((p,i)=>{
   const last=i===points.length-1,wanted=policy==='all'||i===0||last;
   return p?.final===last&&p.snapshotTaken===wanted&&(wanted?integer(p.reachable?.nodes)&&p.reachable.nodes>0:p.reachable===undefined&&p.snapshot===undefined);
  });
  const healthValid=points.every(p=>integer(p?.health?.frames)&&p.health.frames>0&&Number.isFinite(p.health.frameAgeMS)&&p.health.frameAgeMS>=0&&p.health.frameAgeMS<5000&&p.health.hidden===false&&p.health.contextLost===false&&p.health.visibility==='visible'&&p.audio?.voices===0);
  const networkValid=r.networkInspection===false&&points.every(p=>p?.networkEvents===0);
  const driver=files.map(file=>{const found=(Array.isArray(r.driver)?r.driver:[]).filter(d=>d?.file===file);return found.length===1&&/^[a-f0-9]{64}$/.test(found[0].sha256)?found[0].sha256:null;});
  const p=r.profile||{},profileValid=p.width===1280&&p.height===900&&p.dpr===1&&p.cpuSlowdown===4&&p.headless===true&&p.physicalMinimumDevice===false;
  const comparison={browser:/^Chrome\/\d+\.\d+\.\d+\.\d+$/.test(r.browser?.product)?r.browser.product:null,driverHash:r.driver?.length===files.length&&driver.every(Boolean)?digest(driver):null,profileHash:profileValid?digest(p):null,settingsHash:r.settings?.fps===30&&r.settings?.particles===false?digest(r.settings):null,graphicsHash:Array.isArray(r.graphics?.devices)?digest(r.graphics.devices):null};
  const countersValid=base.checkpoints.length>1&&base.checkpoints.every(p=>integer(p.rendererPrivateBytes));
  const first=base.checkpoints[0],last=base.checkpoints.at(-1),cap=base.checkpoints.find(p=>p.mission===49);
  const rendererGrowthBytes=countersValid?last.rendererPrivateBytes-first.rendererPrivateBytes:null;
  const postCapRendererGrowthBytes=countersValid&&cap?last.rendererPrivateBytes-cap.rendererPrivateBytes:null;
  let status='incomplete';
  if(['runtime_failure','heap_limit_exceeded'].includes(base.status))status='failed';
  else if(r.mode===mode&&r.status==='capture_complete_review_required'&&base.status==='post_cap_review_required'&&snapshotsValid&&healthValid&&networkValid&&countersValid&&Object.values(comparison).every(Boolean))status='review_required';
  return {policy,status,source:base.source,comparison,measuredDurationMS:base.measuredDurationMS,completedReplays:base.completedReplays,runtimeErrors:base.runtimeErrors,growthBytes:base.growthBytes,postCapGrowthBytes:base.postCapGrowthBytes,rendererGrowthBytes,postCapRendererGrowthBytes,snapshotCount:points.filter(p=>p?.snapshotTaken===true).length,checkpoints:base.checkpoints};
 }).sort((a,b)=>Number(a.policy==='endpoints')-Number(b.policy==='endpoints'));
 let status='incomplete';
 if(samples.some(s=>s.status==='failed'))status='failed';
 else if(samples.length===2&&samples.every(s=>s.status==='review_required'))status=JSON.stringify([samples[0].source,samples[0].comparison])===JSON.stringify([samples[1].source,samples[1].comparison])?'paired_review_required':'comparison_mismatch';
 return {schema:'brawl-heap-frequency-evidence-v1',status,releaseReady:false,physicalDeviceVerified:false,scope:'Sequential all-checkpoint versus endpoint-only heap snapshots with Network disabled. Collection overhead, duration and run order require review; no causal or release claim. Endpoint retaining paths cannot establish an intermediate plateau.',allMinusEndpointsRendererGrowthBytes:status==='paired_review_required'?samples[0].rendererGrowthBytes-samples[1].rendererGrowthBytes:null,samples};
}
module.exports={heapFrequencyEvidence};

if(require.main===module){
 const fs=require('node:fs'),path=require('node:path');
 try{
  const dirs=process.argv.slice(2);if(dirs.length!==2)throw Error('Two capture directories required');
  const reports=dirs.map(dir=>{
   const files=fs.readdirSync(dir,{withFileTypes:true}).filter(d=>d.isDirectory()).map(d=>path.join(dir,d.name,'memory-report.json')).filter(file=>fs.existsSync(file));
   if(files.length!==1)throw Error('One report per capture directory required');
   return JSON.parse(fs.readFileSync(files[0],'utf8'));
  });
  const output=JSON.stringify(heapFrequencyEvidence(reports),null,2)+'\n';
  fs.writeFileSync(path.join(dirs[0],'public-heap-frequency-evidence.json'),output);process.stdout.write(output);
 }catch(_){console.error('Heap frequency export failed. Supply two capture directories containing one report each. Raw report contents were not printed.');process.exitCode=1;}
}
