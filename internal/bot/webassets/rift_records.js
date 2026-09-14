(function(){
 'use strict';
 const list=document.getElementById('rift-attempt-list'),comparison=document.getElementById('rift-attempt-comparison');
 const outcomes={completed:'Cleared',defeated:'Defeated',exited:'Left early',expired:'Expired / abandoned'};
 let lastKey='';
 const splitText=splits=>'Room splits: '+[0,1,2].map(index=>'Tier '+(index+1)+' '+(splits?.[index]==null?'unavailable':splits[index].toFixed(1)+'s')).join(' · ');
 const delta=value=>(value>0?'+':'')+value.toFixed(1);
 function update(run){
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
