(function(){
  'use strict';
  const $=id=>document.getElementById(id),numbers=new Intl.NumberFormat(undefined,{maximumFractionDigits:0});
  let announced='',summaryKey='';
  let coachingDismissed=false;
  try{coachingDismissed=localStorage.getItem('riftClassCoachingDismissed')==='true';}catch(_){}
  const coaching=document.createElement('div');coaching.id='rift-class-coaching';coaching.hidden=true;coaching.innerHTML='<p role="status"></p><button type="button">Dismiss class coaching</button>';document.querySelector('.rift-run-statistics').before(coaching);
  coaching.querySelector('button').onclick=()=>{coachingDismissed=true;coaching.hidden=true;document.querySelector('.rift-run-statistics > summary').focus();try{localStorage.setItem('riftClassCoachingDismissed','true');}catch(_){}};
  let recentBaseline=null,recentChanges=[];
  function recentDamage(run,replay){
    const current={id:run.id,seconds:run.stats?.seconds||0,damage:run.stats?.damage_taken||0,healing:run.stats?.healing||0};
    const previous=recentBaseline;
    if(replay||!previous||previous.id!==current.id||current.seconds<previous.seconds||current.damage<previous.damage||current.healing<previous.healing)recentChanges=[];
    else if(current.damage>previous.damage||current.healing>previous.healing)recentChanges.push({seconds:current.seconds,damage:current.damage-previous.damage,healing:current.healing-previous.healing});
    recentBaseline=current;recentChanges=recentChanges.filter(change=>current.seconds-change.seconds<5).slice(-256);
    const totals=recentChanges.reduce((total,change)=>({damage:total.damage+change.damage,healing:total.healing+change.healing}),{damage:0,healing:0});
    put($('rift-recent-damage'),'Last 5 combat seconds: '+numbers.format(totals.damage)+' damage taken · '+numbers.format(totals.healing)+' healing');
  }
  const put=(node,value)=>{if(node&&node.textContent!==String(value))node.textContent=value;};
  const attr=(node,key,value)=>{if(node&&node.getAttribute(key)!==String(value))node.setAttribute(key,String(value));};
  function meter(selector,label,current,max){const node=document.querySelector(selector);attr(node,'role','meter');attr(node,'aria-label',label);attr(node,'aria-valuemin',0);attr(node,'aria-valuemax',Math.max(1,max));attr(node,'aria-valuenow',Math.min(max,Math.max(0,current)));}
  function duration(seconds){seconds=Math.max(0,Math.floor(seconds));return Math.floor(seconds/60)+':'+String(seconds%60).padStart(2,'0');}
  function reason(skill,run,playing){
    if(!['fighting','cleared'].includes(run.status))return 'Expedition ended';
    if(!playing||run.paused)return 'Paused';
    const remaining=run.skill_timers[skill.id]||0;
    if(remaining>0)return remaining.toFixed(1)+' seconds cooldown';
    if(run.player.mana<skill.cost)return Math.ceil(skill.cost-run.player.mana)+' more mana needed';
    return 'Ready · '+skill.cost+' mana';
  }
  function update(run,playing,replay=false){
    recentDamage(run,replay);
    const living=run.enemies.filter(e=>e.hp>0),stats=run.stats||{},boss=living.find(e=>e.kind==='boss');
    $('rift-paused-badge').hidden=!['fighting','cleared'].includes(run.status)||(playing&&!run.paused);
    put($('rift-paused-badge'),run.paused?'Paused':'Not running');
    put($('rift-facing'),run.player.facing<0?'← Facing left':'Facing right →');
    put($('rift-guard-reduction'),run.player.guard&&['fighting','cleared'].includes(run.status)?(playing&&!run.paused?'Guard: 82% frontal reduction after armor':'Guard paused'):'Guard inactive');
    put($('rift-enemy-count'),living.length+' '+(living.length===1?'enemy':'enemies')+' remaining');
    put($('rift-combat-time'),duration(stats.seconds||0)+' combat');
    const continues=run.room<2||($('rift-auto').checked&&run.level?.id<100),health=Math.min(run.player.max_hp,run.player.hp+run.player.max_hp*.25);
    put($('rift-recovery-preview'),continues?'Continue: +'+Math.max(0,health-run.player.hp).toFixed(1)+' HP → '+health.toFixed(1)+'/'+run.player.max_hp.toFixed(1)+' HP; mana refills to 100. Leaving gives no recovery.':'Final checkpoint: bank your rewards and finish. No next-tier recovery.');
    put($('rift-objective'),run.status==='fighting'?(boss?'Defeat '+boss.name+' and its defenders':'Clear the enemy patrol'):run.status==='cleared'?'Room secured · Loot ready to bank':run.status==='defeated'?'Expedition ended':'Rewards secured');
    put($('rift-jump-ready'),(run.skill_timers.jump||0)>0?'Jump '+run.skill_timers.jump.toFixed(1)+'s':'Jump ready');
    put($('rift-combo-step'),'Strike '+(run.combo||0)+'/3');
    put($('rift-barrier-state'),run.barrier>0?'Barrier '+numbers.format(run.barrier):'No barrier');
    put($('rift-slow-state'),run.skill_timers.slowed>0?'Slowed '+run.skill_timers.slowed.toFixed(1)+'s':'Normal speed');
    const marked=living.find(e=>e.id===run.marked);put($('rift-mark-state'),marked?'Marked: '+marked.name:'No marked target');
    const finisher=run.build.signatures?.find(s=>s.role==='finisher');
    put($('rift-finisher-state'),!finisher?'Class abilities unlock in Abyss':run.resource>0?'Finisher: '+reason(finisher,run,playing)+' · '+run.resource+' charges':'Build charges with '+window.RiftControls.label('signature0'));
    const builder=run.build.signatures?.find(s=>s.role==='builder');
    coaching.hidden=coachingDismissed||!builder||!finisher||(stats.empty_finishers||0)<3;
    if(!coaching.hidden)put(coaching.querySelector('p'),'Three or more finishers used no charges. Practice '+builder.name+' ('+window.RiftControls.label('signature0')+') before '+finisher.name+' ('+window.RiftControls.label('signature1')+'). Build up to three charges, then spend them with your finisher.');
    meter('#rift-vitals .hp','Player health',run.player.hp,run.player.max_hp);
    meter('#rift-vitals .mana','Player mana',run.player.mana,100);
    if(boss)meter('#rift-boss .hp',boss.name+' health',boss.hp,boss.max_hp);
    for(const [id,skills] of [['rift-skills',run.build.skills],['rift-signatures',[...(run.build.signatures||[]),...(run.build.ultimate?[run.build.ultimate]:[])]]]){
      [...$(id).children].forEach((button,index)=>{const skill=skills[index];if(!skill)return;const why=reason(skill,run,playing);window.RiftAbilities.update(button,skill,run,why,skill===run.build.ultimate);});
    }
    const values=[['Enemies defeated',stats.kills],['Bosses defeated',stats.bosses],['Rooms cleared',stats.rooms_cleared],['Damage dealt',stats.damage_dealt],['Damage taken',stats.damage_taken],['Healing received',stats.healing],['Guard prevented',stats.guard_blocked],['Barrier prevented',stats.barrier_blocked],['Armor prevented',stats.armor_blocked],['Largest hit',stats.largest_hit],['Mana spent',stats.mana_spent],['Skills cast',stats.skills_cast],['Charged finishers',stats.charged_finishers],['Finishers without charges',stats.empty_finishers],['Charges spent',stats.charges_spent],['Basic attacks',stats.attacks],['Successful guards',stats.guards],['Jumps',stats.jumps]];
    const abilityUses=[...run.build.skills.map(skill=>[skill,'optional']),...(run.build.signatures||[]).map(skill=>[skill,skill.role||'class']),...(run.build.ultimate?[[run.build.ultimate,'ultimate']]:[])];
    let attributed=0,attributedMana=0,attributedHealing=0,attributedBarrier=0;for(const [skill,kind] of abilityUses){const casts=stats.skill_uses?.[skill.id]||0,mana=stats.skill_mana?.[skill.id]||0,healing=stats.skill_healing?.[skill.id]||0,barrier=stats.skill_barrier?.[skill.id]||0;attributed+=casts;attributedMana+=mana;attributedHealing+=healing;attributedBarrier+=barrier;const name=skill.name+' ('+kind+')';values.push(['Casts · '+name,casts],['Mana · '+name,mana],['Hits · '+name,stats.skill_hits?.[skill.id]||0],['Healing · '+name,healing],['Absorbed · '+name,barrier]);}
    if((stats.skills_cast||0)>attributed)values.push(['Earlier casts without per-skill records',stats.skills_cast-attributed]);
    if((stats.mana_spent||0)>attributedMana+.001)values.push(['Earlier mana without per-skill records',stats.mana_spent-attributedMana]);
    if((stats.healing||0)>attributedHealing+.001)values.push(['Healing without per-skill records',stats.healing-attributedHealing]);
    if((stats.barrier_blocked||0)>attributedBarrier+.001)values.push(['Absorption without per-skill records',stats.barrier_blocked-attributedBarrier]);
    const key=JSON.stringify(values);
    if(summaryKey!==key){summaryKey=key;$('rift-statistics').replaceChildren();for(const [label,value] of values){const dt=document.createElement('dt'),dd=document.createElement('dd');dt.textContent=label;dd.textContent=numbers.format(value||0);$('rift-statistics').append(dt,dd);}}
    const state=[run.id,run.level?.id,run.room,run.status,run.paused].join(':');
    if(state!==announced){
      announced=state;
      if(run.status==='cleared')put($('rift-announcer'),'Room '+(run.room+1)+' cleared. Loot is ready to bank.');
      else if(run.status==='defeated')put($('rift-announcer'),'Expedition ended. Banked rewards are safe.');
      else if(['complete','banked'].includes(run.status))put($('rift-announcer'),'Expedition finished. '+numbers.format(run.banked_gold)+' gold and '+numbers.format(run.banked_items.length)+' '+(run.banked_items.length===1?'item':'items')+' banked.');
      else if(run.paused)put($('rift-announcer'),'Expedition paused.');
    }
  }
  window.RiftHUD={update,duration};
})();
