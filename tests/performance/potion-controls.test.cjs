const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs'),vm=require('node:vm');
const source=fs.readFileSync('internal/bot/webassets/rift_potions.js','utf8');
function setup(fetchOverride){
 const nodes=new Map();
 function element(){return {value:'',hidden:false,disabled:false,textContent:'',children:[],addEventListener(){},replaceChildren(){this.children=[];this.value='';},append(option){this.children.push(option);if(!this.value)this.value=option.value;}};}
 for(const id of ['rift-potions','rift-potion-select','rift-potion-use','rift-potion-refresh','rift-potion-status'])nodes.set(id,element());
 let reads=0;
 const context={window:{addEventListener(){}},document:{getElementById:id=>nodes.get(id),createElement:element},location:{href:'http://localhost/abyss/rift'},URL,AbortController,setTimeout,clearTimeout,fetch:async()=>{reads++;if(fetchOverride)return fetchOverride();return {ok:true,json:async()=>({ok:true,potions:[{id:'small_health_potion',name:'Small potion',count:2,heal_hp:50}]})};}};
 vm.runInNewContext(source,context);return {api:context.window.RiftPotions,nodes,reads:()=>reads};
}
const run=()=>({status:'fighting',paused:false,player:{hp:30,max_hp:100},skill_timers:{}});
const settle=()=>new Promise(resolve=>setImmediate(resolve));
test('potion UI queues once, cancels on pause, and requires recovery after uncertainty',async()=>{
 const f=setup(),controller=f.api.create({api:'/api/abyss/rift',practice:''});
 f.nodes.get('rift-potion-refresh').onclick();await settle();controller.update(run(),true);
 const use=f.nodes.get('rift-potion-use');assert.equal(use.disabled,false);
 use.onclick();use.onclick();assert.equal(controller.take(),'small_health_potion');assert.equal(controller.take(),'');assert.equal(use.disabled,true);
 controller.finish(false);assert.match(f.nodes.get('rift-potion-status').textContent,/Recover/);use.onclick();assert.equal(controller.take(),'');
 controller.recover();await settle();controller.update(run(),true);use.onclick();controller.cancel();assert.equal(controller.take(),'');assert.equal(use.disabled,true);
 controller.update(run(),true);use.onclick();assert.equal(controller.take(),'small_health_potion');controller.finish(true);await settle();assert.equal(f.reads(),3);
});
test('potion UI blocks full health, cooldown, paused and ended state without sending',async()=>{
 const f=setup(),controller=f.api.create({api:'/api/abyss/rift',practice:''});f.nodes.get('rift-potion-refresh').onclick();await settle();
 for(const change of [r=>r.player.hp=100,r=>r.skill_timers.healing_potion=3,r=>r.paused=true,r=>r.status='cleared',r=>r.player.hp=0]){
  const r=run();change(r);controller.update(r,true);f.nodes.get('rift-potion-use').onclick();assert.equal(controller.take(),'');assert.equal(f.nodes.get('rift-potion-use').disabled,true);
 }
});
test('potion inventory validation rejects ambiguous or malformed healing metadata',()=>{
 const f=setup(),valid={id:'p',name:'Potion',count:2,heal_hp:50};
 assert.doesNotThrow(()=>f.api.validate({ok:true,potions:[valid]}));
 for(const p of [{...valid,count:0},{...valid,heal_fraction:.5},{...valid,heal_hp:NaN},{...valid,id:''}])assert.throws(()=>f.api.validate({ok:true,potions:[p]}));
 assert.throws(()=>f.api.validate({ok:true,potions:[valid,valid]}));
});

test('inventory failure stays visible across combat updates until a successful refresh',async()=>{
 let failed=true;
 const f=setup(async()=>({ok:!failed,json:async()=>({ok:true,potions:[{id:'small_health_potion',name:'Potion',count:1,heal_hp:50}]})}));
 const controller=f.api.create({api:'/api/abyss/rift',practice:''});
 f.nodes.get('rift-potion-refresh').onclick();await settle();
 for(let i=0;i<3;i++){controller.update(run(),true);assert.match(f.nodes.get('rift-potion-status').textContent,/unavailable.*Refresh/);}
 assert.equal(f.nodes.get('rift-potion-use').disabled,true);
 assert.equal(f.nodes.get('rift-potion-refresh').disabled,false);
 failed=false;f.nodes.get('rift-potion-refresh').onclick();await settle();controller.update(run(),true);
 assert.equal(f.nodes.get('rift-potion-use').disabled,false);
 assert.match(f.nodes.get('rift-potion-status').textContent,/Ready/);
});
