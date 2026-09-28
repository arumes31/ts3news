const {test}=require('node:test');
const assert=require('node:assert/strict'),fs=require('node:fs'),vm=require('node:vm');
const source=fs.readFileSync('internal/bot/webassets/rift.js','utf8');
const between=(first,last)=>{const a=source.indexOf(first),b=source.indexOf(last,a);assert(a>=0&&b>a);return source.slice(a,b);};
function setup(overrides={}){
 const timers=new Map([[1,{delay:85}]]);let next=1;
 const context=vm.createContext({playing:true,busy:false,loopRunning:false,pendingInput:false,checkpointPending:false,run:{status:'fighting'},inputMarks:null,performance:{now:()=>1000},timer:1,audio:{tick(){}},potions:null,countdownAnnounced:0,
  setTimeout:(callback,delay)=>{const id=++next;timers.set(id,{callback,delay});return id;},clearTimeout:id=>timers.delete(id),...overrides});
 vm.runInContext(between('  function markInput(', '  function takeInputTiming(')+between('  async function loop(){','  let pausePending='),context);
 return {context,timers,press:()=>vm.runInContext("markInput('jump')",context),loop:()=>vm.runInContext('loop()',context)};
}
test('fresh input replaces the idle timer without requiring diagnostics',()=>{
 const f=setup();f.press();assert.equal(f.context.pendingInput,true);assert.deepEqual([...f.timers.values()].map(t=>t.delay),[0]);
 f.press();assert.equal(f.timers.size,1);
});
for(const blocked of [{busy:true},{loopRunning:true},{checkpointPending:true}])test('input waits for current work: '+JSON.stringify(blocked),()=>{
 const f=setup(blocked);f.press();assert.equal(f.context.pendingInput,true);assert.deepEqual([...f.timers.values()].map(t=>t.delay),[85]);
});
for(const state of [{playing:false},{run:{status:'cleared'}}])test('inactive combat does not wake: '+JSON.stringify(state),()=>{
 const f=setup(state);f.press();assert.equal(f.context.pendingInput,false);assert.deepEqual([...f.timers.values()].map(t=>t.delay),[85]);
});
test('input arriving in flight waits for completion and schedules one immediate follow-up',async()=>{
 let finish,calls=0;
 const f=setup({send:()=>{calls++;return new Promise(resolve=>finish=resolve);}});f.timers.clear();
 const current=f.loop();f.press();const duplicate=f.loop();assert.equal(calls,1);await duplicate;assert.equal(f.timers.size,0);
 finish(true);await current;assert.equal(f.context.loopRunning,false);assert.deepEqual([...f.timers.values()].map(t=>t.delay),[0]);
});
test('ordinary idle responses keep the 85ms delay',async()=>{
 const f=setup({send:async()=>true});f.timers.clear();await f.loop();assert.deepEqual([...f.timers.values()].map(t=>t.delay),[85]);
});
test('pausing while a request is in flight prevents all follow-ups',async()=>{
 let finish;const f=setup({send:()=>new Promise(resolve=>finish=resolve)});f.timers.clear();
 const current=f.loop();f.press();f.context.playing=false;finish(true);await current;assert.equal(f.timers.size,0);assert.equal(f.context.loopRunning,false);
});
test('consuming controls clears the wake, including gamepad recognition during polling',()=>{
 const f=setup({loopRunning:true});const empty=()=>new Set();
 Object.assign(f.context,{keys:empty(),taps:empty(),mouse:empty(),touch:empty(),keyOrder:new Map(),guardLatched:false,touchJoystick:{x:0,y:0},controls:{codes:()=>[],toggleGuard:false},window:{RiftGamepad:{consume:()=>{f.press();return {actions:empty(),x:0,y:0};}},RiftJump:{input:()=>false},RiftIntents:{take:()=>({skill:'',wait:false})}}});
 vm.runInContext(between('  function input() {','  function resetInput()'),f.context);
 f.press();assert.equal(f.context.pendingInput,true);vm.runInContext('input()',f.context);assert.equal(f.context.pendingInput,false);
});
test('boss confirmation retains the idle timer even with a previous input wake',async()=>{
 const f=setup({run:{status:'cleared'},pendingInput:true,$:()=>({checked:true}),awaitingBossConfirmation:()=>true});f.timers.clear();await f.loop();
 assert.deepEqual([...f.timers.values()].map(t=>t.delay),[85]);assert.equal(f.context.loopRunning,false);
});
