(function(){
  'use strict';
  const roles={builder:['＋','Builder'],finisher:['◆','Finisher'],ultimate:['★','Ultimate']};
  const healthCost=(run,skill)=>run.build?.class==='voidwalker'&&skill?.role==='finisher'&&run.resource>0?Math.min(Math.max(0,run.player.hp-1),run.player.max_hp*.05):0;
  function describe(skill,build){
    const ref=skill.reference;if(!ref)return 'Combat details unavailable for this skill.';
    const lines=[];
    if(ref.target==='self')lines.push('Self-targeted. Facing and lane depth do not affect this skill.');
    else if(ref.target==='area')lines.push((skill.kind==='slash'?'Melee area':'Area attack')+': hits enemies in either direction, less than '+ref.horizontal+' horizontal units and '+ref.depth+' lane-depth units away.');
    else lines.push('Ranged projectile: travels in your facing direction and hits the first enemy within '+ref.horizontal+' horizontal and '+ref.depth+' lane-depth units of the projectile. Enemies behind you or outside that lane can be missed.');
    if(ref.healing>0)lines.push('Direct healing: '+Number((ref.healing*100).toFixed(1))+'% of maximum HP, capped at full health.');
    if(ref.barrier)lines.push('Grants a protective barrier, capped at half of maximum HP.');
    if(ref.target!=='self')lines.push(skill.pierce>0?'Base armor piercing: '+Number((Math.min(1,skill.pierce)*100).toFixed(1))+'%.':'No base armor piercing.');
    if(skill.role==='finisher'){
      const piercing={vanguard:[30,false],marksman:[60,true],geomancer:[45,false],voidwalker:[25,true],alchemist:[35,true]}[build.class];
      if(piercing&&ref.target!=='self')lines.push('Charged class bonus: '+piercing[0]+'% additional armor piercing'+(piercing[1]?' against the marked target':'')+'.');
      if(build.class==='bloodblade'||build.class==='alchemist')lines.push('Charged class recovery: '+(build.class==='bloodblade'?4:3)+'% of maximum HP per charge spent, capped at full health.');
      if(build.class==='runesmith')lines.push('Spending charges also grants a class barrier.');
      lines.push('Charged class bonuses can modify the base effects.');
    }
    return lines.join(' ');
  }
  function update(button,skill,run,reason,ultimate=false){
    const role=ultimate?'ultimate':skill.role,identity=roles[role];
    let ring=button.querySelector('.rift-cooldown-ring');
    if(!ring){ring=document.createElement('span');ring.className='rift-cooldown-ring';ring.setAttribute('aria-hidden','true');button.append(ring);}
    const kbd=button.querySelector('kbd');
    if(kbd)kbd.setAttribute('aria-hidden','true');
    let marker=button.querySelector('.rift-role-mark');
    if(identity&&!marker){marker=document.createElement('span');marker.className='rift-role-mark';marker.setAttribute('aria-hidden','true');if(kbd)kbd.after(marker);else button.prepend(marker);}
    if(marker){marker.textContent=identity?.[0]||'';marker.hidden=!identity;}
    const remaining=Math.max(0,run.skill_timers[skill.id]||0),duration=Math.max(0,skill.cooldown),cooling=remaining>0;
    ring.style.setProperty('--cooldown-progress',(duration?Math.min(1,remaining/duration)*100:0)+'%');
    ring.hidden=!cooling;
    const cost=healthCost(run,skill),name=skill.name+(identity?' · '+identity[1]:'')+(cost>0?' · Spends '+cost.toFixed(1)+' HP':'');
    button.setAttribute('aria-label',name+' · '+reason);
    button.removeAttribute('title');
    button.dataset.abilityRole=identity?role:'optional';
  }
  window.RiftAbilities={update,healthCost,describe};
})();
