(function(){
 'use strict';
 const panel=document.getElementById('rift-class-challenges'),toggle=document.getElementById('rift-class-challenges-enabled'),body=document.getElementById('rift-class-challenges-body'),summary=document.getElementById('rift-class-challenges-summary'),list=document.getElementById('rift-class-challenges-list');
 let latest=null,lastKey='';
 try{toggle.checked=localStorage.getItem('riftClassChallenges')==='true';}catch(_){}
 function update(run){
  latest=run;panel.hidden=!!run?.practice||!!document.getElementById('rift-app').dataset.practice;body.hidden=!toggle.checked;if(!toggle.checked||panel.hidden)return;
  const build=run?.build,builder=build?.signatures?.find(s=>s.role==='builder'),finisher=build?.signatures?.find(s=>s.role==='finisher');
  if(!run||!builder||!finisher){summary.textContent=!run?'Start an expedition to track class challenges.':'Class challenges need your equipped builder and finisher. Unlock them in Abyss My Build.';list.replaceChildren();lastKey='';return;}
  const stats=run.stats||{},goals=[['First charged finisher','Cast '+finisher.name+' with at least one charge.',stats.charged_finishers,1],['Resource practice','Spend nine '+(build.resource||'class')+' charges across finisher casts.',stats.charges_spent,9],['Finisher practice','Cast ten charged finishers during this expedition.',stats.charged_finishers,10],['Three tiers','Clear three tiers during this expedition.',stats.rooms_cleared,3]];
  const valid=value=>Number.isSafeInteger(value)&&value>=0,completed=goals.filter(([, ,value,target])=>valid(value)&&value>=target).length;
  const key=JSON.stringify([run.id,window.RiftRecords.classIdentity(build),goals]);if(key===lastKey)return;lastKey=key;
  summary.textContent=window.RiftRecords.classIdentity(build)+' · '+completed+'/4 class challenges complete'+(completed===4?' — Track complete!':'.');list.replaceChildren();
  for(const [name,description,value,target] of goals){const item=document.createElement('li'),title=document.createElement('strong'),text=document.createElement('p');const done=valid(value)&&value>=target;item.dataset.complete=String(done);title.textContent=(done?'✓ Complete · ':'')+name;const progress=valid(value)?Math.min(value,target)+'/'+target:'Progress unavailable in this save';text.textContent=description+' '+progress;item.append(title,text);if(valid(value)){const meter=document.createElement('progress');meter.max=target;meter.value=Math.min(value,target);meter.setAttribute('aria-label',name+' progress');item.append(meter);}list.append(item);}
 }
 toggle.addEventListener('change',()=>{try{localStorage.setItem('riftClassChallenges',String(toggle.checked));}catch(_){}lastKey='';update(latest);});
 window.RiftClassChallenges={update};
})();
