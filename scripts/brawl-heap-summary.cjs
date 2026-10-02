// Counts are reachable nodes after GC, not dominator retained sizes. Preserve the
// original snapshot for retaining-path inspection; aggregate counts cannot prove
// which owner retains an object or whether a particular increase is a leak.
function summarizeHeap(heap){
 const fields=heap.snapshot.meta.node_fields,n=fields.length;
 const typeIndex=fields.indexOf('type'),nameIndex=fields.indexOf('name');
 const types=heap.snapshot.meta.node_types[typeIndex],byType=Object.create(null),objects=Object.create(null);
 const detachedIndex=fields.indexOf('detachedness');
 let detached=0;
 for(let i=0;i<heap.nodes.length;i+=n){
  const type=types[heap.nodes[i+typeIndex]],name=heap.strings[heap.nodes[i+nameIndex]];
  byType[type]=(byType[type]||0)+1;
  if(type==='object'||type==='native'||type==='closure')objects[name]=(objects[name]||0)+1;
  if(detachedIndex>=0&&heap.nodes[i+detachedIndex]===2)detached++;
 }
 return {nodes:heap.nodes.length/n,byType,detached,objects};
}

module.exports={summarizeHeap};
