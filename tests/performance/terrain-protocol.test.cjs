const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');
const context={window:{}};
vm.runInNewContext(fs.readFileSync('internal/bot/webassets/rift_protocol.js','utf8'),context);
const validate=cover=>context.window.RiftProtocol.validate({ok:true,run:null,build:{name:'Fighter',class:'warrior',skills:[],gear:[]},rooms:[],bestiary:[],levels:[{id:1,name:'Test',region_name:'Ruins',tactic:'Test',difficulty:'Wayfarer',region:0,rooms:Array.from({length:3},()=>({name:'Room',obstacles:[],hazards:[],cover:[cover]}))}]},'GET');
const wood=()=>({id:'wood',x:480,y:395,w:20,h:30,material:'wood',hp:60,max_hp:60});
test('cover protocol accepts legacy, intact volatile, warning and spent states',()=>{
 for(const fields of [{},{volatile:true},{volatile:true,hp:0,blast_fuse:1.2},{volatile:true,hp:0,blast_fuse:.6},{volatile:true,hp:0}])assert.doesNotThrow(()=>validate({...wood(),...fields}));
});
test('cover protocol rejects malformed or impossible reaction state',()=>{
 for(const fields of [{volatile:'yes'},{volatile:true,blast_fuse:.5},{blast_fuse:.5,hp:0},{volatile:false,blast_fuse:.5,hp:0},{volatile:true,blast_fuse:1.21,hp:0},{volatile:true,blast_fuse:-1,hp:0},{volatile:true,blast_fuse:NaN,hp:0},{volatile:true,blast_fuse:Infinity,hp:0},{volatile:true,material:'stone',hp:0,max_hp:0}])assert.throws(()=>validate({...wood(),...fields}),/incomplete expedition/);
});
