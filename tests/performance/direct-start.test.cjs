const {test}=require('node:test'),assert=require('node:assert/strict');
const {startDirectExpedition}=require('../../scripts/brawl-direct-page.cjs');
function pageFor(states){let index=0,readyChecks=0;const clicks=[];return {clicks,get reads(){return index;},async evaluate(){return ++readyChecks>1;},async click(selector){assert.ok(readyChecks>1,'wait for the previous UI request to settle');clicks.push(selector);},async read(){return states[Math.min(index++,states.length-1)];},async checkErrors(){},async wait(predicate){for(let i=0;i<10;i++)if(await predicate())return;throw Error('condition timed out');}};}
test('replay waits through old completion, same identity and paused replacement',async()=>{
 const old={id:'old',status:'complete'},fresh={id:'new',status:'fighting',paused:false};
 const page=pageFor([old,old,{id:'old',status:'fighting',paused:false},{...fresh,paused:true},fresh]);
 await startDirectExpedition(page,old);assert.equal(page.reads,5);assert.deepEqual(page.clicks,['#rift-replay']);
});
test('initial start and resume wait for confirmed unpaused gameplay',async()=>{
 for(const existing of [null,{id:'same',status:'fighting',paused:true}]){
  const page=pageFor([{id:'same',status:'fighting',paused:true},{id:'same',status:'fighting',paused:false}]);
  await startDirectExpedition(page,existing);assert.equal(page.reads,2);assert.deepEqual(page.clicks,['#rift-start']);
 }
});
test('a replay that never begins still fails instead of counting the old completion',async()=>{
 const old={id:'old',status:'complete'},page=pageFor([old]);await assert.rejects(startDirectExpedition(page,old),/condition timed out/);
});
test('runtime errors during acknowledgement are not suppressed',async()=>{
 const page=pageFor([{id:'new',status:'fighting',paused:false}]);page.checkErrors=async()=>{throw Error('runtime failure');};await assert.rejects(startDirectExpedition(page,{id:'old',status:'complete'}),/runtime failure/);
});
