const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs'),vm=require('node:vm');
const context={window:{}};vm.runInNewContext(fs.readFileSync('internal/bot/webassets/rift_protocol.js','utf8'),context);
const shape=()=>({x:800,y:402.5,radius_x:740,radius_y:87.5});
function validate(round,entrance={x:160,y:402.5},exit={x:1440,y:402.5}){
 return context.window.RiftProtocol.validate({ok:true,run:null,build:{name:'Fighter',class:'warrior',skills:[],gear:[]},rooms:[],bestiary:[],levels:[{id:1,name:'Test',region_name:'Ruins',tactic:'Test',difficulty:'Wayfarer',region:0,rooms:Array.from({length:3},()=>({name:'Room',obstacles:[],hazards:[],round,entrance,exit}))}]},'GET');
}
test('circular floors preserve legacy and valid explicit anchors',()=>{
 assert.doesNotThrow(()=>validate(undefined));assert.doesNotThrow(()=>validate(shape()));
});
test('circular floor protocol rejects malformed shapes and unreachable anchors',()=>{
 for(const changes of [{radius_x:800},{radius_y:60},{y:450},{radius_y:NaN},{x:Infinity}])assert.throws(()=>validate({...shape(),...changes}),/incomplete expedition/);
 assert.throws(()=>validate(shape(),{x:160,y:320}),/incomplete expedition/);
 assert.throws(()=>validate(shape(),undefined,{x:1440,y:490}),/incomplete expedition/);
});
