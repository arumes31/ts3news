const {test}=require('node:test');
const assert=require('node:assert/strict');
const {sessionOptions}=require('../../scripts/brawl-session-options.cjs');
test('normal sessions preserve the three thirty-minute samples',()=>{
 const o=sessionOptions({});assert.equal(o.samples,3);assert.equal(o.durationMS,1800000);assert.equal(o.minimumReplays,0);assert.deepEqual(o.replayCheckpoints,[]);
});
test('post-cap capture requires sixty complete replays and thirty minutes',()=>{
 const o=sessionOptions({BRAWL_SESSION_POST_CAP:'1'});assert.equal(o.samples,1);assert.equal(o.minimumReplays,60);assert.equal(o.durationMS,1800000);assert.deepEqual(o.replayCheckpoints,[49,54,59,60]);assert.equal(o.timeoutMS,3600000);
});
test('smoke cannot masquerade as a post-cap capture',()=>{
 const o=sessionOptions({BRAWL_SESSION_SMOKE:'1'});assert.equal(o.samples,1);assert.equal(o.durationMS,60000);assert.equal(o.minimumReplays,0);
 assert.throws(()=>sessionOptions({BRAWL_SESSION_SMOKE:'1',BRAWL_SESSION_POST_CAP:'1'}),/mutually exclusive/);
});
