// Public evidence contains only allowlisted numbers, hashes and fixed labels.
'use strict';
const integer=value=>Number.isSafeInteger(value)&&value>=0;
const hash=(value,length)=>typeof value==='string'&&new RegExp('^[a-f0-9]{'+length+'}$').test(value)?value:null;
function sessionEvidence(reports){
 if(!Array.isArray(reports)||reports.length>3)throw Error('Expected up to three session reports');
 const seen=new Set();
 const samples=reports.map(r=>{
  if(!r||!Number.isInteger(r.sample)||r.sample<1||r.sample>3||seen.has(r.sample))throw Error('Invalid or duplicate sample number');
  seen.add(r.sample);
  const source={revision:hash(r.server?.revision,40),trackedDiffSHA256:hash(r.server?.trackedDiffSHA256,64)};
  const points=(Array.isArray(r.checkpoints)?r.checkpoints:[]).map(p=>({elapsedMS:integer(p.elapsedMS)?p.elapsedMS:null,heapBytes:integer(p.heap?.usedSize)?p.heap.usedSize:null}));
  const first=points[0],last=points.at(-1);
  const growthBytes=points.length>=2&&first.heapBytes!==null&&last.heapBytes!==null?last.heapBytes-first.heapBytes:null;
  const runtimeErrors=Array.isArray(r.errors)?r.errors.length:null;
  const measuredDurationMS=integer(r.measuredDurationMS)?r.measuredDurationMS:null;
  const complete=typeof r.finishedAt==='string'&&/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$/.test(r.finishedAt)&&Number.isFinite(Date.parse(r.finishedAt))&&!r.error;
  let status='incomplete';
  if(runtimeErrors>0)status='runtime_failure';
  else if(complete&&r.mode==='30 minutes of repeated complete three-tier missions'&&source.revision&&source.trackedDiffSHA256&&runtimeErrors===0&&measuredDurationMS>=1800000&&points.length>=2&&points.every((p,i)=>p.heapBytes!==null&&p.elapsedMS!==null&&(i===0||p.elapsedMS>=points[i-1].elapsedMS))&&first.elapsedMS===0&&last.elapsedMS>=1800000&&last.elapsedMS<=measuredDurationMS){
   status=growthBytes>10*1048576?'heap_limit_exceeded':'heap_size_pass_review_pending';
  }
  return {sample:r.sample,source,status,measuredDurationMS,runtimeErrors,growthBytes,completedReplays:Array.isArray(r.expeditions)?r.expeditions.filter(e=>Number.isSafeInteger(e.index)&&e.index>0&&e.status==='complete').length:0,checkpoints:points};
 }).sort((a,b)=>a.sample-b.sample);
 let status='incomplete';
 if(samples.some(s=>['runtime_failure','heap_limit_exceeded'].includes(s.status)))status='failed';
 else if(samples.length===3&&samples.every(s=>s.status==='heap_size_pass_review_pending'))status=new Set(samples.map(s=>JSON.stringify(s.source))).size===1?'retention_review_required':'source_mismatch';
 return {schema:'brawl-session-evidence-v1',status,releaseReady:false,physicalDeviceVerified:false,scope:'Development session JS-heap measurements only. Retention paths, separate process memory, other performance gates and physical-device evidence require independent review.',samples};
}
module.exports={sessionEvidence};
if(require.main===module){
 const fs=require('node:fs'),path=require('node:path');
 const root=process.argv[2];
 if(!root){console.error('Usage: node scripts/brawl-session-evidence.cjs <capture-directory>');process.exitCode=1;}
 else try{
  const reports=fs.readdirSync(root,{withFileTypes:true}).filter(d=>d.isDirectory()).map(d=>path.join(root,d.name,'memory-report.json')).filter(p=>fs.existsSync(p)).map(p=>JSON.parse(fs.readFileSync(p,'utf8')));
  if(!reports.length)throw Error('No reports');
  const output=JSON.stringify(sessionEvidence(reports),null,2)+'\n';
  fs.writeFileSync(path.join(root,'public-session-evidence.json'),output);
  process.stdout.write(output);
 }catch(_){console.error('Evidence export failed: verify complete readable report files and unique sample numbers. No raw report contents were printed.');process.exitCode=1;}
}
