'use strict';
// Test-only candidate. Keep fractional prop coordinates in their original atlas
// coordinate system while retaining the existing cache size/eviction limits.
function propOriginCandidate(input){
 const source=input.replace(/\r\n/g,'\n');
 const guard='if(![sx,sy,sw,sh].every(Number.isInteger))return null;';
 const origin='const left=Math.max(0,Math.floor(sx)-1),top=Math.max(0,Math.floor(sy)-1),';
 if(!source.includes(guard)||!source.includes(origin))throw Error('Prop-origin experiment source layout changed');
 return source.replace(guard,'if(img!==images.props&&![sx,sy,sw,sh].every(Number.isInteger))return null;').replace(origin,'const left=img===images.props?0:Math.max(0,Math.floor(sx)-1),top=img===images.props?0:Math.max(0,Math.floor(sy)-1),');
}
module.exports={propOriginCandidate};
