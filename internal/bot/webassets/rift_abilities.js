(function(){
  'use strict';
  const roles={builder:['＋','Builder'],finisher:['◆','Finisher'],ultimate:['★','Ultimate']};
  function update(button,skill,run,reason,ultimate=false){
    const role=ultimate?'ultimate':skill.role,identity=roles[role];
    let ring=button.querySelector('.rift-cooldown-ring');
    if(!ring){ring=document.createElement('span');ring.className='rift-cooldown-ring';ring.setAttribute('aria-hidden','true');button.append(ring);}
    let marker=button.querySelector('.rift-role-mark');
    if(identity&&!marker){marker=document.createElement('span');marker.className='rift-role-mark';marker.setAttribute('aria-hidden','true');button.querySelector('kbd').after(marker);}
    if(marker){marker.textContent=identity?.[0]||'';marker.hidden=!identity;}
    const remaining=Math.max(0,run.skill_timers[skill.id]||0),duration=Math.max(0,skill.cooldown),cooling=remaining>0;
    ring.style.setProperty('--cooldown-progress',(duration?Math.min(1,remaining/duration)*100:0)+'%');
    ring.hidden=!cooling;
    const name=skill.name+(identity?' · '+identity[1]:'');
    button.setAttribute('aria-label',name+' · '+reason);
    button.title=name+' · '+skill.cost+' MP · '+skill.cooldown+'s cooldown · '+reason;
    button.dataset.abilityRole=identity?role:'optional';
  }
  window.RiftAbilities={update};
})();
