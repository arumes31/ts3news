'use strict';
// Diagnostic omissions only. Never load this module from the application.
function actorLayerCandidate(source,layer,scope='actor'){
 if(!['actor','outside'].includes(scope))throw Error('Unknown drawing scope');
 const groups={control:[],paths:['fill','stroke'],rectangles:['fillRect','strokeRect'],text:['fillText','strokeText'],images:['drawImage']};
 if(!Object.hasOwn(groups,layer))throw Error('Unknown actor drawing layer');
 const marker='  function actor(unit, now) {';
 if(typeof source!=='string'||source.split(marker).length!==2)throw Error('Actor boundary changed');
 const injection=`  let diagnosticActorDepth=0;
  const diagnosticDrawCounts=renderer.actorLayerProbe={actorCalls:0,selectedCalls:0,omittedCalls:0};
  for(const method of ${JSON.stringify(groups[layer])}){
    const original=ctx[method];
    ctx[method]=function(...args){diagnosticDrawCounts.selectedCalls++;if(${scope==='actor'?'diagnosticActorDepth>0':'diagnosticActorDepth===0'}){diagnosticDrawCounts.omittedCalls++;return;}return original.apply(this,args);};
  }
  function actor(unit,now){
    diagnosticDrawCounts.actorCalls++;diagnosticActorDepth++;
    try{return diagnosticActor(unit,now);}finally{diagnosticActorDepth--;}
  }
  function diagnosticActor(unit, now) {`;
 return source.replace(marker,injection);
}
module.exports={actorLayerCandidate};
