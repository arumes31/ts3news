const {test}=require('node:test');
const assert=require('node:assert/strict');
const {sessionEvidence}=require('../../scripts/brawl-session-evidence.cjs');
const report=()=>({sample:1,mode:'30 minutes of repeated complete three-tier missions',server:{revision:'a'.repeat(40),trackedDiffSHA256:'b'.repeat(64)},measuredDurationMS:1805000,finishedAt:'2026-09-28T00:30:05.000Z',errors:[],expeditions:[{index:'warmup',status:'complete'},{index:1,status:'complete'}],checkpoints:[{elapsedMS:0,heap:{usedSize:7000000}},{elapsedMS:1800000,heap:{usedSize:8000000}}]});
test('exports only aggregate evidence and never copies private free text',()=>{
 const r=report(),secret='PRIVATE_SENTINEL_PLAYER_UID';
 r.server.patch=secret;r.server.dirtyFiles=secret;r.host={username:secret};r.lastCombat={name:secret};r.errors=[];
 r.expeditions[1].id=secret;r.checkpoints[0].snapshot=secret;r.checkpoints[0].reachable={objects:{[secret]:1}};
 r.gate=secret;r.browser=secret;r.startedAt=secret;
 const e=sessionEvidence([r]);assert.equal(JSON.stringify(e).includes(secret),false);
 assert.equal(e.samples[0].status,'heap_size_pass_review_pending');assert.equal(e.samples[0].growthBytes,1000000);
 assert.equal(e.samples[0].completedReplays,1);assert.equal(e.status,'incomplete');
});
test('three numeric passes never imply retention or release approval',()=>{
 const reports=[1,2,3].map(sample=>({...report(),sample}));
 const e=sessionEvidence(reports);assert.equal(e.status,'retention_review_required');
 assert.equal(e.releaseReady,false);assert.equal(e.physicalDeviceVerified,false);
});
test('smoke, runtime failure and unfinished samples cannot pass',()=>{
 for(const [change,status] of [[{mode:'smoke (not a gate run)'},'incomplete'],[{errors:[{message:'PRIVATE_ERROR'}]},'runtime_failure'],[{finishedAt:undefined},'incomplete'],[{measuredDurationMS:60000},'incomplete'],[{error:'PRIVATE_ERROR'},'incomplete']]){
  const e=sessionEvidence([{...report(),...change}]);assert.equal(e.samples[0].status,status);assert.equal(JSON.stringify(e).includes('PRIVATE_ERROR'),false);
 }
});
test('computes growth from checkpoints and detects mismatched source',()=>{
 const r=report();r.growthBytes=0;r.checkpoints[1].heap.usedSize=30000000;
 assert.equal(sessionEvidence([r]).samples[0].status,'heap_limit_exceeded');
 const reports=[1,2,3].map(sample=>({...report(),sample}));reports[2].server={revision:'c'.repeat(40),trackedDiffSHA256:'d'.repeat(64)};
 assert.equal(sessionEvidence(reports).status,'source_mismatch');
});
test('duplicate sample numbers and missing or invalid evidence are rejected',()=>{
 assert.throws(()=>sessionEvidence([report(),report()]),/duplicate/);
 for(const change of [{errors:undefined},{checkpoints:[]},{server:{}},{measuredDurationMS:NaN}])assert.equal(sessionEvidence([{...report(),...change}]).samples[0].status,'incomplete');
});

test('a baseline alone cannot be reported as zero growth',()=>{
 const r=report();r.checkpoints.length=1;
 const e=sessionEvidence([r]);assert.equal(e.samples[0].growthBytes,null);assert.equal(e.samples[0].status,'incomplete');
});
