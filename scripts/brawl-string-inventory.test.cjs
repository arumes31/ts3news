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

test('HTML extraction preserves template parameters, labels and entity decoding',()=>{
 const source='<div data-art="{{asset "/static/test.png"}}" title="Help &amp; tips">Hello {{.Name}}!<button aria-label="Jump">Go</button></div>\n<!-- Hidden comment --><script>"Not visible"</script><style>.x{}</style>';
 const result=inventory([['rift.html',source]]);
 const greeting=result.entries.find(entry=>entry.text==='Hello {1}!');
 assert.ok(greeting);assert.deepEqual(greeting.occurrences[0].parameters,['.Name']);
 assert.equal(greeting.occurrences[0].line,1);
 for(const text of ['Help & tips','Jump','Go'])assert.ok(result.entries.some(entry=>entry.text===text));
 assert.equal(result.entries.some(entry=>entry.text.includes('Not visible')||entry.text.includes('Hidden comment')||entry.text.includes('/static/test')),false);
});

test('Go entries retain source provenance in the combined inventory',()=>{
 const result=inventory([['a.js','const a="Ready";']],[{file:'server.go',sha256:'sourcehash',strings:[{file:'server.go',line:9,column:4,kind:'go_literal',text:'Ready',parameters:[],context:'GoStringLiteral'}]}]);
 assert.equal(result.entries.length,2);assert.deepEqual(result.files[1],{file:'server.go',sha256:'sourcehash'});
 const entry=result.entries.find(entry=>entry.kind==='go_literal');assert.equal(entry.occurrences[0].line,9);assert.equal(entry.translation,null);
});
