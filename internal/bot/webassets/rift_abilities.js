(function(){
  'use strict';
  const roles={builder:['＋','Builder'],finisher:['◆','Finisher'],ultimate:['★','Ultimate']};
  const healthCost=(run,skill)=>run.build?.class==='voidwalker'&&skill?.role==='finisher'&&run.resource>0?Math.min(Math.max(0,run.player.hp-1),run.player.max_hp*.05):0;
  function signatureHelp(build){
    const skills=build?.signatures||[],builder=skills.some(s=>s.role==='builder'),finisher=skills.some(s=>s.role==='finisher');
    if(builder&&finisher)return '';
    const missing=!builder&&!finisher?'Builder and finisher are':!builder?'Builder is':'Finisher is';
    const foundation=!build?.base_class||build.class===build.base_class;
    const unlock=foundation?(!builder?'Earn your first class point in Abyss to unlock the foundation builder. ':'')+(!finisher?'The foundation finisher unlocks at three class points. ':'')+'Spend five foundation talent points in My Build to unlock a subclass, then select it. ':'Check your selected subclass and class loadout in Abyss My Build. ';
    return missing+' missing from this build. '+unlock+'Start a new Brawl expedition after unlocking or changing class abilities; this expedition keeps its saved build. Basic attacks and equipped skills remain available.';
  }
  function oracleHealing(run){
    const builder=run.build.signatures?.find(s=>s.role==='builder'),fraction=builder?.reference?.healing??builder?.heal??(builder?.kind==='heal'?.15:0);
    if(!builder||!(fraction>0))return 'Healing stops at maximum HP. Excess healing is discarded; it does not become a barrier.';
    const raw=run.player.max_hp*fraction,restored=Math.min(Math.max(0,run.player.max_hp-run.player.hp),raw),overflow=Math.max(0,raw-restored);
    return builder.name+': '+restored.toFixed(1)+' HP restored now; '+overflow.toFixed(1)+' HP overflow discarded (no barrier). '+(run.resource>=3?'Charges full: no additional '+(run.build.resource||'class')+' charge.':'Successful cast grants 1 '+(run.build.resource||'class')+' charge even at full HP (maximum 3).');
  }
  function chargeBenefits(run){
    const skill=run.build.signatures?.find(s=>s.role==='finisher');
    if(!skill)return 'Equip a class finisher in Abyss to spend charges.';
    const charges=Math.max(0,Math.min(3,run.resource||0));
    if(!charges)return 'No charge bonus yet. Build charges before your finisher.';
    const b=run.build,p=run.player,parts=[];
    if(!['shield','heal'].includes(skill.kind))parts.push('finisher damage ×'+(1+charges*.2).toFixed(1)+' before class bonuses and defenses');
    const recover=percent=>Math.min(Math.max(0,p.max_hp-p.hp),p.max_hp*percent*charges).toFixed(1)+' HP recovery now (capped at full health)';
    switch(b.class){
      case 'vanguard':parts.push('+30 percentage points of armor piercing');break;
      case 'berserker':parts.push('damage ×1.25 against enemies at 50% HP or below');break;
      case 'bloodblade':parts.push(recover(.04));break;
      case 'marksman':parts.push('+60 percentage points of armor piercing against the marked target');break;
      case 'beastmaster':parts.push('damage ×'+(1+Math.min(3,b.pets||0)*.1).toFixed(1)+' from '+Math.min(3,b.pets||0)+' pets');break;
      case 'elementalist':parts.push('damage ×1.2 against the marked target');break;
      case 'chronomancer':parts.push('reduce other ability and jump cooldowns by up to 1.5s');break;
      case 'oracle':parts.push('damage ×1.15');break;
      case 'geomancer':parts.push('+45 percentage points of armor piercing');break;
      case 'voidwalker':parts.push('spend '+healthCost(run,skill).toFixed(1)+' HP; damage ×1.25 if above 1 HP at impact; +25 percentage points of armor piercing against the marked target');break;
      case 'runesmith':parts.push('+'+Math.max(0,Math.min(15+(b.armor||0)*2,p.max_hp*.5-(run.barrier||0))).toFixed(1)+' barrier now (capped at 50% maximum HP)');if(b.relic)parts.push('damage ×1.15 with equipped relic');break;
      case 'alchemist':parts.push(recover(.03),'+35 percentage points of armor piercing against the marked target');break;
    }
    if(parts.some(part=>part.includes('armor piercing')))parts.push('total armor piercing capped at 100%');
    if(!['shield','heal'].includes(skill.kind))parts.push('boss hits stagger for 0.45s');
    return charges+' charges spent: '+(parts.length?parts.join('; '):'no additional charge effect for this ability')+'.';
  }
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
  function aimHelp(skill,run){
    const ref=skill?.reference,p=run?.player;
    if(!ref||!p)return '';
    if(ref.target==='self')return 'Self-targeted: facing and lane depth cannot cause a miss.';
    const target=(run.enemies||[]).filter(enemy=>enemy.hp>0).reduce((nearest,enemy)=>!nearest||Math.hypot(enemy.x-p.x,enemy.y-p.y)<Math.hypot(nearest.x-p.x,nearest.y-p.y)?enemy:nearest,null);
    if(!target)return 'No living target nearby.';
    const dx=target.x-p.x,dy=target.y-p.y,notes=[];
    if(ref.target==='projectile'&&dx*(p.facing<0?-1:1)<0)notes.push('behind you: turn '+(dx<0?'left':'right')+' before casting; projectiles travel forward');
    if(Math.abs(dy)>=ref.depth)notes.push('outside the lane: move '+(dy<0?'up':'down')+' until lane separation is less than '+ref.depth+' units');
    if(ref.target==='area'&&Math.abs(dx)>=ref.horizontal)notes.push('too far horizontally: move closer than '+ref.horizontal+' units');
    if(ref.target==='area')notes.push('this area skill hits in either facing direction');
    if(!notes.length)notes.push('currently aligned; movement, cover or another enemy may still prevent a hit');
    return 'Nearest target, '+target.name+' — '+notes.join('; ')+'.';
  }
  const rejectionHints=new WeakMap();
  function updateRejectionHint(button,reason){
    let state=rejectionHints.get(button);
    if(!state){
      const hint=document.createElement('span');hint.className='rift-ability-rejection';hint.hidden=true;hint.setAttribute('aria-hidden','true');button.append(hint);
      state={hint,reason:'',until:0};rejectionHints.set(button,state);
      const show=()=>{if(button.disabled||!state.reason)return;state.until=performance.now()+2000;hint.textContent=state.reason;hint.hidden=false;};
      button.addEventListener('pointerdown',show);
      button.addEventListener('click',event=>{if(event.detail===0)show();});
    }
    state.reason=/seconds cooldown|more mana needed/.test(reason)?reason:'';
    if(!state.reason)state.until=0;
    state.hint.hidden=!state.reason||performance.now()>=state.until;
    state.hint.textContent=state.hint.hidden?'':state.reason;
  }
  function update(button,skill,run,reason,ultimate=false){
    updateRejectionHint(button,reason);
    let icon=button.querySelector('.rift-catalog-icon');if(!icon){icon=document.createElement('span');icon.className='rift-catalog-icon';icon.setAttribute('aria-hidden','true');button.prepend(icon);}if(icon.textContent!==(skill.icon||'✦'))icon.textContent=skill.icon||'✦';
    const role=ultimate?'ultimate':skill.role,identity=roles[role];
    let ring=button.querySelector('.rift-cooldown-ring');
    if(!ring){ring=document.createElement('span');ring.className='rift-cooldown-ring';ring.setAttribute('aria-hidden','true');button.append(ring);}
    const kbd=button.querySelector('kbd');
    if(kbd)kbd.setAttribute('aria-hidden','true');
    let marker=button.querySelector('.rift-role-mark');
    if(identity&&!marker){marker=document.createElement('span');marker.className='rift-role-mark';marker.setAttribute('aria-hidden','true');if(kbd)kbd.after(marker);else button.prepend(marker);}
    if(marker){
      marker.textContent=identity?identity[0]+' '+identity[1]:'';marker.hidden=!identity;
      const label=button.querySelector('.rift-action-label');
      if(label&&!button.querySelector('.rift-ability-identity')){const group=document.createElement('span');group.className='rift-ability-identity';label.before(group);group.append(label,marker);}
    }
    const remaining=Math.max(0,run.skill_timers[skill.id]||0),duration=Math.max(0,skill.cooldown),cooling=remaining>0;
    ring.style.setProperty('--cooldown-progress',(duration?Math.min(1,remaining/duration)*100:0)+'%');
    ring.hidden=!cooling;
    const cost=healthCost(run,skill),name=skill.name+(identity?' · '+identity[1]:'')+(cost>0?' · Spends '+cost.toFixed(1)+' HP':'');
    let healthLabel=button.querySelector('.rift-health-cost-label');
    if(cost>0&&!healthLabel){healthLabel=document.createElement('span');healthLabel.className='rift-health-cost-label';healthLabel.setAttribute('aria-hidden','true');(button.querySelector('.rift-ability-identity')||button).append(healthLabel);}
    if(healthLabel){healthLabel.hidden=cost<=0;healthLabel.textContent='−'+cost.toFixed(1)+' HP';}
    button.setAttribute('aria-label',name+' · '+reason);
    button.removeAttribute('title');
    button.dataset.abilityRole=identity?role:'optional';
    button.dataset.abilityName=skill.name;
  }
  window.RiftAbilities={update,healthCost,describe,chargeBenefits,oracleHealing,signatureHelp,aimHelp};
})();
