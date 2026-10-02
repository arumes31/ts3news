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

const wards=()=>[{element:'Fire',weakness:'Water'},{element:'Air',weakness:'Fire'},{element:'Earth',weakness:'Air'}];
function bossData(){const d=data();d.bestiary=[{id:'boss',name:'Boss',kind:'boss',x:300,y:410,hp:100,max_hp:100,facing:-1,tier:'Boss',art_key:'monster:Boss',phase:1,elemental_phases:wards()}];return d;}
test('boss phase wards accept all three current phases and legacy omission',()=>{
 for(const phase of [1,2,3]){const d=bossData();d.bestiary[0].phase=phase;assert.doesNotThrow(()=>validate(d));}
 const d=bossData();delete d.bestiary[0].elemental_phases;assert.doesNotThrow(()=>validate(d));
});
test('boss phase wards reject malformed arrays, values, nonboss actors and invalid phases',()=>{
 for(const bad of [null,{},[],wards().slice(0,2),[...wards(),wards()[0]],[{element:'Physical',weakness:'Water'},...wards().slice(1)],[{element:'Fire',weakness:{}},...wards().slice(1)]]){
  const d=bossData();d.bestiary[0].elemental_phases=bad;assert.throws(()=>validate(d),/incomplete expedition/);
 }
 for(const phase of [0,4,1.5,'2',null]){const d=bossData();d.bestiary[0].phase=phase;assert.throws(()=>validate(d),/incomplete expedition/);}
 const d=bossData();d.bestiary[0].kind='goblin';assert.throws(()=>validate(d),/incomplete expedition/);
});
