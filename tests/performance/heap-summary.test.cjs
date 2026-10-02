const {test}=require('node:test');
const assert=require('node:assert/strict');
const {summarizeHeap}=require('../../scripts/brawl-heap-summary.cjs');

test('snapshot counts preserve prototype-like names as numeric counters',()=>{
 const heap={snapshot:{meta:{node_fields:['type','name','detachedness'],node_types:[['object','closure','native']]}},strings:['constructor','__proto__','toString','AudioBuffer'],nodes:[0,0,0,0,0,0,0,1,0,1,2,0,2,3,2]};
 const result=summarizeHeap(heap);
 assert.equal(result.nodes,5);
 assert.equal(result.objects.constructor,2);
 assert.equal(result.objects.__proto__,1);
 assert.equal(result.objects.toString,1);
 assert.equal(result.objects.AudioBuffer,1);
 assert.equal(result.detached,1);
 assert.equal(result.byType.object,3);
 assert.equal(result.byType.closure,1);
 assert.equal(result.byType.native,1);
 for(const count of Object.values(result.objects))assert.equal(typeof count,'number');
});
