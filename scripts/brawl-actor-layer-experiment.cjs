'use strict';
// Diagnostic omissions only. Never load this module from the application.
function actorLayerCandidate(source,layer){
 const groups={control:[],paths:['fill','stroke'],rectangles:['fillRect','strokeRect'],text:['fillText','strokeText'],images:['drawImage']};
 if(!Object.hasOwn(groups,layer))throw Error('Unknown actor drawing layer');
 const marker='  function actor(unit, now) {';
 if(typeof source!=='string'||source.split(marker).length!==2)throw Error('Actor boundary changed');
 const injection=`  let diagnosticActorDepth=0;
  for(const method of ${JSON.stringify(groups[layer])}){
    const original=ctx[method];
    ctx[method]=function(...args){if(diagnosticActorDepth===0)return original.apply(this,args);};
  }
  function actor(unit,now){
    diagnosticActorDepth++;
    try{return diagnosticActor(unit,now);}finally{diagnosticActorDepth--;}
  }
  function diagnosticActor(unit, now) {`;
 return source.replace(marker,injection);
}
module.exports={actorLayerCandidate};
