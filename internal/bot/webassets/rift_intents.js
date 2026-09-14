(function(){
  'use strict';
  let queue=[],identity='',mode='hold';
  try{if(localStorage.getItem('riftAbilityMode')==='tap')mode='tap';}catch(_){}
  const label=document.createElement('label'),select=document.createElement('select'),description=document.createElement('p');select.id='rift-ability-mode';description.id='rift-ability-mode-description';select.setAttribute('aria-describedby',description.id);
  for(const [value,text] of [['hold','Hold to repeat abilities'],['tap','One ability cast per press']]){const option=document.createElement('option');option.value=value;option.textContent=text;select.append(option);}
  label.className='rift-ability-setting';label.append(document.createTextNode('Ability input'),select);document.querySelector('.rift-settings').append(label,description);
  function explain(){select.value=mode;description.textContent=(mode==='hold'?'Holding repeats an ability when ready. ':'Each press requests one ability cast. ')+'Taps queue in order for up to 1.2 seconds. Pausing clears queued casts. Movement, attack and guard still respond while an ability is unavailable.';}
  select.onchange=()=>{mode=select.value;queue=[];try{localStorage.setItem('riftAbilityMode',mode);}catch(_){}explain();window.dispatchEvent(new Event('riftintentchange'));};explain();
  const ability=action=>/^(signature[01]|skill[0-2]|ultimate|skill:.+)$/.test(action);
  function entries(run){return [...(run?.build.signatures||[]).map((skill,i)=>({action:'signature'+i,skill})),...(run?.build.ultimate?[{action:'ultimate',skill:run.build.ultimate}]:[]),...(run?.build.skills||[]).map((skill,i)=>({action:'skill'+i,skill}))];}
  window.RiftIntents={
    press(action){const now=performance.now();queue=queue.filter(item=>item.until>now);if(ability(action)&&queue.length<8)queue.push({action,until:now+1200});},
    cancel(action){for(let i=queue.length-1;i>=0;i--)if(queue[i].action===action){queue.splice(i,1);break;}},
    reset(){queue=[];},
    sync(run,replay){const next=[run.id,run.level?.id,run.room].join(':');if(replay||identity!==next)queue=[];identity=next;},
    take(run,held,guard){
      const now=performance.now(),skills=entries(run);queue=queue.filter(item=>item.until>now);
      while(queue.length){
        const item=queue[0],entry=skills.find(entry=>entry.action===item.action||'skill:'+entry.skill.id===item.action);
        if(!entry){queue.shift();continue;}
        const skill=entry.skill,available=run.player.mana>=skill.cost&&!(run.skill_timers[skill.id]>0);
        if(guard||!available)return {skill:'',wait:false};
        if(run.player.cooldown>0)return {skill:'',wait:true};
        queue.shift();return {skill:skill.id,wait:false};
      }
      return {skill:mode==='hold'?(skills.find(entry=>held(entry.action)||held('skill:'+entry.skill.id))?.skill.id||''):'',wait:false};
    }
  };
})();
