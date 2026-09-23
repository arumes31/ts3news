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
   if(!preview){const value=entry.id==='timed'?Math.floor(entry.current)+' / '+entry.target+' combat seconds':entry.id==='aerial_finish'?entry.current+' / '+entry.target+' aerial finishes':entry.id==='treasure_capture'?entry.current+' / '+entry.target+' goblins captured':entry.id==='hazard_avoidance'?entry.current+' hazard contacts':entry.id==='save_ultimate'?entry.current+' ultimates cast':entry.id==='finisher'?entry.current+' / '+entry.target+' charged finishers':entry.id==='guard'?entry.current+' / '+entry.target+' blocks':entry.id==='no_damage'?entry.current.toFixed(1)+' health damage':entry.current+' abilities used';text('small',value+(entry.reason?' · '+entry.reason:''),row);}
   parent.append(row);
  }
 }
 function update(run,paused){
  lastRun=run;lastPaused=paused;
  root.hidden=!toggle.checked||!!run?.practice||!!document.getElementById('rift-app').dataset.practice;
  if(root.hidden)return;
  const current=run?.objectives,previous=run?.last_objectives;
  const next=JSON.stringify([current,previous,run?.objective_history,paused,!!run,options],(name,value)=>name==='current'&&typeof value==='number'?Math.floor(value):value);
  if(next===key)return;key=next;
  caption.textContent=current?'Mission '+current.mission+' · '+(current.finished?'Results · '+(current.banked?'Banked':'Not banked'):paused?'Paused':'In progress'):run?'Tracking starts with your next mission.':'Before you enter: optional goals for all three tiers.';
  rows(list,current?.entries||options,!current);
  const history=document.getElementById('rift-objective-history'),historyList=document.getElementById('rift-objective-history-list');historyList.replaceChildren();
  for(const [difficulty,counts] of Object.entries(run?.objective_history||{}).sort(([a],[b])=>a.localeCompare(b))){
   for(const [id,count] of Object.entries(counts).sort(([a],[b])=>a.localeCompare(b)))if(count>0)text('li',difficulty+' · '+(options.find(entry=>entry.id===id)?.name||id)+' — '+count,historyList);
  }
  history.hidden=!historyList.children.length;
  last.hidden=!previous;
  if(previous){document.getElementById('rift-objectives-last-title').textContent='Previous mission '+previous.mission+' objectives · '+(previous.banked?'Banked':'Not banked');rows(lastList,previous.entries,false);}
 }
 window.RiftObjectives={init(values){options=values;key='';update(null,false);},update};
})();
