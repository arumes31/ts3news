(function(){
 'use strict';
 const root=document.getElementById('rift-optional-objectives'),list=document.getElementById('rift-objectives-list'),caption=document.getElementById('rift-objectives-caption'),last=document.getElementById('rift-objectives-last'),lastList=document.getElementById('rift-objectives-last-list');
 let options=[],key='',lastRun=null,lastPaused=false;
 const toggle=document.getElementById('rift-show-objectives');
 try{toggle.checked=localStorage.getItem('riftShowObjectives')!=='false';}catch(_){}
 toggle.addEventListener('change',()=>{try{localStorage.setItem('riftShowObjectives',String(toggle.checked));}catch(_){}key='';update(lastRun,lastPaused);});
 function text(tag,value,parent){const el=document.createElement(tag);el.textContent=value;parent.append(el);return el;}
 function rows(parent,entries,preview){
  parent.replaceChildren();
  for(const entry of entries){
   const row=document.createElement('li');row.dataset.objectiveId=entry.id;row.dataset.state=preview?'preview':entry.status;
   text('strong',entry.name,row);text('span',preview?'Optional':({active:'In progress',failed:'Failed',complete:'Complete'}[entry.status]),row);
   text('p',entry.description,row);
   if(!preview){const value=entry.id==='no_potions'?entry.current+' potion'+(entry.current===1?'':'s')+' used':entry.id==='timed'?Math.floor(entry.current)+' / '+entry.target+' combat seconds':(entry.id==='ranged_priority'||entry.id==='elite_priority')?entry.current+' priority violations':entry.id==='limited_dodge'?entry.current+' / '+entry.target+' allowed dodges':entry.id==='melee_only'?entry.current+' non-melee casts':entry.id==='aerial_finish'?entry.current+' / '+entry.target+' aerial finishes':entry.id==='treasure_capture'?entry.current+' / '+entry.target+' goblins captured':entry.id==='hazard_avoidance'?entry.current+' hazard contacts':entry.id==='save_ultimate'?entry.current+' ultimates cast':entry.id==='finisher'?entry.current+' / '+entry.target+' charged finishers':entry.id==='guard'?entry.current+' / '+entry.target+' blocks':entry.id==='no_damage'?entry.current.toFixed(1)+' health damage':entry.current+' abilities used';text('small',value+(entry.reason?' · '+entry.reason:''),row);}
   parent.append(row);
  }
 }
 function pendingGold(run){
  const o=run?.objectives;
  if(run?.practice||run?.status!=='cleared'||run.room!==2||!o?.finished||o.banked||o.reward_gold||o.reward_per_objective!==5)return 0;
  return o.entries.filter(e=>e.status==='complete').length*o.reward_per_objective;
 }
 function update(run,paused){
  lastRun=run;lastPaused=paused;
  root.hidden=!toggle.checked||!!run?.practice||!!document.getElementById('rift-app').dataset.practice;
  if(root.hidden)return;
  const current=run?.objectives,previous=run?.last_objectives;
  const next=JSON.stringify([current,previous,run?.objective_history,paused,!!run,options],(name,value)=>name==='current'&&typeof value==='number'?Math.floor(value):value);
  if(next===key)return;key=next;
  caption.textContent=current?'Mission '+current.mission+' · '+(current.finished?'Results · '+(current.banked?'Banked':'Not banked'):paused?'Paused':'In progress'):run?'Tracking starts with your next mission.':'Before you enter: optional goals for all three tiers.';
  document.getElementById('rift-objective-reward-terms').textContent=current&&!current.reward_per_objective?'This saved mission has no objective bonus offer. Failures do not end your mission.':'Earn 5 gold per completed objective when you bank the final tier. Objective bonuses are separate from fight loot. Failures do not end your mission.';
  document.getElementById('rift-objective-reward-total').textContent=current?.banked?'Objective bonus banked: '+(current.reward_gold||0)+' gold.':pendingGold(run)?'Objective bonus ready to bank: '+pendingGold(run)+' gold.':'';
  rows(list,current?.entries||options,!current);
  const history=document.getElementById('rift-objective-history'),historyList=document.getElementById('rift-objective-history-list');historyList.replaceChildren();
  for(const [difficulty,counts] of Object.entries(run?.objective_history||{}).sort(([a],[b])=>a.localeCompare(b))){
   for(const [id,count] of Object.entries(counts).sort(([a],[b])=>a.localeCompare(b)))if(count>0)text('li',difficulty+' · '+(options.find(entry=>entry.id===id)?.name||id)+' — '+count,historyList);
  }
  history.hidden=!historyList.children.length;
  last.hidden=!previous;
  if(previous){document.getElementById('rift-objectives-last-title').textContent='Previous mission '+previous.mission+' objectives · '+(previous.banked?'Banked · Objective bonus: '+(previous.reward_gold||0)+' gold':'Not banked');rows(lastList,previous.entries,false);}
 }
 window.RiftObjectives={pendingGold,init(values){options=values;key='';update(null,false);},update};
})();
