const test=require('node:test'),assert=require('node:assert/strict'),vm=require('node:vm');
const {actorLayerCandidate}=require('../../scripts/brawl-actor-layer-experiment.cjs');
const methods=['fill','stroke','fillRect','strokeRect','fillText','strokeText','drawImage','save','restore'];
const source=`  function actor(unit, now) { for(const name of methods)ctx[name]();if(unit.fail)throw Error('probe');return now; }
actor({},7);for(const name of methods)ctx[name]();`;
for(const [layer,omitted] of Object.entries({control:[],paths:['fill','stroke'],rectangles:['fillRect','strokeRect'],text:['fillText','strokeText'],images:['drawImage']}))test(layer+' omits only scoped draws and preserves outside drawing',()=>{
 const calls=[],ctx=Object.fromEntries(methods.map(method=>[method,()=>calls.push(method)]));
 const context={ctx,methods};vm.runInNewContext(actorLayerCandidate(source,layer),context);
 assert.deepEqual(calls,[...methods.filter(m=>!omitted.includes(m)),...methods]);
 assert.throws(()=>context.actor({fail:true},1),/probe/);calls.length=0;for(const method of methods)ctx[method]();assert.deepEqual(calls,methods);
});
test('unknown layer or ambiguous source is rejected',()=>{
 assert.throws(()=>actorLayerCandidate(source,'constructor'),/Unknown/);
 assert.throws(()=>actorLayerCandidate('', 'paths'),/boundary/);
 assert.throws(()=>actorLayerCandidate(source+source,'paths'),/boundary/);
});
