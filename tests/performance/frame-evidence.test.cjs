const {test}=require('node:test');
const assert=require('node:assert/strict');
const {frameEvidence}=require('../../scripts/brawl-frame-evidence.cjs');
const report=()=>({sample:1,revision:'a'.repeat(40),trackedDiffSHA256:'b'.repeat(64),smoke:false,profiling:false,errors:[],profile:{viewport:{width:1280,height:900},dpr:1,cpuSlowdown:4,preset:'lowPower',headless:true,physicalMinimumDevice:false},capture:{started:100,ended:60100,hidden:false,contextLost:false,samples:Array.from({length:1800},(_,i)=>({interval:100/3,render:10,at:100+(i+1)*100/3}))}});
test('derives frame percentiles and excludes private free text',()=>{
 const r=report(),secret='PRIVATE_PLAYER_COOKIE_PATH';r.host={name:secret};r.command=secret;r.capture.samples[0].secret=secret;r.summary={renderP95:999};r.gate=secret;
 const e=frameEvidence('crowd',[r]);assert.equal(e.samples[0].status,'development_numeric_pass');assert.equal(e.samples[0].renderP95,10);assert.equal(e.status,'incomplete');assert.equal(JSON.stringify(e).includes(secret),false);assert.equal(e.releaseReady,false);
});
test('three captures remain development evidence, never physical approval',()=>{
 const e=frameEvidence('crowd',[1,2,3].map(sample=>({...report(),sample})));
 assert.equal(e.status,'development_numeric_pass');assert.equal(e.physicalDeviceVerified,false);
});
test('slow frames cannot be hidden by a claimed pass or relaxed report thresholds',()=>{
 const r=report();r.gate='pass';r.thresholds={renderP95:1000};r.capture.samples.forEach(s=>s.render=25);
 assert.equal(frameEvidence('crowd',[r]).samples[0].status,'threshold_failure');
});
test('incomplete, malformed, wrong-profile and instrumented captures cannot pass',()=>{
 const changes=[r=>r.smoke=true,r=>r.profiling=true,r=>r.capture.ended=1000,r=>r.capture.hidden=true,r=>r.capture.contextLost=true,r=>r.capture.samples[1].interval=-1,r=>r.capture.samples[1].at=0,r=>r.capture.samples=[],r=>r.capture.samples=r.capture.samples.slice(0,1),r=>r.profile.dpr=2,r=>delete r.revision,r=>r.failure='PRIVATE_ERROR'];
 for(const change of changes){const r=report();change(r);const e=frameEvidence('crowd',[r]);assert.notEqual(e.samples[0].status,'development_numeric_pass');assert.equal(JSON.stringify(e).includes('PRIVATE_ERROR'),false);}
});
test('boss encounter shorter than a minute needs an actual terminal outcome',()=>{
 const r=report();r.capture.ended=30100;r.capture.samples=r.capture.samples.slice(0,900);r.capture.status='cleared';r.capture.enemyPeak=3;r.capture.projectilePeak=2;
 assert.equal(frameEvidence('boss',[r]).samples[0].status,'development_numeric_pass');
 r.capture.status=null;assert.equal(frameEvidence('boss',[r]).samples[0].status,'incomplete');
});
test('errors are counts only and source mismatch prevents aggregate pass',()=>{
 const r=report();r.errors=[{message:'PRIVATE_ERROR'}];const e=frameEvidence('crowd',[r]);assert.equal(e.samples[0].status,'runtime_failure');assert.equal(JSON.stringify(e).includes('PRIVATE_ERROR'),false);
 const rs=[1,2,3].map(sample=>({...report(),sample}));rs[2].revision='c'.repeat(40);assert.equal(frameEvidence('crowd',rs).status,'source_mismatch');
 assert.throws(()=>frameEvidence('crowd',[report(),report()]));assert.throws(()=>frameEvidence('unknown',[]));
});

test('legacy metadata gaps do not conceal measured threshold failures',()=>{
 const r=report();delete r.profiling;r.capture.samples.forEach(s=>s.render=25);
 const e=frameEvidence('crowd',[r]);assert.equal(e.samples[0].status,'incomplete');assert.equal(e.samples[0].numericStatus,'threshold_failure');
});
