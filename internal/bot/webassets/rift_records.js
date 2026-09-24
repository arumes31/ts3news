(function(){
 'use strict';
 const list=document.getElementById('rift-attempt-list'),comparison=document.getElementById('rift-attempt-comparison');
 const outcomes={completed:'Cleared',defeated:'Defeated',exited:'Left early',expired:'Expired / abandoned'};
 let lastKey='',latestRun=null;
 const exportButton=document.getElementById('rift-export-records'),exportStatus=document.getElementById('rift-export-status');
 exportButton.onclick=()=>{
  if(!latestRun)return;
  try{
   const run=latestRun,past=run.past_expeditions||{},stats=run.stats||{},at=new Date();
   const records={version:1,exported_at:at.toISOString(),completed_missions:run.completed_levels||[],mission_history:run.mission_history||{},region_records:run.region_records||{},monster_records:run.monster_records||{},recent_attempts:run.attempt_history||[],career:{enemies:(past.enemies||0)+(stats.kills||0),bosses:(past.bosses||0)+(stats.bosses||0),treasure_goblins:(past.treasure_goblins||0)+(stats.treasure_goblins||0),gold:(past.gold||0)+(run.banked_gold||0),gear:(past.gear||0)+(run.banked_items?.length||0)},current_expedition:{mission:run.level?.id,status:run.status,stats,room_splits:run.room_splits||[null,null,null],clear_streak:run.clear_streak||0,best_clear_streak:run.best_clear_streak||0},notes:'Recent history retains up to 50 attempts. Older records can be incomplete. Missing tier times remain null.'};
   const url=URL.createObjectURL(new Blob([JSON.stringify(records,null,2)],{type:'application/json'})),link=document.createElement('a');link.href=url;link.download='rift-brawl-records-'+at.toISOString().slice(0,10)+'.json';document.body.append(link);
   try{link.click();exportStatus.textContent='Record download prepared.';}finally{link.remove();setTimeout(()=>URL.revokeObjectURL(url),1000);}
  }catch(_){exportStatus.textContent='Could not prepare the export. Try again.';}
 };
 const splitText=splits=>'Room splits: '+[0,1,2].map(index=>'Tier '+(index+1)+' '+(splits?.[index]==null?'unavailable':splits[index].toFixed(1)+'s')).join(' · ');
 const delta=value=>(value>0?'+':'')+value.toFixed(1);
 let masteryKey='';
 let classNames={};
 const classLabel=name=>Object.prototype.hasOwnProperty.call(classNames,name)?classNames[name]:name.charAt(0).toUpperCase()+name.slice(1);
 function classIdentity(build){
  const subclass=build?.class,base=build?.base_class;
  const name=subclass?(Object.prototype.hasOwnProperty.call(classNames,subclass)?classLabel(subclass):build.class_name||classLabel(subclass)):'Adventurer';
  return base&&base!==subclass?classLabel(base)+' · '+name:name;
 }
 function init(names){classNames=Object.fromEntries(Object.entries(names||{}).filter(([id,name])=>id&&typeof name==='string'&&name));masteryKey='';lastKey='';}
 function mastery(run){
  const node=document.getElementById('rift-class-mastery');node.hidden=!!run.practice;if(run.practice)return;
  const totals=new Map(),current=run.build?.class;
  if(current)totals.set(current,{missions:0,clears:0});
  for(const [id,record] of Object.entries(run.mission_history||{})){
   if(!Number.isInteger(Number(id))||Number(id)<1||Number(id)>100)continue;
   for(const [name,count] of Object.entries(record.completed_by_class||{})){
    if(!name||!Number.isSafeInteger(count)||count<=0)continue;
    const value=totals.get(name)||{missions:0,clears:0};value.missions++;value.clears+=count;totals.set(name,value);
   }
  }
  const rows=[...totals].sort(([a],[b])=>a===current?-1:b===current?1:a.localeCompare(b)),key=JSON.stringify([current,rows]);
  if(key!==masteryKey){
   masteryKey=key;const list=document.getElementById('rift-class-mastery-list');list.replaceChildren();
   for(const [name,counts] of rows){const item=document.createElement('li');item.dataset.class=name;item.textContent=classLabel(name)+(name===current?' (current)':'')+' · '+counts.missions+' distinct missions · '+counts.clears+' recorded clears · '+(counts.missions>=10?'Mastery badge earned':(10-counts.missions)+' more distinct missions to mastery');list.append(item);}
   if(!rows.length){const item=document.createElement('li');item.textContent='No recorded subclass results yet.';list.append(item);}
  }
  const stats=run.stats||{},charged=stats.charged_finishers,empty=stats.empty_finishers,spent=stats.charges_spent,valid=[charged,empty,spent].every(n=>Number.isSafeInteger(n)&&n>=0);
  let description='Current expedition'+(current?' · '+classLabel(current):'')+': ';
  if(!valid)description+='finisher statistics unavailable for this saved expedition.';
  else{const casts=charged+empty;description+=charged+' charged finishers · '+empty+' empty finishers · '+spent+' charges spent · '+(casts?Math.round(charged/casts*100)+'% of finisher casts charged':'No finisher casts yet')+' · '+(charged?(spent/charged).toFixed(1)+' average charges per charged finisher':'Average charge spend unavailable')+'. Casts include misses.';}
  const summary=document.getElementById('rift-class-mastery-current');if(summary.textContent!==description)summary.textContent=description;
 }
 let regionKey='';
 function regionRecords(run){
  const section=document.getElementById('rift-region-records');section.hidden=!!run.practice;if(run.practice)return;
  const records=run.region_records||{},key=JSON.stringify(records);if(key===regionKey)return;regionKey=key;
  const list=document.getElementById('rift-region-record-list');list.replaceChildren();
  for(let region=0;region<10;region++){
   const record=records[region],item=document.createElement('li');item.textContent='Region '+(region+1)+' · Missions '+(region*10+1)+'–'+(region*10+10)+' · '+(record?record.best_seconds.toFixed(1)+'s'+(record.at_ms?' · '+new Date(record.at_ms).toLocaleString():''):'No complete regional run recorded');list.append(item);
  }
 }
 function update(run){
  latestRun=run;exportButton.disabled=false;mastery(run);regionRecords(run);
  const current='Current mission '+(run.level?.id||'')+' · '+splitText(run.room_splits);const splitNode=document.getElementById('rift-room-splits');if(splitNode.textContent!==current)splitNode.textContent=current;
  const attempts=run.attempt_history||[],key=JSON.stringify(attempts);if(key===lastKey)return;lastKey=key;list.replaceChildren();
  for(const record of [...attempts].reverse()){
   const item=document.createElement('li'),heading=document.createElement('strong'),body=document.createElement('p');
   heading.textContent='Mission '+record.mission+' · '+outcomes[record.outcome]+' · '+(record.at_ms>0?new Date(record.at_ms).toLocaleString():'Date unavailable');
   body.textContent=(record.class?classLabel(record.class):'Subclass unavailable')+' · '+record.seconds.toFixed(1)+'s combat · End HP '+record.hp.toFixed(1)+'/'+record.max_hp.toFixed(1)+' · Damaging hits '+(record.hits??'unavailable');
   const splits=document.createElement('p');splits.textContent=splitText(record.splits);item.append(heading,body,splits);list.append(item);
  }
  const latest=attempts.at(-1),previous=latest?[...attempts.slice(0,-1)].reverse().find(record=>record.mission===latest.mission):null;
  if(!latest){comparison.textContent='No recorded attempts yet.';return;}
  if(!previous){comparison.textContent='No previous recorded attempt for Mission '+latest.mission+'.';return;}
  comparison.textContent='Mission '+latest.mission+' compared with previous attempt ('+outcomes[previous.outcome]+' → '+outcomes[latest.outcome]+'): combat time '+delta(latest.seconds-previous.seconds)+'s; end HP '+delta(latest.hp-previous.hp)+'; damaging hits '+(latest.hits==null||previous.hits==null?'comparison unavailable':delta(latest.hits-previous.hits))+'.';
 }
 window.RiftRecords={update,init,classLabel,classIdentity};
})();
