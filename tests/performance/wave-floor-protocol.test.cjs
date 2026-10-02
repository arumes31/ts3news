const {test}=require('node:test');
const assert=require('node:assert/strict'),fs=require('node:fs'),vm=require('node:vm');
const context={window:{}};vm.runInNewContext(fs.readFileSync('internal/bot/webassets/rift_protocol.js','utf8').replace('window.RiftProtocol={validate,hydrate,captureBase};','window.RiftProtocol={validate,hydrate,captureBase};window.testWaveFloor=waveFloorState;'),context);
function validate(fragile_floor,objective='survive_waves'){
 return context.window.RiftProtocol.validate({ok:true,run:null,build:{name:'Fighter',class:'warrior',skills:[],gear:[]},rooms:[],bestiary:[],levels:[{id:1,name:'Test',region_name:'Ruins',tactic:'Test',difficulty:'Wayfarer',region:0,rooms:Array.from({length:3},()=>({name:'Room',obstacles:[],hazards:[],objective,fragile_floor}))}]},'GET');
}
const panel=()=>({x:500,y:380,w:100,h:60});
test('wave-floor metadata accepts legacy and separated panels',()=>{
 assert.doesNotThrow(()=>validate(undefined));assert.doesNotThrow(()=>validate([panel(),{...panel(),x:900}]));
});
test('wave-floor metadata rejects invalid geometry and unrelated objectives',()=>{
 for(const change of [{x:NaN},{x:Infinity},{y:330},{w:30},{h:90},{x:1490}])assert.throws(()=>validate([{...panel(),...change}]),/incomplete expedition/);
 assert.throws(()=>validate([panel(),panel()]),/incomplete expedition/);
 assert.throws(()=>validate([panel()],'hold_circle'),/incomplete expedition/);
});

function active(){return {room:0,level:{rooms:[{fragile_floor:[panel()]}]},room_objective:{kind:'survive_waves',complete:false,next_wave_seconds:0,floor_segments:[{...panel(),collapsed:false,collapse_in:3}]}};}
test('live wave-floor state matches frozen panels and valid warning or collapsed states',()=>{
 const run=active();assert.equal(context.window.testWaveFloor(run),true);
 run.room_objective.floor_segments[0].collapse_in=0;run.room_objective.floor_segments[0].collapsed=true;assert.equal(context.window.testWaveFloor(run),true);
 run.room_objective.floor_segments[0].collapsed=false;run.room_objective.next_wave_seconds=2;assert.equal(context.window.testWaveFloor(run),true);
 assert.equal(context.window.testWaveFloor({room:0}),true);
});
test('live wave-floor state rejects mismatches and impossible recovery states',()=>{
 for(const change of [{collapse_in:-1},{collapse_in:4},{collapse_in:NaN},{collapse_in:Infinity},{collapsed:true},{collapsed:'true'},{x:501}]){
  const run=active();Object.assign(run.room_objective.floor_segments[0],change);assert.equal(context.window.testWaveFloor(run),false);
 }
 for(const field of ['complete','next_wave_seconds']){const run=active();run.room_objective[field]=field==='complete'?true:1;assert.equal(context.window.testWaveFloor(run),false);}
 const missing=active();missing.room_objective.floor_segments=[];assert.equal(context.window.testWaveFloor(missing),false);
 const wrong=active();wrong.room_objective.kind='hold_circle';assert.equal(context.window.testWaveFloor(wrong),false);
});

test('malformed empty panel collections are not treated as omitted legacy state',()=>{
 for(const value of [0,false,'']){
  const run={room:0,room_objective:{kind:'survive_waves',floor_segments:value}};
  assert.equal(context.window.testWaveFloor(run),false);
 }
});
