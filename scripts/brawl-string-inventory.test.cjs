const {test}=require('node:test');
const assert=require('node:assert/strict');
const {extractStrings,inventory}=require('./brawl-string-inventory.cjs');
test('extracts source text without executing code and keeps placeholders',()=>{
 const source='throw new Error("Do not execute"); const label = `Hello ${player.name}!`; // "not copy"\n const again="Do not execute"; const re=/"not a string"/;';
 const found=extractStrings('example.js',source);
 assert.equal(found.filter(entry=>entry.text==='Do not execute').length,2);
 assert.equal(found.find(entry=>entry.kind==='template').text,'Hello {1}!');
 assert.deepEqual(found.find(entry=>entry.kind==='template').parameters,['player.name']);
 assert.equal(found.some(entry=>entry.text==='not copy'||entry.text==='not a string'),false);
 assert.equal(found.at(-1).line,2);
});
test('inventory deduplicates copy while retaining locations and stable identities',()=>{
 const a=inventory([['a.js','const a="Ready"; const b="Ready";'],['b.js','const c="Ready";']]);
 const b=inventory([['b.js','const c="Ready";'],['a.js','const a="Ready"; const b="Ready";']]);
 assert.deepEqual(a,b);assert.equal(a.entries.length,1);assert.equal(a.entries[0].occurrences.length,3);
 assert.equal(a.entries[0].translation,null);assert.equal(a.entries[0].review,'unreviewed');
 const shifted=inventory([['a.js','\nconst a="Ready";']]);assert.equal(shifted.entries[0].id,a.entries[0].id);
});
test('parse failures are explicit rather than silently losing strings',()=>{
 assert.throws(()=>extractStrings('broken.js','const ='),/broken.js/);
});
