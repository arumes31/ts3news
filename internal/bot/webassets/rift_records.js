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
   const records={version:1,exported_at:at.toISOString(),completed_missions:run.completed_levels||[],mission_history:run.mission_history||{},monster_records:run.monster_records||{},recent_attempts:run.attempt_history||[],career:{enemies:(past.enemies||0)+(stats.kills||0),bosses:(past.bosses||0)+(stats.bosses||0),treasure_goblins:(past.treasure_goblins||0)+(stats.treasure_goblins||0),gold:(past.gold||0)+(run.banked_gold||0),gear:(past.gear||0)+(run.banked_items?.length||0)},current_expedition:{mission:run.level?.id,status:run.status,stats,room_splits:run.room_splits||[null,null,null],clear_streak:run.clear_streak||0,best_clear_streak:run.best_clear_streak||0},notes:'Recent history retains up to 50 attempts. Older records can be incomplete. Missing tier times remain null.'};
   const url=URL.createObjectURL(new Blob([JSON.stringify(records,null,2)],{type:'application/json'})),link=document.createElement('a');link.href=url;link.download='rift-brawl-records-'+at.toISOString().slice(0,10)+'.json';document.body.append(link);
   try{link.click();exportStatus.textContent='Record download prepared.';}finally{link.remove();setTimeout(()=>URL.revokeObjectURL(url),1000);}
  }catch(_){exportStatus.textContent='Could not prepare the export. Try again.';}
 };
 const splitText=splits=>'Room splits: '+[0,1,2].map(index=>'Tier '+(index+1)+' '+(splits?.[index]==null?'unavailable':splits[index].toFixed(1)+'s')).join(' · ');
 const delta=value=>(value>0?'+':'')+value.toFixed(1);
 function update(run){
  latestRun=run;exportButton.disabled=false;
  const current='Current mission '+(run.level?.id||'')+' · '+splitText(run.room_splits);const splitNode=document.getElementById('rift-room-splits');if(splitNode.textContent!==current)splitNode.textContent=current;
  const attempts=run.attempt_history||[],key=JSON.stringify(attempts);if(key===lastKey)return;lastKey=key;list.replaceChildren();
  for(const record of [...attempts].reverse()){
   const item=document.createElement('li'),heading=document.createElement('strong'),body=document.createElement('p');
   heading.textContent='Mission '+record.mission+' · '+outcomes[record.outcome]+' · '+(record.at_ms>0?new Date(record.at_ms).toLocaleString():'Date unavailable');
   body.textContent=(record.class||'Subclass unavailable')+' · '+record.seconds.toFixed(1)+'s combat · End HP '+record.hp.toFixed(1)+'/'+record.max_hp.toFixed(1)+' · Damaging hits '+(record.hits??'unavailable');
   const splits=document.createElement('p');splits.textContent=splitText(record.splits);item.append(heading,body,splits);list.append(item);
  }
  const latest=attempts.at(-1),previous=latest?[...attempts.slice(0,-1)].reverse().find(record=>record.mission===latest.mission):null;
  if(!latest){comparison.textContent='No recorded attempts yet.';return;}
  if(!previous){comparison.textContent='No previous recorded attempt for Mission '+latest.mission+'.';return;}
  comparison.textContent='Mission '+latest.mission+' compared with previous attempt ('+outcomes[previous.outcome]+' → '+outcomes[latest.outcome]+'): combat time '+delta(latest.seconds-previous.seconds)+'s; end HP '+delta(latest.hp-previous.hp)+'; damaging hits '+(latest.hits==null||previous.hits==null?'comparison unavailable':delta(latest.hits-previous.hits))+'.';
 }
 window.RiftRecords={update};
})();
