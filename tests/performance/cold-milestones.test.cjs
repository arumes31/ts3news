const test=require('node:test'),assert=require('node:assert/strict');
const {summarizeResources}=require('../../scripts/brawl-cold-milestones.cjs');
const entries=[
 {entryType:'navigation',name:'http://127.0.0.1/abyss/rift',responseEnd:10,transferSize:100,encodedBodySize:80,duration:10},
 {entryType:'resource',name:'http://127.0.0.1/static/base.png?v=secret',responseEnd:90,transferSize:1000,encodedBodySize:900,duration:80},
 {entryType:'resource',name:'http://127.0.0.1/static/creature.png?v=hash',responseEnd:150,transferSize:400,encodedBodySize:300,duration:40},
 {entryType:'resource',name:'http://127.0.0.1/api/abyss/rift',responseEnd:180,transferSize:200,encodedBodySize:100,duration:30},
 {entryType:'resource',name:'http://127.0.0.1/late',responseEnd:220,transferSize:500,encodedBodySize:400,duration:10}
];
test('first fight totals include artwork and API traffic after idle readiness',()=>{
 const idle=summarizeResources(entries,100),fight=summarizeResources(entries,180);
 assert.equal(idle.transferBytes,1100);assert.equal(idle.resourceCount,1);
 assert.equal(fight.transferBytes,1700);assert.equal(fight.encodedBodyBytes,1380);assert.equal(fight.resourceCount,3);
 assert.deepEqual(fight.largest.map(item=>item.path),['/static/base.png','/static/creature.png','/api/abyss/rift']);
 assert.equal(JSON.stringify(fight).includes('secret'),false);
});
test('milestone excludes unfinished or later resources and keeps zero-byte cache entries',()=>{
 const sample=summarizeResources([...entries,{entryType:'resource',name:'http://127.0.0.1/cached',responseEnd:100,transferSize:0,encodedBodySize:20,duration:1}],100);
 assert.equal(sample.resourceCount,2);assert.equal(sample.transferBytes,1100);assert.equal(sample.encodedBodyBytes,1000);
});
