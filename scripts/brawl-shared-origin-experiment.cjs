"use strict";
// Test-only candidate: preserve shared atlas coordinates under the existing limits.
function sharedOriginCandidate(source){
 const replacements=[
  ["const bestiary=window.RiftBestiary,catalogImages={},display=window.RiftDisplay;","const bestiary=window.RiftBestiary,catalogImages={},catalogAtlasImages=new WeakSet(),display=window.RiftDisplay;"],
  ["catalogImages[path]=img;","catalogImages[path]=img;catalogAtlasImages.add(img);"],
  ["if(img!==images.props&&![sx,sy,sw,sh].every(Number.isInteger))return null;","const sharedAtlas=catalogAtlasImages.has(img);const preserveOrigin=img===images.props||sharedAtlas;if(sharedAtlas&&sx+sw>img.width/4)return null;if(!preserveOrigin&&![sx,sy,sw,sh].every(Number.isInteger))return null;"],
  ["const left=img===images.props?0:","const left=preserveOrigin?0:"],
  ["top=img===images.props?0:","top=preserveOrigin?0:"],
  ["width=Math.min(img.width,Math.ceil(sx+sw)+1)-left", "width=sharedAtlas?Math.min(img.width,Math.ceil(img.width/4)+1):Math.min(img.width,Math.ceil(sx+sw)+1)-left"],
  ["height=Math.min(img.height,Math.ceil(sy+sh)+1)-top", "height=sharedAtlas?img.height:Math.min(img.height,Math.ceil(sy+sh)+1)-top"],
  ["    atlasMisses++;", "    if(sharedAtlas&&atlasFrameBytes+bytes>atlasFrameLimit)return null;\n    atlasMisses++;"]
 ];
 for(const [before,after] of replacements){if(source.split(before).length!==2)throw Error('Shared origin experiment source mismatch');source=source.replace(before,after);}
 return source;
}
module.exports={sharedOriginCandidate};
