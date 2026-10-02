const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs'),os=require('node:os'),path=require('node:path');
const {spawnSync}=require('node:child_process');
function inspect(checkpointKey,id='current',weakOnly=false,diagnostics=false){
 const root=fs.mkdtempSync(path.join(os.tmpdir(),'brawl-retention-'));
 try{
  const sample=path.join(root,'sample');fs.mkdirSync(sample);
  // Root -> context -> run; a weak owner must not be reported as a retainer.
  const strings=['(root)','system / Context','Object','player','enemies','status','id',id,'run','weak-owner'];
  const heap={snapshot:{meta:{node_fields:['type','name','edge_count'],node_types:[['synthetic','object','string']],edge_fields:['type','name_or_index','to_node'],edge_types:[['property','internal','weak']]}},strings,
   nodes:[0,0,1,1,1,1,1,2,4,2,7,0,1,9,1],
   edges:[1,8,3,weakOnly?2:1,8,6,0,3,9,0,4,9,0,5,9,0,6,9,2,8,6]};
  if(diagnostics){
   heap.snapshot.meta.node_fields.push('self_size','detachedness');
   const original=heap.nodes;heap.nodes=[];
   for(let i=0;i<original.length;i+=3)heap.nodes.push(...original.slice(i,i+3),0,0);
   for(let i=2;i<heap.edges.length;i+=3)heap.edges[i]=heap.edges[i]/3*5;
   heap.snapshot.meta.node_types[0].push('native');
   strings.push('blink::NetworkResourcesData::ResourceData','Detached HTMLDivElement');
   heap.nodes.push(3,10,0,288,0,3,11,0,64,2);
   heap.nodes[7]+=2;
   heap.edges.splice(6,0,1,8,25,1,8,30);
   heap.nodes.push(1,2,6,32,0);heap.nodes[7]++;
   heap.edges.splice(12,0,1,8,35);
   for(const key of ['mission','outcome','at_ms','splits','hp','class']){strings.push(key);heap.edges.push(0,strings.length-1,15);}
  }
  const snapshot=path.join(sample,'heap.heapsnapshot');fs.writeFileSync(snapshot,JSON.stringify(heap));
  const report={sample:1,checkpoints:[{[checkpointKey]:0,elapsedMS:0,snapshot,reachable:{detached:0},dom:{documents:1,nodes:4,jsEventListeners:0},heap:{usedSize:100}}],expeditions:[{index:'warmup',id:'current'}]};
  fs.writeFileSync(path.join(sample,'memory-report.json'),JSON.stringify(report));
  const result=spawnSync('python',[path.resolve(__dirname,'../../scripts/analyze-brawl-restart-memory.py'),root],{encoding:'utf8'});
  const output=path.join(root,'retention-review.json');return {...result,review:fs.existsSync(output)?JSON.parse(fs.readFileSync(output,'utf8')):null};
 }finally{fs.rmSync(root,{recursive:true,force:true});}
}
for(const key of ['cycle','mission'])test('retention review accepts '+key+' checkpoints',()=>{
 const result=inspect(key);assert.equal(result.status,0,result.stderr);
 const point=result.review[0].checkpoints[0];assert.equal(point[key],0);assert.equal(point.shapes.runShapes,1);
 assert.equal(point.runRetainers.length,1);assert.equal(point.runRetainers[0].ownerName,'system / Context');
 assert.equal(point.runRootPaths[0].status,'found');
 assert.deepEqual(point.runRootPaths[0].path.map(p=>p.name),['(root)','system / Context','Object']);
 assert.deepEqual(point.runRootPaths[0].path.slice(1).map(p=>p.via.edgeType),['internal','internal']);
});
test('retention review rejects a previous expedition still retained',()=>{
 const result=inspect('mission','older');assert.notEqual(result.status,0);assert.match(result.stderr,/Retained run IDs differ/);assert.equal(result.review,null);
});

test('weak-only paths are not reported as retaining roots',()=>{
 const result=inspect('mission','current',true);assert.equal(result.status,0,result.stderr);
 const point=result.review[0].checkpoints[0];assert.equal(point.runRetainers.length,0);
 assert.equal(point.runRootPaths[0].status,'no non-weak root path');assert.deepEqual(point.runRootPaths[0].path,[]);
});

test('retention review reports native resource and detached-node root paths',()=>{
 const result=inspect('mission','current',false,true);assert.equal(result.status,0,result.stderr);
 const point=result.review[0].checkpoints[0];
 assert.equal(point.networkResources.count,1);assert.equal(point.networkResources.selfBytes,288);
 assert.equal(point.networkResources.examplePaths[0].status,'found');
 assert.deepEqual(point.networkResources.examplePaths[0].path.map(p=>p.name),['(root)','system / Context','blink::NetworkResourcesData::ResourceData']);
 assert.equal(point.attemptRootPaths.length,1);assert.equal(point.attemptRootPaths[0].status,'found');
 assert.equal(point.detachedNodes.count,1);assert.equal(point.detachedNodes.examplePaths[0].status,'found');
});
