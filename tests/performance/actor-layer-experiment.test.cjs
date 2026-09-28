const test=require('node:test'),assert=require('node:assert/strict'),vm=require('node:vm');
const {actorLayerCandidate}=require('../../scripts/brawl-actor-layer-experiment.cjs');
const methods=['fill','stroke','fillRect','strokeRect','fillText','strokeText','drawImage','save','restore'];
const source=`  function actor(unit, now) { for(const name of methods)ctx[name]();if(unit.fail)throw Error('probe');return now; }
actor({},7);for(const name of methods)ctx[name]();`;
for(const scope of ['actor','outside'])for(const [layer,omitted] of Object.entries({control:[],paths:['fill','stroke'],rectangles:['fillRect','strokeRect'],text:['fillText','strokeText'],images:['drawImage']}))test(scope+' '+layer+' omits only scoped draws and records calls',()=>{
 const calls=[],ctx=Object.fromEntries(methods.map(method=>[method,()=>calls.push(method)]));
 const context={ctx,methods,renderer:{}};vm.runInNewContext(actorLayerCandidate(source,layer,scope),context);
 assert.deepEqual(calls,scope==='actor'?[...methods.filter(m=>!omitted.includes(m)),...methods]:[...methods,...methods.filter(m=>!omitted.includes(m))]);
 assert.equal(context.renderer.actorLayerProbe.actorCalls,1);assert.equal(context.renderer.actorLayerProbe.selectedCalls,omitted.length*2);assert.equal(context.renderer.actorLayerProbe.omittedCalls,omitted.length);
 assert.throws(()=>context.actor({fail:true},1),/probe/);calls.length=0;for(const method of methods)ctx[method]();assert.deepEqual(calls,scope==='actor'?methods:methods.filter(m=>!omitted.includes(m)));
});
test('unknown layer or ambiguous source is rejected',()=>{
 assert.throws(()=>actorLayerCandidate(source,'constructor'),/Unknown/);
 assert.throws(()=>actorLayerCandidate(source,'paths','everywhere'),/scope/);
 assert.throws(()=>actorLayerCandidate('', 'paths'),/boundary/);
 assert.throws(()=>actorLayerCandidate(source+source,'paths'),/boundary/);
});
