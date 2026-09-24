(function(){
  'use strict';
  let queue=[],identity='',mode='hold';
  try{if(localStorage.getItem('riftAbilityMode')==='tap')mode='tap';}catch(_){}
  const label=document.createElement('label'),select=document.createElement('select'),description=document.createElement('p');select.id='rift-ability-mode';description.id='rift-ability-mode-description';select.setAttribute('aria-describedby',description.id);
  for(const [value,text] of [['hold','Hold to repeat abilities'],['tap','One ability cast per press']]){const option=document.createElement('option');option.value=value;option.textContent=text;select.append(option);}
  label.className='rift-ability-setting';label.append(document.createTextNode('Ability input'),select);document.querySelector('.rift-settings').append(label,description);
  function explain(){select.value=mode;description.textContent=(mode==='hold'?'Holding repeats an ability when ready. ':'Each press requests one ability cast. ')+'Taps queue in order for up to 1.2 seconds. Pausing clears queued casts. Movement, attack and guard still respond while an ability is unavailable.';}
  select.onchange=()=>{mode=select.value;queue=[];try{localStorage.setItem('riftAbilityMode',mode);}catch(_){}explain();window.dispatchEvent(new Event('riftintentchange'));};explain();
  const healthLabel=document.createElement('label'),healthConfirm=document.createElement('input'),healthPrompt=document.createElement('p');
  healthConfirm.type='checkbox';healthConfirm.id='rift-confirm-health-cost';
  try{healthConfirm.checked=localStorage.getItem('riftConfirmHealthCost')==='true';}catch(_){}
  healthLabel.append(healthConfirm,document.createTextNode('Confirm health-cost skills with a second press'));
  document.querySelector('.rift-settings').append(healthLabel);
  healthPrompt.id='rift-health-confirmation';healthPrompt.hidden=true;healthPrompt.setAttribute('role','status');document.querySelector('.rift-classbar').after(healthPrompt);
  let healthIntent=null,healthExpiry=0;
  function clearHealthIntent(){healthIntent=null;clearTimeout(healthExpiry);healthPrompt.hidden=true;healthPrompt.textContent='';}
  healthConfirm.onchange=()=>{clearHealthIntent();queue=[];try{localStorage.setItem('riftConfirmHealthCost',String(healthConfirm.checked));}catch(_){}window.dispatchEvent(new Event('riftintentchange'));};
  function confirmedSkill(skill,run,explicit){
    const cost=window.RiftAbilities.healthCost(run,skill),now=performance.now();
    if(!healthConfirm.checked||cost<=0){clearHealthIntent();return true;}
    if(healthIntent?.id===skill.id&&healthIntent.until>now){
      if(explicit){clearHealthIntent();return true;}
      return false;
    }
    clearHealthIntent();
    if(explicit){
      healthIntent={id:skill.id,until:now+2000};healthPrompt.hidden=false;
      healthPrompt.textContent=skill.name+' spends '+cost.toFixed(1)+' HP (leaves at least 1 HP). Release and press this ability again within 2 seconds to cast. Holding does not confirm.';
      healthExpiry=setTimeout(clearHealthIntent,2000);
    }
    return false;
  }
  const ability=action=>/^(signature[01]|skill[0-2]|ultimate|skill:.+)$/.test(action);
  function entries(run){return [...(run?.build.signatures||[]).map((skill,i)=>({action:'signature'+i,skill})),...(run?.build.ultimate?[{action:'ultimate',skill:run.build.ultimate}]:[]),...(run?.build.skills||[]).map((skill,i)=>({action:'skill'+i,skill}))];}
  const readout=document.createElement('p'),readoutLabel=document.createElement('label'),enabled=document.createElement('input');readout.id='rift-input-readout';readout.hidden=true;readout.setAttribute('aria-live','off');readout.setAttribute('aria-label','Input feedback');document.querySelector('.rift-classbar').after(readout);
  enabled.type='checkbox';enabled.id='rift-input-readout-enabled';try{enabled.checked=JSON.parse(localStorage.getItem('riftInputReadout'))===true;}catch(_){}readoutLabel.append(enabled,document.createTextNode('Show recognized controls and blocked actions'));document.querySelector('.rift-settings').append(readoutLabel);
  let currentRun=null,recognized='',detail='',until=0,expiry=0;
  const names={attack:'Attack',jump:'Jump',guard:'Guard',up:'Move up',down:'Move down',left:'Move left',right:'Move right',stop:'Stop movement'};
  function paint(){
    clearTimeout(expiry);const remaining=until-performance.now(),paused=currentRun?.paused;
    const value=paused?'Input paused':remaining>0&&recognized?'Recognized: '+recognized+(detail?' · '+detail:''):'';
    if(readout.textContent!==value)readout.textContent=value;readout.hidden=!enabled.checked||!value;
    if(remaining>0&&!paused)expiry=setTimeout(paint,remaining+1);
  }
  function recognize(action){const entry=entries(currentRun).find(entry=>entry.action===action||'skill:'+entry.skill.id===action),name=entry?.skill.name||names[action];if(!name)return;recognized=name;detail='';until=performance.now()+2000;paint();}
  function result(skill='',wait=false,reason=''){detail=reason;paint();return {skill,wait};}
  function clearReadout(){recognized=detail='';until=0;paint();}
  enabled.onchange=()=>{try{localStorage.setItem('riftInputReadout',JSON.stringify(enabled.checked));}catch(_){}clearReadout();};
  window.RiftIntents={
    recognize,
    press(action){recognize(action);if(action==='jump')window.RiftJump.press();const now=performance.now();queue=queue.filter(item=>item.until>now);if(ability(action)){if(queue.length<8)queue.push({action,until:now+1200});else result('',false,'Ability queue full');}},
    cancel(action){if(action==='jump')window.RiftJump.reset();for(let i=queue.length-1;i>=0;i--)if(queue[i].action===action){queue.splice(i,1);break;}clearReadout();},
    reset(){window.RiftJump.reset();queue=[];clearReadout();clearHealthIntent();},
    sync(run,replay){window.RiftJump.sync(run,replay);currentRun=run;const next=[run.id,run.level?.id,run.room].join(':');if(replay||identity!==next){queue=[];clearReadout();clearHealthIntent();}identity=next;paint();},
    take(run,held,guard){
      const now=performance.now(),skills=entries(run);queue=queue.filter(item=>item.until>now);
      while(queue.length){
        const item=queue[0],entry=skills.find(entry=>entry.action===item.action||'skill:'+entry.skill.id===item.action);
        if(!entry){queue.shift();continue;}
        const skill=entry.skill,available=run.player.mana>=skill.cost&&!(run.skill_timers[skill.id]>0);
        if(guard||!available){
          if(!guard){
            if(run.player.mana<skill.cost)window.RiftAudio?.playEmptyMana?.();
            else if(run.skill_timers[skill.id]>0)window.RiftAudio?.playCooldownRejection?.();
          }
          return result('',false,skill.name+': '+(guard?'Release guard':run.player.mana<skill.cost?'Not enough mana':'Cooldown '+run.skill_timers[skill.id].toFixed(1)+'s'));
        }
        if(run.player.cooldown>0)return result('',true,skill.name+': Casting recovery');
        queue.shift();if(!confirmedSkill(skill,run,true))return result('',false,'Health cost: press again to confirm');return result(skill.id,false,'Requested: '+skill.name);
      }
      const skill=mode==='hold'?skills.find(entry=>held(entry.action)||held('skill:'+entry.skill.id))?.skill:null;
      if(skill&&!guard){
        if(run?.player?.mana<skill.cost)window.RiftAudio?.playEmptyMana?.();
        else if(run?.skill_timers?.[skill.id]>0)window.RiftAudio?.playCooldownRejection?.();
      }
      if(skill&&!confirmedSkill(skill,run,false))return result('');
      return result(skill?.id||'');
    }
  };
})();
