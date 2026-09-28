const {test}=require('node:test');
const assert=require('node:assert/strict');
const {createLedgeNavigator}=require('../../scripts/brawl-session-navigation.cjs');
const make=()=>({room:0,player:{x:297.337,y:425.019},level:{rooms:[{drop_edges:[{id:'ledge',x:260,y:365,w:80,landing_y:425}]}]}});
test('observed failed position routes around an end before ascending',()=>{
 const next=createLedgeNavigator(),run=make(),target={x:342.411,y:394.536};
 assert.deepEqual(next(run,target),{x:364,y:425.019});
 run.player.x=363;assert.deepEqual(next(run,target),{x:364,y:347});
 run.player.y=350;assert.equal(next(run,target),target);
});
test('right end, target retreat, room changes and absent target reset detours',()=>{
 const next=createLedgeNavigator(),run=make(),target={x:300,y:340};run.player.x=330;
 assert.deepEqual(next(run,target),{x:364,y:425.019});
 const lower={x:300,y:460};assert.equal(next(run,lower),lower);
 assert.deepEqual(next(run,target),{x:364,y:425.019});
 run.room=1;assert.equal(next(run,target),target);
 assert.equal(next(run,null),null);
});
test('ordinary pursuit and downward crossings do not invent a detour',()=>{
 const next=createLedgeNavigator(),run=make(),target={x:500,y:460};
 assert.equal(next(run,target),target);run.player.y=340;target.y=400;
 assert.equal(next(run,target),target);run.player.x=350;run.player.y=425;target.y=340;
 assert.equal(next(run,target),target);
});
