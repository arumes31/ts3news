const {test}=require('node:test');
const assert=require('node:assert/strict');
const {createChest}=require('../../internal/bot/webassets/rift_chest.js');
const run=(status='fighting')=>({id:'one',status,room:0,paused:false,player:{x:160,y:410,elevation:0},level:{id:1,rooms:[{}]},drops:[{mission:1,tier:1,gold:15,collected:true,banked:false}]});
test('only a confirmed clear opens once and keeps rewards unchanged',async()=>{
 let loads=0;const c=createChest(async()=>{loads++;return {};});const r=run();const before=JSON.stringify(r);
 assert.equal(c.observe(r,false,0),false);await c.ready();assert.equal(loads,1);assert.equal(c.frame(0,false),null);
 assert.equal(JSON.stringify(r),before);r.status='cleared';assert.equal(c.observe(r,false,100),true);
 assert.equal(c.frame(100,false).index,0);assert.equal(c.frame(820,false).index,5);
 assert.equal(c.observe(r,false,900),false);assert.equal(loads,1);assert.equal(r.drops[0].gold,15);
});
test('replay and reduced motion use final frame; leaving retires chest',()=>{
 const c=createChest(async()=>({})),r=run('cleared');assert.equal(c.observe(r,true,100),false);assert.equal(c.frame(100,false).index,5);
 r.status='fighting';c.observe(r,false,200);assert.equal(c.frame(200,false),null);
 r.status='cleared';assert.equal(c.observe(r,false,300),false);
 const next=run();next.id='two';c.observe(next,false,400);next.status='cleared';assert.equal(c.observe(next,false,500),true);assert.equal(c.frame(500,true).index,5);
 next.status='complete';c.observe(next,false,600);assert.equal(c.frame(600,false),null);
});
test('optional art waits for active combat and failure retries only on a new room',async()=>{
 let loads=0;const c=createChest(async()=>{loads++;throw Error('offline');}),r=run();r.paused=true;c.observe(r,true,0);assert.equal(loads,0);
 r.paused=false;c.observe(r,false,10);await c.ready();c.observe(r,false,20);await c.ready();assert.equal(loads,1);assert.equal(c.image(),null);
 r.room=1;r.level.rooms.push({});c.observe(r,false,30);await c.ready();assert.equal(loads,2);
});
test('practice, ended runs and empty rewards never show a chest',()=>{
 for(const change of [{practice:{}},{status:'defeated'},{drops:[]}]){const c=createChest(async()=>({})),r={...run('cleared'),...change};assert.equal(c.observe(r,false,0),false);assert.equal(c.frame(0,false),null);}
});

test('chest keeps clear of exit portal and tall cover',()=>{
 const c=createChest(async()=>({})),r=run();r.player.x=1500;r.level.rooms[0]={exit:{x:1510,y:410},high_cover:[{x:1410,y:390,w:50,h:45}]};
 c.observe(r,false,0);r.status='cleared';c.observe(r,false,10);const frame=c.frame(10,false);
 assert.ok(Math.abs(frame.x-1510)>=80);assert.ok(frame.x+24<=1410||frame.x-24>=1460);
});

test('recovering a cleared snapshot during opening settles immediately without replaying',()=>{
 const c=createChest(async()=>({})),r=run();c.observe(r,false,0);r.status='cleared';assert.equal(c.observe(r,false,100),true);assert.equal(c.frame(100,false).index,0);
 assert.equal(c.observe(r,true,120),false);assert.equal(c.frame(120,false).index,5);
});
