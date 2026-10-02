const {test}=require('node:test');const assert=require('node:assert/strict');
const {CDPConnection}=require('../../scripts/brawl-direct-cdp.cjs');
class Socket extends EventTarget{sent=[];send(text){this.sent.push(JSON.parse(text));}receive(value){this.dispatchEvent(new MessageEvent('message',{data:JSON.stringify(value)}));}close(){this.dispatchEvent(new Event('close'));}}
test('routes out-of-order replies and target notifications without enabling domains',async()=>{
 const ws=new Socket(),cdp=new CDPConnection(ws);assert.equal(ws.sent.length,0);
 const one=cdp.send('Browser.getVersion'),two=cdp.send('Runtime.evaluate',{expression:'1'},'target');assert.equal(ws.sent[1].sessionId,'target');
 const seen=[];cdp.on('Runtime.exceptionThrown',event=>seen.push(event),'target');ws.receive({method:'Runtime.exceptionThrown',sessionId:'other',params:{value:0}});ws.receive({method:'Runtime.exceptionThrown',sessionId:'target',params:{value:1}});
 ws.receive({id:2,result:{value:2}});ws.receive({id:1,result:{value:1}});assert.deepEqual(await one,{value:1});assert.deepEqual(await two,{value:2});assert.deepEqual(seen,[{value:1}]);cdp.close();
});
test('protocol errors and closed connections reject pending commands',async()=>{
 const ws=new Socket(),cdp=new CDPConnection(ws);const failed=cdp.send('Invalid');ws.receive({id:1,error:{message:'method unavailable'}});await assert.rejects(failed,/method unavailable/);
 const pending=cdp.send('Runtime.evaluate');ws.close();await assert.rejects(pending,/closed/);await assert.rejects(cdp.send('Browser.getVersion'),/closed/);
});
test('timeouts remove pending commands and late replies are ignored',async()=>{
 const ws=new Socket(),cdp=new CDPConnection(ws);await assert.rejects(cdp.send('Slow',{},undefined,10),/timed out/);ws.receive({id:1,result:{}});cdp.close();
});
