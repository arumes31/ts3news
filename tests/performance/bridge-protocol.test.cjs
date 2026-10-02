const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs'),vm=require('node:vm');
const context={window:{}};
vm.runInNewContext(fs.readFileSync('internal/bot/webassets/rift_protocol.js','utf8'),context);
const deck=()=>({id:'deck',x:400,y:370,w:200,h:90});
function validate(bridges){
 return context.window.RiftProtocol.validate({ok:true,run:null,build:{name:'Fighter',class:'warrior',skills:[],gear:[]},rooms:[],bestiary:[],levels:[{id:1,name:'Test',region_name:'Ruins',tactic:'Test',difficulty:'Wayfarer',region:0,rooms:Array.from({length:3},()=>({name:'Room',obstacles:[],hazards:[],bridges}))}]},'GET');
}
test('bridge protocol accepts legacy and separated decks',()=>{
 for(const bridges of [undefined,[],[deck()],[deck(),{...deck(),id:'other',x:900}]])assert.doesNotThrow(()=>validate(bridges));
});
test('bridge protocol rejects malformed and overlapping walkable bounds',()=>{
 for(const changes of [{id:''},{id:'x'.repeat(81)},{x:NaN},{x:Infinity},{x:99},{w:79},{h:79},{y:450}])assert.throws(()=>validate([{...deck(),...changes}]),/incomplete expedition/);
 for(const bridges of [[deck(),deck()],[deck(),{...deck(),id:'other',x:650}],Array.from({length:5},(_,i)=>({...deck(),id:String(i)}))])assert.throws(()=>validate(bridges),/incomplete expedition/);
});
