'use strict';
function summarizeResources(entries,cutoff){
 const completed=entries.filter(entry=>entry.responseEnd>0&&entry.responseEnd<=cutoff);
 const resources=completed.filter(entry=>entry.entryType==='resource');
 return {transferBytes:completed.reduce((sum,entry)=>sum+entry.transferSize,0),encodedBodyBytes:completed.reduce((sum,entry)=>sum+entry.encodedBodySize,0),resourceCount:resources.length,largest:resources.map(entry=>({path:new URL(entry.name).pathname,bytes:entry.encodedBodySize,durationMS:entry.duration})).sort((a,b)=>b.bytes-a.bytes).slice(0,10)};
}
module.exports={summarizeResources};
