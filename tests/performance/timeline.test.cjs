const {test}=require('node:test');
const assert=require('node:assert/strict'),fs=require('node:fs'),os=require('node:os'),path=require('node:path'),{EventEmitter}=require('node:events');
const {startTimeline}=require('../../scripts/brawl-timeline.cjs');
class Fake extends EventEmitter{
 constructor(chunks){super();this.chunks=chunks;this.calls=[];}
 async send(method,args){this.calls.push({method,args});if(method==='Tracing.end')this.emit('Tracing.tracingComplete',{stream:'fixture',dataLossOccurred:false});if(method==='IO.read'){const c=this.chunks.shift();if(c instanceof Error)throw c;return c;}return {};}
}
function fixture(t){const dir=fs.mkdtempSync(path.join(os.tmpdir(),'brawl-timeline-'));const output=path.join(dir,'trace.json');t.after(()=>{if(fs.existsSync(output))fs.unlinkSync(output);fs.rmdirSync(dir);});return output;}
test('streams mixed encodings, closes handle and stops once',async t=>{
 const output=fixture(t),cdp=new Fake([{data:'{"traceEvents":',eof:false},{data:Buffer.from('[]}').toString('base64'),base64Encoded:true,eof:true}]);
 const capture=await startTimeline(cdp,output);const result=await capture.stop();await capture.stop();
 assert.equal(fs.readFileSync(output,'utf8'),'{"traceEvents":[]}');assert.equal(result.dataLossOccurred,false);assert.equal(result.bytes,18);
 assert.equal(cdp.calls.filter(c=>c.method==='Tracing.end').length,1);assert.equal(cdp.calls.filter(c=>c.method==='IO.close').length,1);assert.equal(cdp.listenerCount('Tracing.tracingComplete'),0);
});
test('read failure closes the stream and removes listener',async t=>{
 const cdp=new Fake([Error('read failed')]),capture=await startTimeline(cdp,fixture(t));
 await assert.rejects(capture.stop(),/read failed/);assert.equal(cdp.calls.at(-1).method,'IO.close');assert.equal(cdp.listenerCount('Tracing.tracingComplete'),0);
});
test('size limit closes the stream instead of accepting incomplete evidence',async t=>{
 const cdp=new Fake([{data:'oversized',eof:true}]),capture=await startTimeline(cdp,fixture(t),{maxBytes:2});
 await assert.rejects(capture.stop(),/size limit/);assert.equal(cdp.calls.at(-1).method,'IO.close');
});

test('completion timeout removes the listener',async t=>{
 const cdp=new Fake([]);cdp.send=async()=>({});
 const capture=await startTimeline(cdp,fixture(t),{timeoutMS:10});
 await assert.rejects(capture.stop(),/timed out/);assert.equal(cdp.listenerCount('Tracing.tracingComplete'),0);
});
test('missing stream is rejected without creating output',async t=>{
 const output=fixture(t),cdp=new Fake([]);cdp.send=async method=>{if(method==='Tracing.end')cdp.emit('Tracing.tracingComplete',{dataLossOccurred:true});return {};};
 const capture=await startTimeline(cdp,output);await assert.rejects(capture.stop(),/stream missing/);assert.equal(fs.existsSync(output),false);assert.equal(cdp.listenerCount('Tracing.tracingComplete'),0);
});
