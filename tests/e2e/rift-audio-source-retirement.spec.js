const {test,expect}=require('@playwright/test');
const path=require('node:path');
test('combat exit stops and disconnects every sound source across repeated resumes',async({page})=>{
 await page.setContent('<button id="activate">Activate</button>');
 await page.evaluate(()=>{
  window.sourceRecords=[];
  for(const method of ['createOscillator','createBufferSource']){
   const create=AudioContext.prototype[method];
   AudioContext.prototype[method]=function(...args){
    const node=create.apply(this,args),record={kind:method,started:false,stopped:false,disconnected:false};
    window.sourceRecords.push(record);
    for(const [method,flag] of [['start','started'],['stop','stopped'],['disconnect','disconnected']]){
     const original=node[method];node[method]=function(...args){record[flag]=true;return original.apply(this,args);};
    }
    return node;
   };
  }
 });
 await page.addScriptTag({path:path.resolve(__dirname,'../../internal/bot/webassets/rift_audio.js')});
 await page.click('#activate');
 const cycles=await page.evaluate(async()=>{
  const audio=window.RiftAudio,cycles=[];
  for(let cycle=0;cycle<6;cycle++){
   const start=window.sourceRecords.length;
   await audio.setActive(true,cycle);
   audio.play('arrival',0);audio.play('hit',0);audio.startBossMusic(0);
   audio.area((cycle+1)%10,.8);audio.fadeBossMusic(.8);
   const before={voices:audio.voices,nodes:audio.getTrackedAmbienceCount()};
   if(cycle%3===0)await audio.setActive(false);
   else if(cycle%3===1)audio.silence();
   else window.dispatchEvent(new PageTransitionEvent('pagehide',{persisted:true}));
   const records=window.sourceRecords.slice(start);
   cycles.push({before,count:records.length,oscillators:records.filter(r=>r.kind==='createOscillator').length,buffers:records.filter(r=>r.kind==='createBufferSource').length,unretired:records.filter(r=>r.started&&(!r.stopped||!r.disconnected)),voices:audio.voices,nodes:audio.getTrackedAmbienceCount(),boss:audio.bossMusicActive,active:audio.active});
   if(cycle%3===2)window.dispatchEvent(new PageTransitionEvent('pageshow',{persisted:true}));
  }
  return cycles;
 });
 for(const cycle of cycles){
  expect(cycle.before.voices).toBeGreaterThan(0);expect(cycle.before.nodes).toBeGreaterThan(0);
  expect(cycle.oscillators).toBeGreaterThan(0);expect(cycle.buffers).toBeGreaterThan(0);
  expect(cycle.unretired).toEqual([]);expect(cycle.voices).toBe(0);expect(cycle.nodes).toBe(0);
  expect(cycle.boss).toBe(false);expect(cycle.active).toBe(false);
 }
});
