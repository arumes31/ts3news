(function(){
 'use strict';
 const root=document.getElementById('rift-boss-intro'),title=document.getElementById('rift-boss-intro-name');
 const skipped=new Set();
 try{const saved=JSON.parse(localStorage.getItem('riftSkippedBossIntros'));if(Array.isArray(saved))for(const key of saved.slice(-1000))if(typeof key==='string'&&key.length<=512)skipped.add(key);}catch(_){}
 let current='',bosses=[],timer=0,last=null;
 function hide(){clearTimeout(timer);root.hidden=true;}
 function update(run){
  last=run;
  if(!run||run.practice||run.status!=='fighting'){hide();current='';return;}
  const alive=run.enemies.filter(actor=>actor.kind==='boss'&&actor.hp>0);
  if(!alive.length){hide();return;}
  const key=JSON.stringify([run.id,run.level?.id,run.room,alive.map(actor=>actor.id)]);
  if(key===current)return;
  current=key;hide();bosses=alive.filter(actor=>!skipped.has(actor.art_key||actor.name));
  if(!bosses.length)return;
  title.textContent=bosses.map(actor=>actor.name).join(' · ');root.hidden=false;
  timer=setTimeout(()=>{if(!root.contains(document.activeElement))hide();},8000);
 }
 document.getElementById('rift-boss-intro-skip').addEventListener('click',()=>{
  for(const boss of bosses)skipped.add(boss.art_key||boss.name);
  while(skipped.size>1000)skipped.delete(skipped.values().next().value);
  try{localStorage.setItem('riftSkippedBossIntros',JSON.stringify([...skipped]));}catch(_){}
  hide();document.getElementById('rift-canvas').focus({preventScroll:true});
 });
 document.getElementById('rift-boss-intro-reset').addEventListener('click',()=>{
  skipped.clear();try{localStorage.setItem('riftSkippedBossIntros','[]');}catch(_){}
  current='';update(last);document.getElementById('rift-boss-intro-setting-status').textContent='Boss introductions will be shown again.';
 });
 window.RiftBossIntro={update};
})();
