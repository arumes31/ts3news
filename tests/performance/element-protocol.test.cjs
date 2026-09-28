const {test}=require('node:test');
const assert=require('node:assert/strict');
const vm=require('node:vm'),fs=require('node:fs');
const context={window:{}};
vm.runInNewContext(fs.readFileSync('internal/bot/webassets/rift_protocol.js','utf8'),context);
function data(){return {ok:true,run:null,build:{name:'Test',class:'warrior',skills:[{id:'test',name:'Test',cost:12,cooldown:2}],gear:[]},rooms:[],bestiary:[],levels:[{id:1,name:'Test',region_name:'Ruins',tactic:'Test',difficulty:'Wayfarer',region:0,rooms:Array.from({length:3},()=>({name:'Room',obstacles:[],hazards:[]}))}]};}
const validate=value=>context.window.RiftProtocol.validate(value,'GET');
test('element metadata accepts absent legacy fields and string-valued source metadata',()=>{
 for(const element of [undefined,'','Physical','Fire','Water','Earth','Air','future-element']){
  const d=data();d.build.weapon_element=element;d.build.skills[0].element=element;assert.doesNotThrow(()=>validate(d));
 }
});
test('element metadata rejects objects, arrays, numbers, booleans and null',()=>{
 for(const value of [{},[],12,true,null])for(const field of ['weapon','skill','signature','ultimate']){
  const d=data();
  if(field==='weapon')d.build.weapon_element=value;
  if(field==='skill')d.build.skills[0].element=value;
  if(field==='signature')d.build.signatures=[{...d.build.skills[0],element:value}];
  if(field==='ultimate')d.build.ultimate={...d.build.skills[0],element:value};
  assert.throws(()=>validate(d),/incomplete expedition/);
 }
});
