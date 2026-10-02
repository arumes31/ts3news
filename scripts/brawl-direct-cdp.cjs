'use strict';
// Direct CDP transport for isolated synthetic measurements. No domains are enabled
// implicitly: the caller decides whether Network inspection is part of a sample.
class CDPConnection{
 constructor(socket){
  this.socket=socket;this.nextID=0;this.pending=new Map();this.listeners=new Map();this.closed=false;
  socket.addEventListener('message',event=>{
   let message;try{message=JSON.parse(event.data);}catch{this.fail(new Error('Invalid CDP message'));return;}
   if(message.id){const pending=this.pending.get(message.id);if(!pending)return;this.pending.delete(message.id);clearTimeout(pending.timer);if(message.error)pending.reject(new Error(message.error.message||'CDP command failed'));else pending.resolve(message.result);}
   else if(message.method){for(const listener of this.listeners.get((message.sessionId||'')+'|'+message.method)||[])listener(message.params);}
  });
  socket.addEventListener('close',()=>this.fail(new Error('CDP connection closed')));
  socket.addEventListener('error',()=>this.fail(new Error('CDP connection failed')));
 }
 send(method,params={},sessionId,timeoutMS=30000){
  if(this.closed)return Promise.reject(new Error('CDP connection closed'));
  const id=++this.nextID;
  return new Promise((resolve,reject)=>{
   const timer=setTimeout(()=>{this.pending.delete(id);reject(new Error('CDP command timed out: '+method));},timeoutMS);
   this.pending.set(id,{resolve,reject,timer});
   try{this.socket.send(JSON.stringify({id,method,params,...(sessionId?{sessionId}:{})}));}catch(error){clearTimeout(timer);this.pending.delete(id);reject(error);}
  });
 }
 on(method,listener,sessionId){const key=(sessionId||'')+'|'+method;if(!this.listeners.has(key))this.listeners.set(key,new Set());this.listeners.get(key).add(listener);return()=>{this.listeners.get(key)?.delete(listener);};}
 fail(error){if(this.closed)return;this.closed=true;for(const item of this.pending.values()){clearTimeout(item.timer);item.reject(error);}this.pending.clear();this.listeners.clear();}
 close(){this.fail(new Error('CDP connection closed'));this.socket.close();}
}
async function connectCDP(url){
 const socket=new WebSocket(url);
 await new Promise((resolve,reject)=>{
  const cleanup=()=>{clearTimeout(timer);socket.removeEventListener('open',opened);socket.removeEventListener('error',failed);};
  const opened=()=>{cleanup();resolve();},failed=()=>{cleanup();reject(new Error('CDP connection failed'));};
  const timer=setTimeout(()=>{cleanup();socket.close();reject(new Error('CDP connection timed out'));},10000);
  socket.addEventListener('open',opened);socket.addEventListener('error',failed);
 });
 return new CDPConnection(socket);
}
module.exports={CDPConnection,connectCDP};
