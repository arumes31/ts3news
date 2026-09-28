'use strict';
// Public output is constructed from fixed labels, validated hashes and numbers.
const nonnegative=v=>Number.isFinite(v)&&v>=0;
const hash=(v,n)=>typeof v==='string'&&new RegExp('^[a-f0-9]{'+n+'}$').test(v)?v:null;
const percentile=(values,p)=>{const sorted=[...values].sort((a,b)=>a-b);return sorted[Math.ceil(sorted.length*p)-1]??null;};
function frameEvidence(kind,reports){
 if(!['crowd','boss'].includes(kind)||!Array.isArray(reports)||reports.length>3)throw Error('Invalid evidence input');
 const seen=new Set();
 const samples=reports.map(r=>{
  if(!r||!Number.isInteger(r.sample)||r.sample<1||r.sample>3||seen.has(r.sample))throw Error('Invalid or duplicate sample');
  seen.add(r.sample);
  const source={revision:hash(r.revision,40),trackedDiffSHA256:hash(r.trackedDiffSHA256,64)};
  const c=r.capture||{},timings=Array.isArray(c.samples)?c.samples:[],profile=r.profile||{};
  const durationMS=nonnegative(c.started)&&nonnegative(c.ended)&&c.ended>=c.started?c.ended-c.started:null;
  const validTimings=timings.length>0&&timings.every((s,i)=>s&&nonnegative(s.interval)&&nonnegative(s.render)&&nonnegative(s.at)&&s.at>=c.started&&s.at<=c.ended&&(i===0||s.at>=timings[i-1].at));
  const summary={frames:timings.length,intervalP95:null,intervalP99:null,renderP95:null};
  if(validTimings){summary.intervalP95=percentile(timings.map(s=>s.interval),.95);summary.intervalP99=percentile(timings.map(s=>s.interval),.99);summary.renderP95=percentile(timings.map(s=>s.render),.95);}
  const coveredMS=validTimings?timings.reduce((sum,s)=>sum+s.interval,0):null;
  const boundaryMS=validTimings?Math.max(...timings.map(s=>s.interval))+Math.max(...timings.map(s=>s.render)):0;
  const covered=validTimings&&durationMS!==null&&Math.abs(coveredMS-durationMS)<=boundaryMS;
  const runtimeErrors=Array.isArray(r.errors)?r.errors.length:null;
  const outcome=['cleared','defeated','complete'].includes(c.status)?c.status:null;
  const validProfile=profile.viewport?.width===1280&&profile.viewport?.height===900&&profile.dpr===1&&profile.cpuSlowdown===4&&profile.preset==='lowPower'&&profile.headless===true&&profile.physicalMinimumDevice===false;
  const fullDuration=durationMS!==null&&(durationMS>=60000||(kind==='boss'&&durationMS>0&&outcome!==null));
  const numericStatus=!validTimings?'unmeasured':summary.intervalP95>50||summary.intervalP99>100||summary.renderP95>16?'threshold_failure':'within_thresholds';
  let status='incomplete';
  if(runtimeErrors>0)status='runtime_failure';
  else if(r.smoke===true||r.profiling===true)status='diagnostic_only';
  else if(!r.failure&&r.smoke===false&&r.profiling===false&&source.revision&&source.trackedDiffSHA256&&runtimeErrors===0&&validProfile&&validTimings&&covered&&fullDuration&&c.hidden===false&&c.contextLost===false){
   status=numericStatus==='threshold_failure'?'threshold_failure':'development_numeric_pass';
  }
  const count=v=>Number.isSafeInteger(v)&&v>=0?v:null;
  return {sample:r.sample,source,status,numericStatus,durationMS,runtimeErrors,...summary,outcome,enemyPeak:count(c.enemyPeak),projectilePeak:count(c.projectilePeak)};
 }).sort((a,b)=>a.sample-b.sample);
 let status='incomplete';
 if(samples.some(s=>['runtime_failure','threshold_failure'].includes(s.status)))status='failed';
 else if(samples.length===3&&samples.every(s=>s.status==='development_numeric_pass'))status=new Set(samples.map(s=>JSON.stringify(s.source))).size===1?'development_numeric_pass':'source_mismatch';
 return {schema:'brawl-frame-evidence-v1',workload:kind,status,releaseReady:false,physicalDeviceVerified:false,thresholds:{intervalP95:50,intervalP99:100,renderP95:16},scope:'Numeric development-frame evidence only. Scenario execution, candidate server identity and physical hardware require separate verification. Raw timings remain local.',samples};
}
module.exports={frameEvidence};
if(require.main===module){
 const fs=require('node:fs'),path=require('node:path');
 try{
  const [kind,root]=process.argv.slice(2);if(!['boss','crowd'].includes(kind)||!root)throw Error('Arguments');
  const files=fs.readdirSync(root,{withFileTypes:true}).filter(d=>d.isDirectory()).map(d=>path.join(root,d.name,kind==='boss'?'frame-report.json':'crowd-report.json')).filter(p=>fs.existsSync(p));
  if(!files.length)throw Error('No reports');
  const output=JSON.stringify(frameEvidence(kind,files.map(p=>JSON.parse(fs.readFileSync(p,'utf8')))),null,2)+'\n';
  fs.writeFileSync(path.join(root,'public-frame-evidence.json'),output);process.stdout.write(output);
 }catch(_){console.error('Frame export failed. Use: node scripts/brawl-frame-evidence.cjs <boss|crowd> <capture-directory>. Raw report fields were not printed.');process.exitCode=1;}
}
