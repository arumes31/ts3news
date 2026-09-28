const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs'),path=require('node:path'),crypto=require('node:crypto');
const root=path.resolve(__dirname,'../internal/bot/webassets');
const assets=fs.readdirSync(root).filter(name=>/^rift_.*\.png$/.test(name)).sort();
test('Brawl generated-art inventory is nonempty',()=>assert.ok(assets.length>=20));
for(const asset of assets)test(asset+' has its preserved prompt and matching output metadata',()=>{
 const file=path.join(root,asset),bytes=fs.readFileSync(file),record=JSON.parse(fs.readFileSync(file.replace(/\.png$/,'.prompt.json'),'utf8'));
 assert.equal(record.asset,asset);
 assert.equal(typeof record.prompt,'string');assert.ok(record.prompt.trim().length>100,'missing generation prompt');
 assert.equal(record.sha256,crypto.createHash('sha256').update(bytes).digest('hex'),'prompt record belongs to a different output');
 assert.equal(record.bytes,bytes.length);
 assert.deepEqual([...bytes.subarray(0,8)],[137,80,78,71,13,10,26,10]);
 assert.deepEqual(record.actualSize,[bytes.readUInt32BE(16),bytes.readUInt32BE(20)]);
});
