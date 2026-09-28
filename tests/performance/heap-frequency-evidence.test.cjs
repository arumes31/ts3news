const {test}=require('node:test'),assert=require('node:assert/strict');
const {heapFrequencyEvidence}=require('../../scripts/brawl-heap-frequency-evidence.cjs');
const files=['tests/performance/rift-direct-memory.spec.js','scripts/brawl-direct-page.cjs','scripts/brawl-direct-cdp.cjs','scripts/brawl-direct-chromium.cjs','scripts/brawl-session-navigation.cjs'];
function report(policy){return {heapSnapshotPolicy:policy,networkInspection:false,mode:policy==='all'?'direct browser paired instrumentation diagnostic':'direct browser endpoint-snapshot instrumentation diagnostic',status:'capture_complete_review_required',finishedAt:'2026-09-28T12:00:00.000Z',measuredDurationMS:3000000,errors:[],server:{revision:'a'.repeat(40),trackedDiffSHA256:'b'.repeat(64)},driver:files.map(file=>({file,sha256:'c'.repeat(64)})),browser:{product:'Chrome/153.0.8010.12'},graphics:{devices:[]},profile:{width:1280,height:900,dpr:1,cpuSlowdown:4,headless:true,physicalMinimumDevice:false},settings:{fps:30,particles:false},expeditions:[{index:'warmup',status:'complete',tiers:[{room:0},{room:1},{room:2}]},...Array.from({length:60},(_,i)=>({index:i+1,status:'complete',tiers:[{room:0},{room:1},{room:2}]}))],checkpoints:[0,49,54,59,60].map((mission,i)=>({mission,elapsedMS:i*700000,attemptHistoryCount:Math.min(50,mission+1),networkEvents:0,heap:{usedSize:7000000+i*10000},browserProcesses:[{type:'renderer',privateBytes:100000000+i*(policy==='all'?2000000:1000000)}],final:i===4,snapshotTaken:policy==='all'||i===0||i===4,...(policy==='all'||i===0||i===4?{reachable:{nodes:1000},snapshot:'PRIVATE_PATH'}:{}),health:{frames:1000,frameAgeMS:30,hidden:false,contextLost:false,visibility:'visible'},audio:{voices:0}}))};}
test('computes matched comparison from full workload without exposing raw fields',()=>{
 const a=report('all'),b=report('endpoints');a.secret='PRIVATE_COOKIE';a.checkpoints[0].reachable.objects={PRIVATE_OBJECT:1};
 const e=heapFrequencyEvidence([a,b]);assert.equal(e.status,'paired_review_required');assert.equal(e.allMinusEndpointsRendererGrowthBytes,4000000);assert.equal(e.samples[0].snapshotCount,5);assert.equal(e.samples[1].snapshotCount,2);assert.equal(e.releaseReady,false);assert.equal(JSON.stringify(e).includes('PRIVATE'),false);
});
test('rejects skipped control snapshots and extra endpoint snapshots',()=>{
 for(const [policy,change] of [['all',r=>r.checkpoints[1].snapshotTaken=false],['endpoints',r=>r.checkpoints[1].snapshotTaken=true],['endpoints',r=>delete r.checkpoints.at(-1).reachable],['endpoints',r=>r.checkpoints[1].final=true]]){const r=report(policy);change(r);assert.equal(heapFrequencyEvidence([r]).status,'incomplete');}
});
test('cannot qualify shortened, instrumented differently, unhealthy or unfinished runs',()=>{
 for(const change of [r=>r.mode='direct browser smoke; not a gate',r=>r.networkInspection=true,r=>r.checkpoints[1].networkEvents=1,r=>r.expeditions.pop(),r=>r.checkpoints.pop(),r=>r.profile.cpuSlowdown=1,r=>r.checkpoints[2].health.hidden=true,r=>r.checkpoints[2].audio.voices=1,r=>r.status='incomplete']){const r=report('all');change(r);assert.equal(heapFrequencyEvidence([r,report('endpoints')]).status,'incomplete');}
});
test('requires matching source, driver, graphics, browser and display settings',()=>{
 for(const change of [r=>r.server.revision='d'.repeat(40),r=>r.driver[0].sha256='e'.repeat(64),r=>r.graphics.devices=[{deviceId:1}],r=>r.browser.product='Chrome/154.0.1.1',r=>r.settings.enemyNames='all']){const r=report('endpoints');change(r);assert.equal(heapFrequencyEvidence([report('all'),r]).status,'comparison_mismatch');}
});
test('counts failures without publishing their messages; rejects duplicate policies',()=>{
 const r=report('all');r.errors=[{message:'PRIVATE_ERROR'}];const e=heapFrequencyEvidence([r]);assert.equal(e.status,'failed');assert.equal(JSON.stringify(e).includes('PRIVATE'),false);assert.throws(()=>heapFrequencyEvidence([report('all'),report('all')]));
});

const fs=require('node:fs'),path=require('node:path'),{execFileSync}=require('node:child_process');
function withCapturePair(fn){
 const root=fs.mkdtempSync(path.resolve(__dirname,'../../.tmp/heap-frequency-cli-'));
 const dirs=['all','endpoints'].map(name=>path.join(root,name));
 for(const [i,dir] of dirs.entries()){fs.mkdirSync(dir);fs.mkdirSync(path.join(dir,'capture'));fs.writeFileSync(path.join(dir,'capture','memory-report.json'),JSON.stringify(report(i?'endpoints':'all')));}
 try{return fn(dirs);}finally{
  for(const dir of dirs){const out=path.join(dir,'public-heap-frequency-evidence.json');if(fs.existsSync(out))fs.unlinkSync(out);fs.unlinkSync(path.join(dir,'capture','memory-report.json'));fs.rmdirSync(path.join(dir,'capture'));fs.rmdirSync(dir);}fs.rmdirSync(root);
 }
}
test('CLI exports numeric evidence from two explicit capture folders',()=>withCapturePair(dirs=>{
 const stdout=execFileSync(process.execPath,[path.resolve(__dirname,'../../scripts/brawl-heap-frequency-evidence.cjs'),...dirs],{encoding:'utf8'});
 const result=JSON.parse(stdout);assert.equal(result.status,'paired_review_required');assert.equal(stdout.includes('PRIVATE'),false);
 assert.deepEqual(JSON.parse(fs.readFileSync(path.join(dirs[0],'public-heap-frequency-evidence.json'),'utf8')),result);
}));
test('CLI fails without printing malformed private report contents',()=>withCapturePair(dirs=>{
 fs.writeFileSync(path.join(dirs[1],'capture','memory-report.json'),'PRIVATE_BROKEN_JSON');
 try{execFileSync(process.execPath,[path.resolve(__dirname,'../../scripts/brawl-heap-frequency-evidence.cjs'),...dirs],{encoding:'utf8',stdio:'pipe'});assert.fail('Expected failure');}
 catch(error){assert.equal(error.status,1);assert.equal(String(error.stderr).includes('PRIVATE'),false);assert.equal(String(error.stdout),'');assert.equal(fs.existsSync(path.join(dirs[0],'public-heap-frequency-evidence.json')),false);}
}));
