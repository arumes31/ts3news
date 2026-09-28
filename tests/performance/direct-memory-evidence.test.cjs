const {test}=require('node:test'),assert=require('node:assert/strict');
const {directMemoryEvidence}=require('../../scripts/brawl-direct-memory-evidence.cjs');
const files=['tests/performance/rift-direct-memory.spec.js','scripts/brawl-direct-page.cjs','scripts/brawl-direct-cdp.cjs','scripts/brawl-direct-chromium.cjs','scripts/brawl-session-navigation.cjs'];
const report=networkInspection=>({networkInspection,sample:networkInspection?2:1,mode:'direct browser paired instrumentation diagnostic',status:'capture_complete_review_required',finishedAt:'2026-09-28T12:00:00.000Z',measuredDurationMS:3000000,errors:[],server:{revision:'a'.repeat(40),trackedDiffSHA256:'b'.repeat(64)},driver:files.map(file=>({file,sha256:'c'.repeat(64)})),browser:{product:'Chrome/153.0.8010.12'},graphics:{devices:[]},profile:{width:1280,height:900,dpr:1,cpuSlowdown:4,headless:true,physicalMinimumDevice:false},settings:{fps:30,particles:false},expeditions:[{index:'warmup',status:'complete',tiers:[{room:0},{room:1},{room:2}]},...Array.from({length:60},(_,i)=>({index:i+1,status:'complete',tiers:[{room:0},{room:1},{room:2}]}))],checkpoints:[0,49,54,59,60].map((mission,i)=>({mission,elapsedMS:i*700000,attemptHistoryCount:Math.min(50,mission+1),networkEvents:networkInspection?100+i*100:0,heap:{usedSize:7000000+i*10000},browserProcesses:[{type:'renderer',privateBytes:100000000+i*(networkInspection?2000000:1000000)}]}))});
test('exports only aggregate paired evidence, with independently computed growth',()=>{
 const a=report(false),b=report(true),secret='PRIVATE_COOKIE_USER_PATH';a.launchArguments=[secret];a.errors=[];a.driver[0].source=secret;a.checkpoints[0].snapshot=secret;b.server.private=secret;
 const e=directMemoryEvidence([a,b]);assert.equal(e.status,'paired_review_required');assert.equal(e.samples[0].rendererGrowthBytes,4000000);assert.equal(e.samples[1].rendererGrowthBytes,8000000);assert.equal(e.rendererGrowthDifferenceBytes,4000000);assert.equal(JSON.stringify(e).includes(secret),false);assert.equal(e.releaseReady,false);assert.equal(e.physicalDeviceVerified,false);
});
test('smoke, incomplete histories, missing provenance and wrong profiles cannot qualify',()=>{
 for(const change of [r=>r.mode='direct browser smoke; not a gate',r=>r.expeditions.pop(),r=>r.checkpoints.pop(),r=>r.driver.pop(),r=>r.profile.dpr=2,r=>r.status='incomplete',r=>r.browser.product='PRIVATE_VERSION',r=>r.checkpoints[2].browserProcesses=[],r=>r.checkpoints[1].networkEvents=2]){const r=report(false);change(r);assert.equal(directMemoryEvidence([r,report(true)]).status,'incomplete');}
});
test('inspection event evidence and matching driver/browser settings are mandatory',()=>{
 const r=report(true);r.checkpoints.forEach(p=>p.networkEvents=0);assert.equal(directMemoryEvidence([report(false),r]).status,'incomplete');
 for(const change of [r=>r.driver[0].sha256='d'.repeat(64),r=>r.browser.product='Chrome/154.0.1.1',r=>r.settings.damageNumbers=true]){const r=report(true);change(r);assert.equal(directMemoryEvidence([report(false),r]).status,'comparison_mismatch');}
});
test('duplicate variants are rejected and runtime failures do not publish error text',()=>{
 assert.throws(()=>directMemoryEvidence([report(false),report(false)]));const r=report(true);r.errors=[{message:'PRIVATE_ERROR'}];const e=directMemoryEvidence([report(false),r]);assert.equal(e.status,'failed');assert.equal(JSON.stringify(e).includes('PRIVATE_ERROR'),false);
});


test('network comparison cannot qualify a changed heap snapshot policy',()=>{
 for(const policy of ['endpoints','none','unknown']){
  const r=report(false);r.heapSnapshotPolicy=policy;
  assert.equal(directMemoryEvidence([r,report(true)]).status,'incomplete');
 }
 const r=report(false);r.heapSnapshotPolicy='all';
 assert.equal(directMemoryEvidence([r,report(true)]).status,'paired_review_required');
});
