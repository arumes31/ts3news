(function(){
  'use strict';
  const $=id=>document.getElementById(id),numbers=new Intl.NumberFormat(undefined,{maximumFractionDigits:0});
  const slowName=run=>({ice:'Ice',poison:'Poison',thorns:'Thorns'}[run.slow_source]||'');
  const slowLabel=run=>'Slowed'+(slowName(run)?' by '+slowName(run):'');
  function hazardDefeatHint(source){
    if(!source||typeof source.kind!=='string')return '';
    if(source.kind==='collapse')return 'Defeated by the advancing collapse. Stay ahead of its moving edge and reach the exit seal. Jumping does not evade collapse damage.';
    const name=source.kind.charAt(0).toUpperCase()+source.kind.slice(1);
    const escape=source.jumpable===true?'Leave the warning zone before the pulse, or jump while it is active.':'Leave the warning zone before the pulse. This hazard cannot be jumped.';
    const extra=['ice','poison','thorns'].includes(source.kind)?' Guard reduces damage but does not prevent slowing.':source.kind==='void'?' Guard reduces damage but does not stop the pull.':' Guard reduces damage; it does not prevent contact.';
    return 'Defeated by '+name+'. '+escape+extra;
  }
  let announced='',summaryKey='',lastAnnouncedBossAttack='';
  let coachingDismissed=false;
  try{coachingDismissed=localStorage.getItem('riftClassCoachingDismissed')==='true';}catch(_){}
  const coaching=document.createElement('div');coaching.id='rift-class-coaching';coaching.hidden=true;coaching.innerHTML='<p role="status"></p><button type="button">Dismiss class coaching</button>';document.querySelector('.rift-run-statistics').before(coaching);
  coaching.querySelector('button').onclick=()=>{coachingDismissed=true;coaching.hidden=true;document.querySelector('.rift-run-statistics > summary').focus();try{localStorage.setItem('riftClassCoachingDismissed','true');}catch(_){}};
  const hintSetting=$('rift-context-hints');
  let hintsEnabled=true,hazardHintState=null,missHintState=null;
  try{hintsEnabled=localStorage.getItem('riftContextHints')!=='false';}catch(_){}
  hintSetting.checked=hintsEnabled;
  let hideDetailedStats=false;
  try{hideDetailedStats=localStorage.getItem('riftHideDetailedStats')==='true';}catch(_){}
  const hideDetailedSetting=$('rift-hide-detailed-stats');
  if(hideDetailedSetting){
    hideDetailedSetting.checked=hideDetailedStats;
    hideDetailedSetting.onchange=()=>{
      hideDetailedStats=hideDetailedSetting.checked;
      try{localStorage.setItem('riftHideDetailedStats',String(hideDetailedStats));}catch(_){}
      syncDetailedStatsVisibility();
    };
  }
  const toggleDetailedBtn=$('rift-toggle-encounter-stats');
  if(toggleDetailedBtn){
    toggleDetailedBtn.onclick=()=>{
      hideDetailedStats=!hideDetailedStats;
      try{localStorage.setItem('riftHideDetailedStats',String(hideDetailedStats));}catch(_){}
      if(hideDetailedSetting)hideDetailedSetting.checked=hideDetailedStats;
      syncDetailedStatsVisibility();
    };
  }
  function syncDetailedStatsVisibility(){
    const statsEl=$('rift-last-encounter-stats');
    const btn=$('rift-toggle-encounter-stats');
    if(statsEl)statsEl.hidden=hideDetailedStats;
    if(btn){
      btn.textContent=hideDetailedStats?'Show details':'Hide details';
      btn.setAttribute('aria-expanded',String(!hideDetailedStats));
    }
  }
  syncDetailedStatsVisibility();
  const hazardCoaching=document.createElement('p');hazardCoaching.id='rift-hazard-coaching';hazardCoaching.hidden=true;hazardCoaching.setAttribute('role','status');document.querySelector('.rift-combat-signals').after(hazardCoaching);
  const missCoaching=document.createElement('p');missCoaching.id='rift-miss-coaching';missCoaching.hidden=true;missCoaching.setAttribute('role','status');hazardCoaching.after(missCoaching);
  hintSetting.onchange=()=>{hintsEnabled=hintSetting.checked;hazardCoaching.hidden=true;hazardHintState=null;missCoaching.hidden=true;missHintState=null;if(!hintsEnabled){for(const id of ['rift-class-coaching','rift-terrain-hint','rift-boss-practice-lesson'])$(id).hidden=true;}window.dispatchEvent(new Event('riftcontextprefschange'));if(lastObservedRun)updateLastEncounter(lastObservedRun);try{localStorage.setItem('riftContextHints',String(hintsEnabled));}catch(_){}};
  let combatHintBudget=null,classHintState=null;
  function updateCombatHintBudget(run){
    const seconds=run.stats?.seconds||0;
    if(!combatHintBudget||combatHintBudget.id!==run.id||seconds<combatHintBudget.seconds)combatHintBudget={id:run.id,seconds,next:seconds};
    combatHintBudget.seconds=seconds;
  }
  function claimCombatHint(){
    if(combatHintBudget.seconds<combatHintBudget.next)return false;
    combatHintBudget.next=combatHintBudget.seconds+20;
    return true;
  }
  function classCoachingAllowed(run){
    if(run.status!=='fighting')return true;
    const key=[run.id,run.level?.id,run.room].join(':'),seconds=run.stats?.seconds||0;
    if(!classHintState||classHintState.key!==key||seconds<classHintState.seconds)classHintState={key,seconds,shown:false,until:0};
    classHintState.seconds=seconds;
    if(!classHintState.shown&&claimCombatHint()){
      classHintState.shown=true;classHintState.until=seconds+8;
    }
    return classHintState.shown&&seconds<classHintState.until;
  }
  function contextualHazardHint(run,replay){
    const key=[run.id,run.level?.id,run.room].join(':'),damage=run.stats?.hazard_damage_taken||0,seconds=run.stats?.seconds||0;
    if(replay||!hazardHintState||hazardHintState.key!==key||damage<hazardHintState.damage||seconds<hazardHintState.seconds){
      hazardHintState={key,damage,seconds,hits:0,shown:false,until:0};hazardCoaching.hidden=true;return;
    }
    const state=hazardHintState;
    if(damage>state.damage)state.hits++;
    state.damage=damage;state.seconds=seconds;
    if(!hintsEnabled||run.status!=='fighting'){hazardCoaching.hidden=true;return;}
    if(state.hits>=3&&!state.shown&&claimCombatHint()){
      state.shown=true;state.until=seconds+8;
      put(hazardCoaching,'Repeated hazard damage: leave the warning zone before it flashes. Jump only when the hazard is marked as jumpable. Guard reduces damage but does not prevent contact.');
    }
    hazardCoaching.hidden=!state.shown||seconds>=state.until;
  }
  function contextualMissHint(run,replay){
    const key=[run.id,run.level?.id,run.room].join(':'),misses=run.stats?.basic_misses||0,seconds=run.stats?.seconds||0;
    if(replay||!missHintState||missHintState.key!==key||misses<missHintState.misses||seconds<missHintState.seconds){
      missHintState={key,misses,seconds,baseline:misses,shown:false,until:0};missCoaching.hidden=true;return;
    }
    const state=missHintState;
    state.misses=misses;state.seconds=seconds;
    if(!hintsEnabled||run.status!=='fighting'){missCoaching.hidden=true;return;}
    if(misses-state.baseline>=3&&!state.shown&&claimCombatHint()){
      state.shown=true;state.until=seconds+8;
      put(missCoaching,'Repeated misses: move closer, line up with the enemy’s feet and face them before swinging. Solid cover blocks melee attacks.');
    }
    missCoaching.hidden=!state.shown||seconds>=state.until;
  }
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
  function getHealthThreshold(current,max){
    const ratio=max>0?(current/max):0;
    if(ratio>0.5)return{state:'healthy',label:'Healthy',symbol:'✓'};
    if(ratio>0.25)return{state:'wounded',label:'Wounded',symbol:'◆'};
    return{state:'critical',label:'Critical',symbol:'⚠'};
  }
  function meter(selector,label,current,max,threshold){const node=document.querySelector(selector);attr(node,'role','meter');attr(node,'aria-label',label);attr(node,'aria-valuemin',0);attr(node,'aria-valuemax',Math.max(1,max));attr(node,'aria-valuenow',Math.min(max,Math.max(0,current)));if(threshold){attr(node,'data-threshold',threshold.state);attr(node,'aria-valuetext',Math.max(0,current).toFixed(0)+' of '+Math.max(1,max).toFixed(0)+' HP, '+threshold.label);}}
  const transientTimers=new Map();
  function triggerTransientCounter(node,text,kind){
    if(!node)return;
    clearTimeout(transientTimers.get(node));
    node.textContent=text;
    if(kind)node.setAttribute('data-kind',kind);
    node.hidden=false;
    node.style.animation='none';
    void node.offsetHeight;
    node.style.animation='';
    const timer=setTimeout(()=>{
      node.hidden=true;
      node.textContent='';
    },1200);
    transientTimers.set(node,timer);
  }
  let lastResourceRunId='',lastObservedResource=null,lastObservedHp=null,lastObservedMana=null;
  function duration(seconds){seconds=Math.max(0,Math.floor(seconds));return Math.floor(seconds/60)+':'+String(seconds%60).padStart(2,'0');}
  function reason(skill,run,playing,stable=false){
    if(!['fighting','cleared'].includes(run.status))return 'Expedition ended';
    if(!playing||run.paused)return 'Paused';
    const remaining=run.skill_timers[skill.id]||0;
    if(remaining>0)return stable?'Cooling down':remaining.toFixed(1)+' seconds cooldown';
    if(run.player.mana<skill.cost)return stable?'More mana needed':Math.ceil(skill.cost-run.player.mana)+' more mana needed';
    return 'Ready · '+skill.cost+' mana';
  }
  let latencySamples=[];
  function updateLatency(ms){
    const node=$('rift-latency');
    if(!node)return;
    latencySamples.push(Math.max(0,Number(ms)||0));
    if(latencySamples.length>5)latencySamples.shift();
    const average=Math.round(latencySamples.reduce((sum,v)=>sum+v,0)/latencySamples.length);
    const state=average<=600?'normal':average<=1200?'elevated':'slow';
    put(node,average+' ms');
    attr(node,'data-state',state);
    attr(node,'aria-label','Connection latency: '+average+' ms ('+state+')');
  }
  let requestedRangeSkill=null,lastObservedRun=null;
  function updateSkillRangeSignal(run){
    const signal=$('rift-skill-range');
    if(!signal)return;
    const skill=requestedRangeSkill||(window.RiftDisplay?.skillRange&&run?.build?.skills?.[0]?run.build.skills[0]:null);
    if(!skill){
      put(signal,'Skill range: None requested');
      return;
    }
    const ref=skill.reference;
    if(!ref){
      put(signal,'Range: '+skill.name+' · Details unavailable');
      return;
    }
    let rangeText;
    if(ref.target==='self'){
      rangeText='Range: '+skill.name+' · Self ('+(ref.barrier?'Barrier':'Healing')+')';
    }else if(ref.target==='area'){
      rangeText='Range: '+skill.name+' · Area ('+ref.horizontal+'h × '+ref.depth+'d)';
    }else{
      const facing=(run?.player?.facing??1)<0?'←':'→';
      rangeText='Range: '+skill.name+' · Projectile (lane ±'+ref.depth+'d, facing '+facing+')';
    }
    put(signal,rangeText+' · '+window.RiftAbilities.aimHelp(skill,run));
  }
  function setRequestedRange(skill){
    requestedRangeSkill=skill;
    if(window.RiftRenderer?.setRangeSkill)window.RiftRenderer.setRangeSkill(skill);
    const rangeToggle=$('rift-range-toggle');
    if(rangeToggle)rangeToggle.setAttribute('aria-pressed',skill?'true':'false');
    updateSkillRangeSignal(lastObservedRun);
  }
  function getRequestedRange(){
    return requestedRangeSkill;
  }
  function detectPlayerAreaEffects(run){
    if(!run||!run.player||!['fighting','cleared'].includes(run.status))return null;
    if(run.status==='fighting'&&(run.skill_timers?.connection_grace||0)>0)return {kind:'connection',state:'evading',label:'Connection recovered: protected ('+run.skill_timers.connection_grace.toFixed(1)+'s)'};
    const p=run.player;
    const arena=run.practice?.arena||run.level?.rooms?.[run.room];
    if(run.status==='fighting'&&arena&&Array.isArray(arena.hazards)){
      for(let i=0;i<arena.hazards.length;i++){
        const h=arena.hazards[i];
        if(h.disabled)continue;
        const inside=p.x>=h.x&&p.x<=h.x+h.w&&p.y>=h.y&&p.y<=h.y+h.h;
        if(inside){
          const phase=(run.clock+h.offset)%h.period;
          const warning=phase<1.2;
          const active=phase>=1.2&&phase<1.2+h.duration;
          const kindName=h.kind.charAt(0).toUpperCase()+h.kind.slice(1);
          if(active){
            const evading=h.jumpable===true&&(p.jump||0)>0.1;
            return{
              kind:h.kind,
              state:evading?'evading':'active',
              name:kindName+' hazard',
              label:evading?'Area effect: '+kindName+' (Evading via jump)':'In area effect: '+kindName+' hazard (Active)',
            };
          }else if(warning){
            return{
              kind:h.kind,
              state:'warning',
              name:kindName+' hazard',
              label:'Area hazard warning: '+kindName+' zone',
            };
          }
        }
      }
    }
    for(const e of run.enemies||[]){
      if(run.status==='fighting'&&e.hp>0&&e.windup>0&&e.kind==='boss'){
        const dx=(p.x-e.target_x)/125,dy=(p.y-e.target_y)/62;
        if(dx*dx+dy*dy<=1){
          const attackName=e.attack_name||(e.art_key&&(e.attacks+1)%2===0?'Aimed Volley':'Ground Slam');
          const evading=(p.jump||0)>0.1;
          return{
            kind:'boss_area',
            state:evading?'evading':'active',
            name:'Boss '+attackName,
            label:evading?'Area effect: '+attackName+' (Evading)':'In area effect: '+attackName+' blast zone',
          };
        }
      }
    }
    if((run.skill_timers?.slowed||0)>0){
      return{
        kind:'slowed',
        state:'debuff',
        name:slowLabel(run),
        label:'Area effect: '+slowLabel(run)+' ('+run.skill_timers.slowed.toFixed(1)+'s)',
      };
    }
    return null;
  }
  function formatActivePlayTime(seconds){
    const s=Math.max(0,Number(seconds)||0);
    const m=Math.floor(s/60);
    const rem=(s%60).toFixed(1);
    return m>0?(m+'m '+rem+'s ('+s.toFixed(1)+'s)'):(s.toFixed(1)+'s');
  }
  function formatRoomSplits(splits){
    if(!splits||!Array.isArray(splits))return '';
    const parts=[];
    for(let i=0;i<splits.length;i++){
      if(splits[i]!=null&&Number.isFinite(splits[i])){
        parts.push('Tier '+(i+1)+': '+Number(splits[i]).toFixed(1)+'s');
      }
    }
    return parts.join(' · ');
  }
  function formatFamilyDefeats(monsterRecords){
    if(!monsterRecords||typeof monsterRecords!=='object')return '';
    const familyCounts={};
    for(const [key,record] of Object.entries(monsterRecords)){
      if(!record||!record.defeats||record.defeats<=0)continue;
      const fam=window.RiftBestiary?.monsterFamily(key)||(key.startsWith('monster:')?key.slice(8):key);
      if(fam)familyCounts[fam]=(familyCounts[fam]||0)+record.defeats;
    }
    const entries=Object.entries(familyCounts);
    if(!entries.length)return '';
    return entries.map(([fam,count])=>fam+': '+numbers.format(count)).join(' · ');
  }
  function formatCompletedObjectives(objectives){
    const entries=objectives?.entries;
    if(!entries||!Array.isArray(entries))return '';
    const completed=entries.filter(e=>e.status==='complete');
    if(!completed.length)return '';
    return completed.map(e=>e.name).join(' · ')+' ('+completed.length+' completed)';
  }
  function formatFailedObjectives(objectives){
    const entries=objectives?.entries;
    if(!entries||!Array.isArray(entries))return '';
    const failed=entries.filter(e=>e.status==='failed');
    if(!failed.length)return '';
    return failed.map(e=>e.name+(e.reason?' ('+e.reason+')':'')).join(' · ');
  }
  function formatCosmeticMilestones(run){
    if(!run)return '';
    const milestones=[];
    const completedLevels=run.completed_levels||[];
    if(completedLevels.length>=100)milestones.push('Campaign Complete (All 100 Missions) — Campaign Badge');
    else if(completedLevels.length>=10)milestones.push('First Region Complete — Region Badge');
    const pg=(run.past_expeditions?.perfect_guards||0)+(run.stats?.perfect_guards||0);
    if(pg>=1000)milestones.push('✦ Perfect guard III · 1,000 confirmed');
    else if(pg>=100)milestones.push('✦ Perfect guard II · 100 confirmed');
    else if(pg>=10)milestones.push('✦ Perfect guard I · 10 confirmed');
    return milestones.join(' · ');
  }
  function formatImprovedRecords(clear){
    if(!clear||!clear.records||!clear.records.length)return '';
    const labels={time:'Fastest clear time',health:'Best finish HP',hits:'Fewest damaging hits'};
    return clear.records.map(k=>labels[k]||k).join(' · ');
  }
  function updateLastEncounter(run){
    const container=$('rift-last-encounter');
    if(!container)return;
    const emptyNode=$('rift-last-encounter-empty');
    const detailsNode=$('rift-last-encounter-details');
    const badgeNode=$('rift-last-encounter-badge');
    const headlineNode=$('rift-last-encounter-headline');
    const statsNode=$('rift-last-encounter-stats');
    if(!emptyNode||!detailsNode||!badgeNode||!headlineNode||!statsNode)return;

    let encounter=run.last_encounter||(run.outcome?run:null);
    if(!encounter&&['cleared','defeated','complete','banked'].includes(run.status)){
      const stats=run.stats||{};
      const roomName=run.level?.rooms?.[run.room]?.name||('Tier '+((run.room||0)+1));
      const enemiesCount=(run.encounter_plan&&run.encounter_plan[run.room])?run.encounter_plan[run.room].length:(stats.kills||0);
      const boss=(run.enemies||[]).find(e=>e.kind==='boss');
      encounter={
        mission:run.level?.id||1,
        mission_name:run.level?.name||'Mossbound Ruins',
        room:run.room||0,
        room_name:roomName,
        outcome:run.status==='defeated'?'defeated':run.status==='complete'||run.status==='banked'?'completed':'cleared',
        seconds:(run.room_splits&&run.room_splits[run.room]!=null)?run.room_splits[run.room]:(stats.seconds||0),
        player_hp:run.player?.hp||0,
        player_max_hp:run.player?.max_hp||100,
        enemies:enemiesCount,
        boss_encounter:Boolean(boss),
        boss_name:boss?.name||'',
        damage_dealt:stats.damage_dealt||0,
        damage_taken:stats.damage_taken||0,
        hazard_damage_taken:stats.hazard_damage_taken||0,
        enemy_damage_taken:stats.enemy_damage_taken||0,
        hits_taken:stats.hits_taken||0,
        guard_blocked:stats.guard_blocked||0,
        barrier_blocked:stats.barrier_blocked||0,
        healing:stats.healing||0,
        gold_gained:run.gold||0,
        loot_items:(run.drops||[]).filter(d=>d.collected&&(d.gear||d.item)).length
      };
    }

    if(!encounter){
      emptyNode.hidden=false;
      detailsNode.hidden=true;
      put(badgeNode,'No encounter');
      attr(badgeNode,'data-outcome','none');
      return;
    }

    emptyNode.hidden=true;
    detailsNode.hidden=false;

    const monsterLinks=$('rift-result-monster-links'),monsterGroup=$('rift-result-monsters');
    if(monsterLinks&&monsterGroup){
      const monsters=[...new Set(encounter.monster_keys||[])].map(key=>[key,window.RiftBestiary?.monsterName(key)]).filter(([,name])=>name);
      const key=JSON.stringify(monsters);
      if(monsterLinks.dataset.roster!==key){
        monsterLinks.dataset.roster=key;monsterLinks.replaceChildren();
        for(const [id,name] of monsters){
          const button=document.createElement('button');button.type='button';button.textContent=name;
          button.setAttribute('aria-label','Review '+name+' in bestiary');
          button.onclick=()=>window.RiftBestiary.openMonster(id,button);monsterLinks.append(button);
        }
      }
      monsterGroup.hidden=monsters.length===0;
    }
    const bossGuide=$('rift-result-bestiary');
    if(bossGuide){
      const bossName=encounter.defeated_by_boss||encounter.boss_name;
      bossGuide.hidden=!bossName||!window.RiftBestiary?.hasBoss(bossName);
      bossGuide.onclick=()=>window.RiftBestiary.openBoss(bossName,bossGuide);
    }

    const isCleared=encounter.outcome==='cleared';
    const isDefeated=encounter.outcome==='defeated';
    const outcomeLabel=isCleared?(encounter.boss_encounter?'Boss defeated':'Room secured'):isDefeated?'Defeated':'Expedition complete';

    put(badgeNode,outcomeLabel);
    attr(badgeNode,'data-outcome',encounter.outcome);

    const tierLabel='Tier '+(encounter.room+1)+': '+encounter.room_name;
    const durationLabel=Number(encounter.seconds||0).toFixed(1)+'s';
    const threshold=getHealthThreshold(encounter.player_hp,encounter.player_max_hp);
    const hpLabel=Number(encounter.player_hp||0).toFixed(1)+'/'+Number(encounter.player_max_hp||0).toFixed(1)+' HP ('+threshold.symbol+' '+threshold.label+')';

    let headline='';
    if(isCleared){
      headline=encounter.mission_name+' · '+tierLabel+' secured in '+durationLabel+' combat. Finished at '+hpLabel+'; dealt '+numbers.format(encounter.damage_dealt)+' damage and took '+numbers.format(encounter.damage_taken)+' damage across '+(encounter.hits_taken||0)+' hits.';
    }else if(isDefeated){
      headline='Defeated in '+encounter.mission_name+' · '+tierLabel+' after '+durationLabel+' combat. Dealt '+numbers.format(encounter.damage_dealt)+' damage and took '+numbers.format(encounter.damage_taken)+' damage.';
    }else{
      headline=encounter.mission_name+' completed in '+durationLabel+' combat. Final HP: '+hpLabel+'.';
    }
    put(headlineNode,headline);

    const rows=[
      ['Outcome',outcomeLabel],
      ['Tier & location',encounter.mission_name+' · '+tierLabel],
      ['Combat duration',durationLabel],
      ['Ending health',hpLabel],
      ['Enemies defeated',numbers.format(encounter.enemies||0)+(encounter.boss_name?' (Boss: '+encounter.boss_name+')':'')],
      ['Damage dealt',numbers.format(encounter.damage_dealt||0)],
      ['Damage taken',numbers.format(encounter.damage_taken||0)+' ('+numbers.format(encounter.hits_taken||0)+' '+((encounter.hits_taken===1)?'hit':'hits')+')'],
      ['HP lost to hazards',numbers.format(encounter.hazard_damage_taken||0)],
      ['HP lost to enemies',numbers.format(encounter.enemy_damage_taken||0)],
      ['Guarded damage',numbers.format(encounter.guard_blocked||0)]
    ];
    if((encounter.barrier_blocked||0)>0)rows.push(['Barrier absorbed',numbers.format(encounter.barrier_blocked)]);
    if((encounter.armor_piercing_damage||0)>0)rows.push(['Armor-piercing damage',numbers.format(encounter.armor_piercing_damage)]);
    if((encounter.highest_attack_chain||0)>0)rows.push(['Longest attack chain',numbers.format(encounter.highest_attack_chain)]);
    if((encounter.combo_score||0)>0)rows.push(['Combo score',numbers.format(encounter.combo_score)]);
    if((encounter.guard_breaks||0)>0)rows.push(['Guard breaks suffered',numbers.format(encounter.guard_breaks)]);
    if((encounter.heavy_attacks||0)>0)rows.push(['Heavy basic attacks',numbers.format(encounter.heavy_attacks)]);
    if((encounter.aerial_attacks||0)>0)rows.push(['Aerial basic attacks',numbers.format(encounter.aerial_attacks)]);
    if((encounter.sweep_attacks||0)>0)rows.push(['Grounded sweep attacks',numbers.format(encounter.sweep_attacks)]);
    if((encounter.launchers||0)>0)rows.push(['Launchers against small enemies',numbers.format(encounter.launchers)]);
    if((encounter.downed_followups||0)>0)rows.push(['Downed enemy follow-ups',numbers.format(encounter.downed_followups)]);
    if((encounter.pack_attacks||0)>0)rows.push(['Coordinated pack attacks',numbers.format(encounter.pack_attacks)]);
    if((encounter.summon_punishes||0)>0)rows.push(['Summon arrival punishes',numbers.format(encounter.summon_punishes)]);
    if((encounter.flank_attempts||0)>0)rows.push(['Enemy flank maneuvers',numbers.format(encounter.flank_attempts)]);
    if((encounter.rear_strikes||0)>0)rows.push(['Unshielded rear strikes',numbers.format(encounter.rear_strikes)]);

    const unclassified=Math.max(0,(encounter.damage_taken||0)-(encounter.hazard_damage_taken||0)-(encounter.enemy_damage_taken||0));
    if(unclassified>.001)rows.push(['Damage without source records',numbers.format(unclassified)]);
    if(isDefeated){
      for(const boss of encounter.bosses||[]){
        const percent=boss.max_hp>0?Math.round(boss.hp/boss.max_hp*100):0;
        rows.push(['Boss: '+(boss.name||'Unknown boss'),numbers.format(boss.hp)+' / '+numbers.format(boss.max_hp)+' HP ('+percent+'%) · Phase '+boss.phase]);
      }
    }

    if(hintsEnabled&&isDefeated&&encounter.defeated_by_hazard)rows.push(['Hazard counterplay',hazardDefeatHint(encounter.defeated_by_hazard)]);
    if(isDefeated&&encounter.defeated_by_boss){
      rows.push(['Defeated by boss',encounter.defeated_by_boss]);
    }
    if(isDefeated&&encounter.defeated_by_enemy&&!encounter.defeated_by_boss){
      rows.push(['Defeated by enemy',encounter.defeated_by_enemy]);
    }
    if(isDefeated){
      const finalHit=encounter.defeated_by_hazard?('Hazard: '+encounter.defeated_by_hazard.kind):encounter.defeated_by_boss?('Boss: '+encounter.defeated_by_boss):encounter.defeated_by_enemy?('Enemy: '+encounter.defeated_by_enemy):'';
      if(finalHit)rows.push(['Final hit source',finalHit]);
      if(encounter.defeat_cause)rows.push(['Cause of defeat',encounter.defeat_cause]);
    }

    if((encounter.treasure_escaped||0)>0){
      rows.push(['Treasure goblins escaped',numbers.format(encounter.treasure_escaped)+' · No loot or defeat credit']);
    }

    if((encounter.healing||0)>0){
      rows.push(['Healing received',numbers.format(encounter.healing)]);
    }

    if(isDefeated){
      const bankedGold=encounter.banked_gold||0;
      const bankedItems=encounter.banked_items_count||0;
      rows.push(['Banked rewards kept',numbers.format(bankedGold)+' gold · '+numbers.format(bankedItems)+' '+((bankedItems===1)?'item':'items')]);
    }

    if((encounter.gold_gained||0)>0||(encounter.loot_items||0)>0){
      const lootText=numbers.format(encounter.gold_gained||0)+' gold'+((encounter.loot_items||0)>0?' · '+numbers.format(encounter.loot_items)+' '+((encounter.loot_items===1)?'item':'items'):'');
      rows.push([isDefeated?'Unbanked loot lost':'Loot collected',lootText]);
    }

    const famText=formatFamilyDefeats(run?.monster_records||encounter?.monster_records);
    if(famText)rows.push(['Enemies defeated by family',famText]);
    const splitsText=formatRoomSplits(run?.room_splits||encounter?.room_splits);
    if(splitsText)rows.push(['Room-by-room splits',splitsText]);
    const compObj=formatCompletedObjectives(run?.objectives||encounter?.objectives);
    if(compObj)rows.push(['Completed optional objectives',compObj]);
    const failObj=formatFailedObjectives(run?.objectives||encounter?.objectives);
    if(failObj)rows.push(['Failed optional objectives',failObj]);
    const cosmeticText=formatCosmeticMilestones(run||encounter);
    if(cosmeticText)rows.push(['Cosmetic milestones',cosmeticText]);
    const improvedText=formatImprovedRecords(run?.last_clear||encounter?.last_clear);
    if(improvedText)rows.push(['Improved personal records',improvedText]);

    const key=JSON.stringify({encounter,rows});
    if(statsNode.dataset.lastEncounterKey!==key){
      statsNode.dataset.lastEncounterKey=key;
      statsNode.replaceChildren();
      for(const [dtText,ddText] of rows){
        const dt=document.createElement('dt');
        const dd=document.createElement('dd');
        dt.textContent=dtText;
        dd.textContent=ddText;
        statsNode.append(dt,dd);
      }
    }
    syncDetailedStatsVisibility();
  }
  let terrainCoachingState=null;
  function terrainCoachingAllowed(run,kind){
    if(run.status!=='fighting')return true;
    const seconds=run.stats?.seconds||0,key=[run.level?.id,run.room].join(':');
    if(!terrainCoachingState||terrainCoachingState.id!==run.id||seconds<terrainCoachingState.seconds)terrainCoachingState={id:run.id,seconds,next:seconds,room:key,seen:new Set(),kind:'',until:0};
    const state=terrainCoachingState;state.seconds=seconds;
    if(state.room!==key){state.room=key;state.seen.clear();state.kind='';}
    if(!state.seen.has(kind)&&seconds>=state.next){
      state.seen.add(kind);state.kind=kind;state.until=seconds+8;state.next=seconds+20;
    }
    return state.kind===kind&&seconds<state.until;
  }
  function nearbyCover(run){
    const arena=run.level?.rooms?.[run.room];if(!arena)return null;
    const candidates=[...(arena.drop_edges||[]).map(edge=>({kind:'ledge',obstacle:{x:edge.x,y:edge.y,w:edge.w,h:edge.landing_y-edge.y}})),...(arena.obstacles||[]).map(obstacle=>({kind:'low',obstacle})),...(arena.high_cover||[]).map(obstacle=>({kind:'tall',obstacle})),...(arena.cover||[]).filter(c=>c.material==='stone'||c.hp>0).map(obstacle=>({kind:obstacle.material,obstacle}))];
    let nearest=null,distance=96;
    for(const candidate of candidates){const o=candidate.obstacle,x=Math.max(o.x,Math.min(run.player.x,o.x+o.w)),y=Math.max(o.y,Math.min(run.player.y,o.y+o.h)),d=Math.hypot(x-run.player.x,y-run.player.y);if(d<distance){distance=d;nearest=candidate;}}
    return nearest;
  }
  function bossPracticeLesson(run){
    const node=$('rift-boss-practice-lesson');if(!node)return;
    node.hidden=!hintsEnabled||run.practice?.mode!=='boss';if(node.hidden)return;
    const boss=run.enemies.find(enemy=>enemy.kind==='boss'&&enemy.hp>0);
    let lesson;
    if(run.practice.completed)lesson='Boss defeated. Reset and try another phase, or turn off longer warnings to practice normal timing.';
    else if(!boss)lesson='Choose a boss and starting phase to practice recognizing its warnings.';
    else if(boss.windup>0){
      const volley=boss.art_key&&(boss.attacks+1)%2===0;
      lesson=(boss.attack_name||(volley?'Aimed Volley':'Ground Slam'))+': '+(volley?'A projectile warning. Move out of the firing lane, or face incoming shots and guard. Keep watching shots after the warning ends.':'A ground-attack warning. Move clear of the marked area, or time your jump near impact. Jumping too early can leave you grounded when the attack lands.');
    }else if(boss.cooldown>0)lesson='Recovery: look for an opening to attack or reposition. Projectiles already in flight can still hit you.';
    else lesson='Watch the boss and its ground markings. Identify a ground attack or projectile warning before choosing your defense.';
    if(run.practice.slow_telegraphs)lesson+=' Longer warnings double preparation time; projectile travel and recovery keep their normal speed.';
    put(node,lesson);
  }
  function update(run,playing,replay=false){
    bossPracticeLesson(run);
    window.RiftMinimap.update(run);
    const cover=nearbyCover(run),hint=$('rift-terrain-hint');hint.hidden=!hintsEnabled||!cover||!['fighting','cleared'].includes(run.status)||!terrainCoachingAllowed(run,cover.kind);
    if(cover){hint.dataset.kind=cover.kind;put(hint,cover.kind==='ledge'?'One-way ledge · Move down to drop safely · Return around either end':cover.kind==='low'?'Low cover · Move + '+(window.RiftControls?.label('jump')||'Space')+' to vault · Projectiles pass over':cover.kind==='wood'?'Wooden barricade · Break with attacks · Blocks projectiles':cover.kind==='stone'?'Stone cover · Walk around · Blocks projectiles':'Tall cover · Walk around · Blocks projectiles');}

    lastObservedRun=run;
    updateSkillRangeSignal(run);
    const runIdentity=[run.id,run.level?.id,run.room].join(':');
    const isLive=playing&&['fighting','cleared'].includes(run.status)&&!run.paused;
    const currentResource=run.resource||0;
    const currentHp=run.player?.hp||0;
    const currentMana=run.player?.mana||0;

    if(runIdentity!==lastResourceRunId||replay||!isLive){
      lastResourceRunId=runIdentity;
      lastObservedResource=currentResource;
      lastObservedHp=currentHp;
      lastObservedMana=currentMana;
    }else{
      if(lastObservedResource!==null&&currentResource>lastObservedResource){
        const diff=currentResource-lastObservedResource;
        const resName=run.build?.resource||'Charge';
        triggerTransientCounter($('rift-resource-gain'),'+'+diff+' '+resName,'resource');
      }
      lastObservedResource=currentResource;

      if(lastObservedHp!==null&&currentHp>lastObservedHp+0.5){
        const diff=Math.round(currentHp-lastObservedHp);
        triggerTransientCounter($('rift-hp-gain'),'+'+diff+' HP','health');
      }
      lastObservedHp=currentHp;

      if(lastObservedMana!==null&&currentMana>lastObservedMana+4.5){
        const diff=Math.round(currentMana-lastObservedMana);
        triggerTransientCounter($('rift-mana-gain'),'+'+diff+' MP','mana');
      }
      lastObservedMana=currentMana;
    }
    updateCombatHintBudget(run);contextualHazardHint(run,replay);contextualMissHint(run,replay);recentDamage(run,replay);window.RiftOnboarding.update(run);window.RiftRecords.update(run);window.RiftClassChallenges.update(run);updateLastEncounter(run);
    const clear=run.last_clear;const clearNode=$('rift-clear-result');clearNode.hidden=!clear;
    if(clear){const record=run.mission_history?.[clear.mission]||{},labels={time:'clear time '+Number(record.best_seconds||0).toFixed(1)+'s',health:'finish HP '+Number(record.best_finish_hp||0).toFixed(1)+'/'+Number(record.best_finish_max_hp||0).toFixed(1),hits:'fewest damaging hits '+(record.fewest_hits??0)};put(clearNode,'Mission '+clear.mission+' · '+(clear.first?'First clear!':'Repeat clear.')+(clear.records.length?' New personal records: '+clear.records.map(key=>labels[key]).join(' · '):' No personal records improved.'));}
    const living=run.enemies.filter(e=>e.hp>0),stats=run.stats||{},boss=living.find(e=>e.kind==='boss');
    $('rift-paused-badge').hidden=!['fighting','cleared'].includes(run.status)||(playing&&!run.paused);
    put($('rift-paused-badge'),run.paused?'Paused':'Not running');
    const catchupNode=$('rift-catchup');
    if(catchupNode){
      const isCatchingUp=Boolean(run?.catchup);
      catchupNode.hidden=!isCatchingUp||!['fighting','cleared'].includes(run.status);
    }
    const savedNode=$('rift-saved-at');
    if(savedNode){
      const ms=run.saved_at_ms;
      if(ms&&ms>0){
        const d=new Date(ms),h=d.getHours(),m=d.getMinutes(),s=d.getSeconds();
        const stamp=String(h).padStart(2,'0')+':'+String(m).padStart(2,'0')+':'+String(s).padStart(2,'0');
        put(savedNode,'Save: '+stamp);
        attr(savedNode,'aria-label','Last confirmed save time: '+d.toLocaleString());
      }else{
        put(savedNode,'Save: --');
        attr(savedNode,'aria-label','Last confirmed save time: unavailable');
      }
    }
    put($('rift-facing'),run.player.facing<0?'← Facing left':'Facing right →');
    const guardNode=$('rift-guard-reduction'),staminaNode=$('rift-guard-stamina');
    const guardStamina=Math.round(run.player.guard_stamina??100);
    const guardBroken=(run.skill_timers?.guard_break_recovery||0)>0;
    put($('rift-guard-state'),['fighting','cleared'].includes(run.status)?guardBroken?'Guard broken. Wait for recovery.':'Guard available.':'');
    if(staminaNode){
      put(staminaNode,guardBroken?'Stamina: Broken ('+run.skill_timers.guard_break_recovery.toFixed(1)+'s)':'Stamina: '+guardStamina+'%');
      staminaNode.dataset.stamina=String(guardStamina);
      if(guardBroken)staminaNode.dataset.broken='true';else delete staminaNode.dataset.broken;
    }
    put(guardNode,guardBroken?'Guard broken ('+run.skill_timers.guard_break_recovery.toFixed(1)+'s)':run.player.guard&&['fighting','cleared'].includes(run.status)?(playing&&!run.paused?'Guard: 82% frontal reduction (Stamina '+guardStamina+'%)':'Guard paused (Stamina '+guardStamina+'%)'):'Guard inactive');
    const recentGuard=(run.events||[]).some(e=>(e.kind==='block'||e.kind==='perfect_guard')&&(run.counter-e.id)<6);
    if(recentGuard)guardNode.dataset.guardedHit='';else delete guardNode.dataset.guardedHit;
    const guardBtn=document.querySelector('.rift-basics button[data-bind="guard"]');
    if(guardBtn){
      if(recentGuard)guardBtn.dataset.guardedHit='';else delete guardBtn.dataset.guardedHit;
      if(guardBroken){
        guardBtn.setAttribute('aria-disabled','true');
        guardBtn.dataset.broken='true';
      }else{
        guardBtn.removeAttribute('aria-disabled');
        delete guardBtn.dataset.broken;
      }
    }
    put($('rift-enemy-count'),living.length+' '+(living.length===1?'enemy':'enemies')+' remaining');
    const cam=Math.max(0,Math.min(640,run.player.x-350));
    const offLeft=living.filter(e=>e.x<cam),offRight=living.filter(e=>e.x>cam+960),offTotal=offLeft.length+offRight.length;
    let offText='No off-screen threats';
    if(run.status==='fighting'&&offTotal>0){
      const parts=[];
      if(offLeft.length)parts.push('← '+offLeft.length+(offLeft.some(e=>e.kind==='boss')?' (Boss)':''));
      if(offRight.length)parts.push(offRight.length+(offRight.some(e=>e.kind==='boss')?' (Boss)':'')+' →');
      offText='Off-screen threats: '+parts.join(' · ');
    }
    put($('rift-offscreen-enemies'),offText);
    put($('rift-combat-time'),duration(stats.seconds||0)+' combat');
    const continues=run.room<2||($('rift-auto').checked&&run.level?.id<100),health=Math.min(run.player.max_hp,run.player.hp+run.player.max_hp*.25);
    put($('rift-recovery-preview'),continues?'Continue: +'+Math.max(0,health-run.player.hp).toFixed(1)+' HP → '+health.toFixed(1)+'/'+run.player.max_hp.toFixed(1)+' HP; mana refills to 100. Leaving gives no recovery.':'Final checkpoint: bank your rewards and finish. No next-tier recovery.');
    put($('rift-objective'),run.status==='fighting'?(run.room_objective&&!run.room_objective.complete?(run.room_objective.kind==='split_defense'?'Defend both lane wards and defeat the patrol':run.room_objective.kind==='rune_gate'?'Clear the patrol and open the rune gate':run.room_objective.kind==='protect_lantern'?'Protect the lantern and defeat the patrol':run.room_objective.kind==='rescue_companions'?'Rescue the companions and defeat the patrol':run.room_objective.kind==='linked_guardians'?'Defeat the linked guardians and patrol':run.room_objective.kind==='escape_collapse'?'Reach the exit before the collapse':run.room_objective.kind==='interrupt_ritual'?'Interrupt the ritual and defeat the patrol':run.room_objective.kind==='escort_spirit'?'Escort the spirit and defeat the patrol':run.room_objective.kind==='moving_beacons'?'Capture the moving beacons and defeat the patrol':run.room_objective.kind==='marked_hunt'?'Defeat the marked hunt targets':run.room_objective.kind==='disable_generators'?'Disable the generators and defeat the patrol':run.room_objective.kind==='carry_relic'?'Deliver the relic and defeat the patrol':run.room_objective.kind==='destroy_totems'?'Destroy the totems and defeat the patrol':run.room_objective.kind==='survive_waves'?'Survive all three waves':run.room_objective.kind==='hold_circle'?'Charge the circle and defeat the patrol':'Gather the sigils and defeat the patrol'):boss?'Defeat '+boss.name+' and its defenders':'Clear the enemy patrol'):run.status==='cleared'?'Room secured · Loot ready to bank':run.status==='defeated'?'Expedition ended':'Rewards secured');
    put($('rift-jump-ready'),(run.skill_timers.jump||0)>0?'Jump '+run.skill_timers.jump.toFixed(1)+'s':'Jump ready');
    const dodgeNode=$('rift-dodge-ready');
    if(dodgeNode)put(dodgeNode,(run.skill_timers.dodge_cooldown||0)>0?'Dodge '+run.skill_timers.dodge_cooldown.toFixed(1)+'s':'Dodge ready');
    put($('rift-combo-step'),'Strike '+(run.combo||0)+'/3');
    put($('rift-barrier-state'),run.barrier>0?'Barrier '+numbers.format(run.barrier):'No barrier');
    const rules=run.build.sequence||'Use equipped class builders and finishers to activate class effects.';
    put($('rift-class-rules'),window.RiftRecords.classIdentity(run.build)+': '+rules+(run.build.class==='berserker'?' Passive Fury: +15% damage while alive at 30% HP or below; healing above that threshold turns it off.':''));
    const temporary=[];
    if(run.barrier>0)temporary.push('Protection capacity: '+numbers.format(run.barrier)+' barrier remaining — consumed by hits, no timer.');
    if(run.skill_timers.connection_grace>0)temporary.push('Timed protection: connection recovery '+run.skill_timers.connection_grace.toFixed(1)+'s remaining.');
    if(run.player.guard&&run.skill_timers.perfect_guard>0)temporary.push('Timed guard window: '+run.skill_timers.perfect_guard.toFixed(2)+'s remaining; requires facing the incoming attack.');
    if(run.skill_timers.slowed>0)temporary.push('Timed movement penalty: '+slowLabel(run)+' · '+run.skill_timers.slowed.toFixed(1)+'s remaining.');
    if(!temporary.length)temporary.push('No temporary protection or movement penalties active.');
    const effectsList=$('rift-temporary-effects'),effectsKey=JSON.stringify(temporary);
    if(effectsList.dataset.effects!==effectsKey){effectsList.dataset.effects=effectsKey;effectsList.replaceChildren();for(const label of temporary){const item=document.createElement('li');item.textContent=label;effectsList.append(item);}}

    put($('rift-slow-state'),run.skill_timers.slowed>0?slowLabel(run)+' '+run.skill_timers.slowed.toFixed(1)+'s':'Normal speed');
    const areaNode=$('rift-area-effects');
    if(areaNode){
      const effect=detectPlayerAreaEffects(run);
      if(effect){
        put(areaNode,effect.label);
        attr(areaNode,'data-effect-state',effect.state);
        attr(areaNode,'data-effect-kind',effect.kind);
      }else{
        put(areaNode,'No active area effects');
        attr(areaNode,'data-effect-state','none');
        attr(areaNode,'data-effect-kind','none');
      }
    }
    const marked=living.find(e=>e.id===run.marked);put($('rift-mark-state'),marked?'Marked: '+marked.name:run.last_mark_end?'Mark ended: '+run.last_mark_end.target_name+' '+(run.last_mark_end.reason==='defeated'?'was defeated':'escaped')+'. Land a builder hit to mark a new target.':'No marked target');
    put($('rift-ultimate-state'),run.build.ultimate?'Ultimate: '+run.build.ultimate.name+' · '+reason(run.build.ultimate,run,playing):'No ultimate in this expedition');
    const finisher=run.build.signatures?.find(s=>s.role==='finisher');
    const oracle=$('rift-oracle-healing');oracle.hidden=run.build.class!=='oracle';if(!oracle.hidden)put(oracle,window.RiftAbilities.oracleHealing(run));
    const reaction=$('rift-elemental-reaction');reaction.hidden=run.build.class!=='elementalist';
    if(!reaction.hidden)put(reaction,marked&&run.resource>0?'Reaction armed: a charged finisher hitting '+marked.name+' deals ×1.2 damage before defenses.':'Reaction setup: mark an enemy with your builder, then hit it with a charged finisher.');
    const pack=$('rift-pack-target');pack.hidden=run.build.class!=='beastmaster';
    const pets=$('rift-pet-synergy');pets.hidden=run.build.class!=='beastmaster';
    if(!pets.hidden){
      const count=run.build.pets||0,counted=Math.min(3,count),bonus=(1+counted*.1).toFixed(1),state=counted===0?'none':!finisher?'missing':run.resource>0?'armed':'building';attr(pets,'data-state',state);
      put(pets,'Pets in this expedition: '+count+' · '+counted+'/3 count toward the bonus. '+(state==='none'?'No pet damage bonus.':state==='missing'?'Equip a class finisher in Abyss to use the pet bonus.':finisher.name+': charged hits deal ×'+bonus+' damage before defenses. '+(state==='armed'?'Finisher: '+reason(finisher,run,playing)+'.':'Build at least one charge to activate this bonus.'))+' Builders and empty finishers get no pet multiplier. Additional pets beyond three add no further bonus. Pet equipment changes apply to a new expedition.');
    }
    if(!pack.hidden)put(pack,marked?'Pack target: '+marked.name+'. Face and line up your shot; it hits the first enemy in that lane.':'Pack target: none. Land a builder hit to direct the pack.');
    const precision=$('rift-precision-state');precision.hidden=run.build.class!=='marksman';
    if(!precision.hidden)put(precision,!marked?'Precision: land a builder hit to mark a target.':!(run.resource>0)?'Precision target: '+marked.name+'. Build charges before firing your finisher.':!finisher?'Precision target: '+marked.name+'. Equip a finisher in Abyss.':'Precision armed on '+marked.name+': next charged finisher gains +60 percentage points of armor piercing on this target (total capped at 100%). Finisher: '+reason(finisher,run,playing,true)+'.');
    put($('rift-finisher-state'),!finisher?'Class abilities unlock in Abyss':run.resource>0?'Finisher: '+reason(finisher,run,playing)+' · '+run.resource+' charges':'Build charges with '+window.RiftControls.label('signature0'));
    const blood=run.last_blood_recovery,bloodNode=$('rift-blood-recovery');bloodNode.hidden=run.build.class!=='bloodblade'||!blood;
    if(!bloodNode.hidden)put(bloodNode,'Last blood recovery · '+blood.skill_name+': +'+blood.healed.toFixed(1)+' HP; '+blood.overflow.toFixed(1)+' HP overflow discarded.');
    const receipt=run.last_cooldown_receipt,receiptNode=$('rift-cooldown-receipt');receiptNode.hidden=run.build.class!=='chronomancer'||!receipt;
    if(!receiptNode.hidden)put(receiptNode,'Last rewind · '+receipt.source+': '+(receipt.recovered.length?receipt.recovered.map(r=>r.name+' −'+Number(r.seconds.toFixed(2))+'s').join(', '):'no active cooldowns to reduce')+'.');
    const spent=run.last_charge_spend,spentNode=$('rift-last-charge-spend');spentNode.hidden=!spent;
    if(spent)put(spentNode,'Last charge spend: '+spent.skill_name+' consumed '+spent.charges+' charge'+(spent.charges===1?'':'s')+'.');
    put($('rift-charge-benefit'),window.RiftAbilities.chargeBenefits(run));
    $('rift-health-cost').hidden=run.build.class!=='voidwalker'||!finisher;
    if(!$('rift-health-cost').hidden)put($('rift-health-cost'),'Finisher health cost: '+window.RiftAbilities.healthCost(run,finisher).toFixed(1)+' HP'+(run.resource>0?' · leaves at least 1 HP':' · no charges to spend'));
    const costWarning=$('rift-health-cost-warning');costWarning.hidden=run.build.class!=='voidwalker'||!finisher||!(run.resource>0)||run.player.hp<=0;
    if(!costWarning.hidden){const after=run.player.hp-window.RiftAbilities.healthCost(run,finisher),low=after<=run.player.max_hp*.25;attr(costWarning,'data-low',String(low));put(costWarning,(low?'Low-health cast: ':'Charged finisher: ')+run.player.hp.toFixed(1)+' → '+after.toFixed(1)+' HP. Self-cost stops at 1 HP; incoming damage can still defeat you.');}
    const builder=run.build.signatures?.find(s=>s.role==='builder');
    const lowResource=$('rift-low-resource');lowResource.hidden=run.status!=='fighting'||run.player.hp<=0||(run.resource||0)>=2;
    if(!lowResource.hidden){
      const tips={vanguard:'A frontal perfect guard can also add one charge per guard raise.',berserker:'Charged finishers gain execution damage against targets at half health or less.',marksman:'Land the builder on your intended target to set up precision piercing.',beastmaster:'Mark your intended target before the pack attack; its bonus uses your equipped pets.',elementalist:'Land the builder, then hit the marked target with a charged finisher for a reaction.',chronomancer:'Spend charges while other ability or jump cooldowns are running to benefit from rewind.',oracle:'Your healing builder still builds '+(run.build.resource||'class charges')+' at full health.',geomancer:'Charged finishers add armor piercing against tough targets.',bloodblade:'Your charged finisher restores health for each charge spent.',voidwalker:'Spending charges costs health; check the health-cost preview before casting.',runesmith:'Spending charges grants a barrier, even without a relic.',alchemist:'Spending '+(run.build.resource||'class')+' charges restores health; a marked target also takes extra armor piercing.'};
      const prefix=(run.build.resource||'Class charges')+': '+(run.resource||0)+'/3. ';
      if(!builder||!finisher)put(lowResource,prefix+'Unlock and equip your class builder and finisher in Abyss.');
      else{const key=window.RiftControls.label('signature'+run.build.signatures.indexOf(builder));put(lowResource,prefix+'Build with '+builder.name+' ('+key+'). Builder: '+reason(builder,run,playing)+'. '+(run.resource>0?'One charge already enables a charged finisher; build up to three if you want to spend more. ':'Build at least one charge before spending it. ')+(tips[run.build.class]||''));}
    }
    const mixture=$('rift-alchemist-sequence');mixture.hidden=run.build.class!=='alchemist';
    if(!mixture.hidden){
      if(!builder||!finisher)put(mixture,(run.build.resource||'Class resource')+' sequence: equip both your class builder and finisher in Abyss.');
      else{const signatures=run.build.signatures,key=s=>window.RiftControls.label('signature'+signatures.indexOf(s)),charges=run.resource||0,buildStep=builder.name+' ('+key(builder)+')',finishStep=finisher.name+' ('+key(finisher)+')';
        const next=charges===0?'Start with '+buildStep+'.':charges<3?'Build again with '+buildStep+' or spend with '+finishStep+'.':(run.build.resource||'Class resource')+' full: spend with '+finishStep+'.';
        const target=marked?'Marked target: '+marked.name+'. Charged burst adds 35 percentage points of armor piercing against it (100% total cap).':'Land a builder hit to mark a target for the charged burst’s piercing bonus.';
        put(mixture,(run.build.resource||'Class resource')+' '+charges+'/3 · '+next+' '+(charges>0?'Spending now restores up to '+Number((charges*3).toFixed(0))+'% maximum HP, capped at full health. ':'')+target);
      }
    }
    coaching.hidden=!hintsEnabled||coachingDismissed||!builder||!finisher||(stats.empty_finishers||0)<3||!classCoachingAllowed(run);
    if(!coaching.hidden)put(coaching.querySelector('p'),'Three or more finishers used no charges. Practice '+builder.name+' ('+window.RiftControls.label('signature0')+') before '+finisher.name+' ('+window.RiftControls.label('signature1')+'). Build up to three charges, then spend them with your finisher.');
    const relic=$('rift-relic-synergy');relic.hidden=run.build.class!=='runesmith';
    if(!relic.hidden){const state=!run.build.relic?'missing':run.resource>0&&finisher?'armed':'equipped';attr(relic,'data-state',state);put(relic,(state==='missing'?'Relic synergy unavailable: no relic in this expedition build.':state==='equipped'?'Relic equipped: build charges and use a finisher for ×1.15 damage.':'Relic synergy armed: charged finisher damage ×1.15 before defenses. Finisher: '+reason(finisher,run,playing,true)+'.')+' The class barrier works with or without a relic.');}
    const furyNode=$('rift-berserker-fury'),fury=run.player.hp>0&&run.player.max_hp>0&&run.player.hp<=run.player.max_hp*.3;
    furyNode.hidden=run.build.class!=='berserker';attr(furyNode,'data-active',String(fury&&!furyNode.hidden));
    if(!furyNode.hidden)put(furyNode,fury?'Fury active: +15% damage at impact while at 30% HP or below.':'Fury inactive: +15% damage while alive at 30% HP or below.');
    const playerThreshold=getHealthThreshold(run.player.hp,run.player.max_hp);
    const hpThresholdNode=$('rift-hp-threshold');
    if(hpThresholdNode){
      put(hpThresholdNode,playerThreshold.symbol+' '+playerThreshold.label);
      attr(hpThresholdNode,'data-threshold',playerThreshold.state);
    }
    meter('#rift-vitals .hp','Player health',run.player.hp,run.player.max_hp,playerThreshold);
    meter('#rift-vitals .mana','Player mana',run.player.mana,100);
    if(boss){
      const bossThreshold=getHealthThreshold(boss.hp,boss.max_hp);
      const bossThresholdNode=$('rift-boss-threshold');
      if(bossThresholdNode){
        put(bossThresholdNode,bossThreshold.symbol+' '+bossThreshold.label);
        attr(bossThresholdNode,'data-threshold',bossThreshold.state);
      }
      if($('rift-boss-name'))$('rift-boss-name').title=boss.name;
      meter('#rift-boss .hp',boss.name+' health',boss.hp,boss.max_hp,bossThreshold);
      const phase=Math.max(1,Math.min(3,boss.phase||1));
      put($('rift-boss-phase'),'Phase '+phase+(phase===1?' · Next: phase 2 at 50% HP':phase===2?' · Next: phase 3 at 25% HP':' · Final phase'));
      const bossAttackNode=$('rift-boss-attack');
      if(bossAttackNode){
        if(boss.windup>0){
          const attack=boss.attack_name||(boss.art_key&&(boss.attacks+1)%2===0?'Aimed Volley':'Ground Slam');
          const volley=boss.art_key&&(boss.attacks+1)%2===0;
          const defense=volley?'Move to evade; face the shot to guard':'Jump or move clear';
          put(bossAttackNode,'⚡ '+attack+' · '+boss.windup.toFixed(1)+'s windup · '+defense);
          bossAttackNode.hidden=false;
          if(playing&&!run.paused){
            const attackKey=JSON.stringify([run.id,run.room,boss.id,boss.attacks||0,attack]);
            if(lastAnnouncedBossAttack!==attackKey){
              lastAnnouncedBossAttack=attackKey;
              put($('rift-announcer'),'Boss preparing '+attack+' · '+boss.name+' · '+Math.ceil(boss.windup)+'s windup. '+defense+'.');
            }
          }
        }else if(boss.weak_point>0){
          put(bossAttackNode,'Weak point · +25% damage · '+boss.weak_point.toFixed(1)+'s');
          bossAttackNode.hidden=false;
          lastAnnouncedBossAttack='';
        }else{
          bossAttackNode.hidden=true;
          lastAnnouncedBossAttack='';
        }
      }
    }else{
      if($('rift-boss-attack'))$('rift-boss-attack').hidden=true;
      lastAnnouncedBossAttack='';
    }
    // All cooldown labels and rings share this authoritative snapshot clock.
    // Per-button wall-clock timers would drift during pause or delayed responses.
    for(const [id,skills] of [['rift-skills',run.build.skills],['rift-signatures',[...(run.build.signatures||[]),...(run.build.ultimate?[run.build.ultimate]:[])]]]){
      [...$(id).children].forEach((button,index)=>{const skill=skills[index];if(!skill)return;const why=reason(skill,run,playing);window.RiftAbilities.update(button,skill,run,why,skill===run.build.ultimate);});
    }
    const values=[['Current mission clear streak',run.clear_streak],['Best mission clear streak',run.best_clear_streak],['Elapsed active play time',formatActivePlayTime(stats.seconds)],['Paused seconds (completed pauses)',stats.paused_seconds],['Enemies defeated',stats.kills],['Bosses defeated',stats.bosses]];
    if((stats.treasure_goblins||0)>0)values.push(['Treasure goblins defeated',stats.treasure_goblins]);
    values.push(['Rooms cleared',stats.rooms_cleared],['Damage dealt',stats.damage_dealt],['Damage taken',stats.damage_taken],['HP lost to hazards',stats.hazard_damage_taken],['HP lost to enemies',stats.enemy_damage_taken],['Damaging hits taken',stats.hits_taken],['Healing received',stats.healing],['Guard prevented',stats.guard_blocked]);
    if((stats.barrier_blocked||0)>0)values.push(['Barrier prevented',stats.barrier_blocked]);
    if((stats.armor_blocked||0)>0)values.push(['Armor prevented',stats.armor_blocked]);
    if((stats.armor_piercing_damage||0)>0)values.push(['Armor-piercing damage',stats.armor_piercing_damage]);
    if((stats.highest_attack_chain||0)>0)values.push(['Longest uninterrupted attack chain',stats.highest_attack_chain]);
    if((stats.combo_score||0)>0)values.push(['Combo score',stats.combo_score]);
    if((stats.guard_breaks||0)>0)values.push(['Guard breaks suffered',stats.guard_breaks]);
    if((stats.heavy_attacks||0)>0)values.push(['Heavy basic attacks',stats.heavy_attacks]);
    if((stats.aerial_attacks||0)>0)values.push(['Aerial basic attacks',stats.aerial_attacks]);
    if((stats.sweep_attacks||0)>0)values.push(['Grounded sweep attacks',stats.sweep_attacks]);
    if((stats.launchers||0)>0)values.push(['Launchers against small enemies',stats.launchers]);
    if((stats.downed_followups||0)>0)values.push(['Downed enemy follow-ups',stats.downed_followups]);
    if((stats.pack_attacks||0)>0)values.push(['Coordinated pack attacks',stats.pack_attacks]);
    if((stats.summon_punishes||0)>0)values.push(['Summon arrival punishes',stats.summon_punishes]);
    if((stats.flank_attempts||0)>0)values.push(['Enemy flank maneuvers',stats.flank_attempts]);
    if((stats.rear_strikes||0)>0)values.push(['Unshielded rear strikes',stats.rear_strikes]);
    if(run.replay_seed!==undefined)values.push(['Deterministic combat replay seed',String(run.replay_seed)]);
    values.push(['Largest hit',stats.largest_hit],['Mana spent',stats.mana_spent],['Skills cast',stats.skills_cast]);
    if(((stats.charged_finishers||0)+(stats.empty_finishers||0)+(stats.charges_spent||0))>0){
      values.push(['Charged finishers',stats.charged_finishers],['Finishers without charges',stats.empty_finishers],['Charges spent',stats.charges_spent]);
    }
    values.push(['Highest basic combo strike (of 3)',stats.highest_combo],['Basic attacks',stats.attacks],['Jumps',stats.jumps]);
    const unclassifiedDamage=Math.max(0,(stats.damage_taken||0)-(stats.hazard_damage_taken||0)-(stats.enemy_damage_taken||0));
    if(unclassifiedDamage>.001)values.push(['Damage without source records',unclassifiedDamage]);
    const abilityUses=[...run.build.skills.map(skill=>[skill,'optional']),...(run.build.signatures||[]).map(skill=>[skill,skill.role||'class']),...(run.build.ultimate?[[run.build.ultimate,'ultimate']]:[])];
    let attributed=0,attributedMana=0,attributedHealing=0,attributedBarrier=0;
    for(const [skill,kind] of abilityUses){
      const casts=stats.skill_uses?.[skill.id]||0,mana=stats.skill_mana?.[skill.id]||0,healing=stats.skill_healing?.[skill.id]||0,barrier=stats.skill_barrier?.[skill.id]||0,hits=stats.skill_hits?.[skill.id]||0;
      attributed+=casts;attributedMana+=mana;attributedHealing+=healing;attributedBarrier+=barrier;
      const name=skill.name+' ('+kind+')';
      values.push(['Casts · '+name,casts],['Mana · '+name,mana]);
      if(hits>0)values.push(['Hits · '+name,hits]);
      if(healing>0)values.push(['Healing · '+name,healing]);
      if(barrier>0)values.push(['Absorbed · '+name,barrier]);
    }
    if((stats.skills_cast||0)>attributed)values.push(['Earlier casts without per-skill records',stats.skills_cast-attributed]);
    if((stats.mana_spent||0)>attributedMana+.001)values.push(['Earlier mana without per-skill records',stats.mana_spent-attributedMana]);
    if((stats.healing||0)>attributedHealing+.001)values.push(['Healing without per-skill records',stats.healing-attributedHealing]);
    if((stats.barrier_blocked||0)>attributedBarrier+.001)values.push(['Absorption without per-skill records',stats.barrier_blocked-attributedBarrier]);
    const rankedUses=abilityUses.map(([skill,kind])=>({name:skill.name+' ('+kind+')',casts:stats.skill_uses?.[skill.id]||0}));
    const mostCasts=Math.max(0,...rankedUses.map(skill=>skill.casts));
    const mostUsed=rankedUses.filter(skill=>skill.casts===mostCasts);
    values.push(['Most used recorded skill',mostCasts>0?(mostUsed.length>1?'Tie: ':'')+mostUsed.map(skill=>skill.name).join(', ')+' — '+numbers.format(mostCasts)+' '+(mostCasts===1?'cast':'casts')+(mostUsed.length>1?' each':''):stats.skills_cast>0?'Per-skill records unavailable.':'No confirmed skill casts.']);
    const rankedDamage=abilityUses.map(([skill,kind])=>({name:skill.name+' ('+kind+')',damage:stats.skill_damage?.[skill.id]||0}));
    const mostDmg=Math.max(0,...rankedDamage.map(s=>s.damage));
    const topDamage=rankedDamage.filter(s=>s.damage===mostDmg);
    values.push(['Highest recorded skill damage',mostDmg>0?(topDamage.length>1?'Tie: ':'')+topDamage.map(s=>s.name).join(', ')+' — '+numbers.format(mostDmg)+' damage'+(topDamage.length>1?' each':''):stats.skills_cast>0?'No per-skill damage recorded.':'No confirmed skill casts.']);
    const classSummary=run.build?.class_name||'Unknown';const subclass=run.build?.subclass_name||'';
    const totalCasts=stats.skills_cast||0;const totalDmg=stats.damage_dealt||0;const totalHeal=stats.healing||0;
    values.push(['Class performance',classSummary+(subclass?' · '+subclass:'')+' — '+numbers.format(totalCasts)+' casts · '+numbers.format(totalDmg)+' damage'+((totalHeal>0)?' · '+numbers.format(totalHeal)+' healing':'')]);
    if((stats.guards||0)>0)values.push(['Successful guards',numbers.format(stats.guards)+(stats.perfect_guards?' ('+numbers.format(stats.perfect_guards)+' perfect)':'')]);
    if((stats.dodges||0)>0)values.push(['Hazard evasions (airborne dodges)',numbers.format(stats.dodges)]);
    const runSplits=formatRoomSplits(run.room_splits);
    if(runSplits)values.push(['Room time splits',runSplits]);
    const runFamilies=formatFamilyDefeats(run.monster_records);
    if(runFamilies)values.push(['Enemies defeated by family',runFamilies]);
    const runCompObj=formatCompletedObjectives(run.objectives);
    if(runCompObj)values.push(['Completed optional objectives',runCompObj]);
    const runFailObj=formatFailedObjectives(run.objectives);
    if(runFailObj)values.push(['Failed optional objectives',runFailObj]);
    const runMilestones=formatCosmeticMilestones(run);
    if(runMilestones)values.push(['Cosmetic milestones',runMilestones]);
    const runImproved=formatImprovedRecords(run.last_clear);
    if(runImproved)values.push(['Improved personal records',runImproved]);
    const past=run.past_expeditions||{};
    const career=[['Enemies defeated',(past.enemies||0)+(stats.kills||0)],['Bosses defeated',(past.bosses||0)+(stats.bosses||0)],['Treasure goblins defeated',(past.treasure_goblins||0)+(stats.treasure_goblins||0)],['Gold banked',(past.gold||0)+(run.banked_gold||0)],['Gear pieces banked',(past.gear||0)+(run.banked_items?.length||0)]];
    const careerKey=JSON.stringify(career);if($('rift-career-statistics').dataset.values!==careerKey){$('rift-career-statistics').dataset.values=careerKey;$('rift-career-statistics').replaceChildren();for(const [label,value] of career){const dt=document.createElement('dt'),dd=document.createElement('dd');dt.textContent=label;dd.textContent=numbers.format(value);$('rift-career-statistics').append(dt,dd);}}
    const key=JSON.stringify(values);
    if(summaryKey!==key){summaryKey=key;$('rift-statistics').replaceChildren();for(const [label,value] of values){const dt=document.createElement('dt'),dd=document.createElement('dd');dt.textContent=label;dd.textContent=typeof value==='string'?value:numbers.format(value||0);if(['Current mission clear streak','Best mission clear streak','Highest basic combo strike (of 3)','Largest hit'].includes(label)){dt.dataset.personalRecord='true';dd.dataset.personalRecord='true';}$('rift-statistics').append(dt,dd);}}
    renderResultHint(run);
    const state=[run.id,run.level?.id,run.room,run.status,run.paused].join(':');
    if(state!==announced){
      announced=state;
      if(run.status==='cleared')put($('rift-announcer'),'Room '+(run.room+1)+' cleared. Loot is ready to bank.');
      else if(run.status==='defeated')put($('rift-announcer'),'Expedition ended. Banked rewards are safe.');
      else if(['complete','banked'].includes(run.status))put($('rift-announcer'),'Expedition finished. '+numbers.format(run.banked_gold)+' gold and '+numbers.format(run.banked_items.length)+' '+(run.banked_items.length===1?'item':'items')+' banked.');
      else if(run.paused)put($('rift-announcer'),'Expedition paused.');
    }
  }
  let lastHintRunId='',lastHintKey='';
  let dismissedHints=new Set();
  try{const raw=localStorage.getItem('riftDismissedHints');if(raw)dismissedHints=new Set(JSON.parse(raw));}catch(_){}
  const hintDismissBtn=$('rift-dismiss-hint');
  if(hintDismissBtn)hintDismissBtn.onclick=()=>{if(lastHintKey){dismissedHints.add(lastHintKey);try{localStorage.setItem('riftDismissedHints',JSON.stringify([...dismissedHints]));}catch(_){}}const el=$('rift-result-hint');if(el)el.hidden=true;};
  function renderResultHint(run){
    const el=$('rift-result-hint'),txt=$('rift-result-hint-text');
    if(!el||!txt)return;
    const terminal=['defeated','complete','banked'].includes(run.status);
    if(!terminal){el.hidden=true;return;}
    const stats=run.stats||{};
    const hints=[];
    if(run.status==='defeated'){
      if((stats.guards||0)===0&&(stats.hits_taken||0)>3)hints.push({key:'guard-unused',text:'You took '+numbers.format(stats.hits_taken)+' hits without guarding. Hold Guard ('+((window.RiftControls?.label('guard'))||'L')+') to reduce frontal damage by 82%.'});
      if((stats.dodges||0)===0&&(stats.hazard_damage_taken||0)>stats.damage_taken*.3)hints.push({key:'dodge-hazards',text:'Hazards dealt '+numbers.format(stats.hazard_damage_taken)+' damage. Jump over hazard pulses when the warning zone appears.'});
      if((stats.healing||0)===0&&(stats.skills_cast||0)>5)hints.push({key:'no-healing',text:'No healing received this expedition. Consider equipping a skill with health recovery.'});
      const avgHitSize=(stats.hits_taken||0)>0?(stats.damage_taken||0)/(stats.hits_taken):0;
      if(avgHitSize>30&&(stats.guard_blocked||0)<avgHitSize)hints.push({key:'large-hits',text:'Average hit dealt '+numbers.format(avgHitSize)+' damage. Guard or jump to avoid the hardest-hitting attacks.'});
      if(run.defeated_by_hazard)hints.push({key:'hazard-defeat-'+run.defeated_by_hazard.kind,text:hazardDefeatHint(run.defeated_by_hazard)});
      if(run.room===0)hints.push({key:'room1-defeat',text:'Defeated in the first tier. Try a lower mission difficulty, or review your build and equipped skills before entering.'});
    }else{
      if((stats.hits_taken||0)===0)hints.push({key:'flawless',text:'Flawless clear — no damage taken. Impressive execution.'});
      else if((stats.damage_taken||0)<20)hints.push({key:'near-flawless',text:'Near-flawless clear with only '+numbers.format(stats.damage_taken)+' damage taken.'});
      if((stats.guards||0)>10)hints.push({key:'heavy-guard',text:'Strong defensive play with '+numbers.format(stats.guards)+' successful guards.'});
    }
    if(hints.length===0){el.hidden=true;return;}
    const available=hints.filter(h=>!dismissedHints.has(h.key));
    if(available.length===0){el.hidden=true;return;}
    let chosen;
    if(run.id!==lastHintRunId){lastHintRunId=run.id;chosen=available[0];}
    else{const current=available.find(h=>h.key===lastHintKey);chosen=current||available[0];}
    lastHintKey=chosen.key;
    txt.textContent=chosen.text;
    el.hidden=false;
  }
  function buildResultSummary(run){
    if(!run)return '';
    const stats=run.stats||{};
    const isLost=run.status==='defeated';
    const isComplete=run.status==='complete';
    const isBanked=run.status==='banked';
    const missionName=run.level?.name||('Mission '+(run.level?.id||1));
    const roomIndex=(run.room||0)+1;
    const roomName=run.level?.rooms?.[run.room]?.name||('Tier '+roomIndex);
    const outcomeText=isComplete?'Expedition Cleared (All 3 Tiers)':isBanked?'Voluntary Exit at Checkpoint (Tier '+roomIndex+')':isLost?'Expedition Defeat (Tier '+roomIndex+': '+roomName+')':'Expedition in Progress';
    const duration=Number(stats.seconds||0).toFixed(1)+'s combat';
    const damageDealt=numbers.format(stats.damage_dealt||0);
    const damageTaken=numbers.format(stats.damage_taken||0)+' ('+numbers.format(stats.hits_taken||0)+' '+((stats.hits_taken===1)?'hit':'hits')+')';
    const bankedRewards=numbers.format(run.banked_gold||0)+' gold · '+numbers.format(run.banked_items?.length||0)+' '+((run.banked_items?.length===1)?'item':'items');

    const lines=[
      'Abyss Rift Brawl — Result Summary',
      'Mission: '+missionName,
      'Outcome: '+outcomeText,
      'Duration: '+duration,
      'Damage Dealt: '+damageDealt,
      'Damage Taken: '+damageTaken,
      'Banked Rewards: '+bankedRewards
    ];
    if(isLost){
      const finalHit=run.defeated_by_hazard?('Hazard: '+run.defeated_by_hazard.kind):run.defeated_by_boss?('Boss: '+run.defeated_by_boss):run.defeated_by_enemy?('Enemy: '+run.defeated_by_enemy):'';
      const exactCause=run.defeat_cause||finalHit;
      if(exactCause)lines.push('Defeat Cause: '+exactCause);
      if(finalHit)lines.push('Final Hit: '+finalHit);
      const lostGold=numbers.format(run.gold||0);
      const lostDrops=numbers.format((run.drops||[]).filter(d=>!d.banked).length);
      lines.push('Lost Pending Finds: '+lostGold+' gold · '+lostDrops+' items');
    }
    if((stats.healing||0)>0){
      lines.push('Healing Received: '+numbers.format(stats.healing));
    }
    if((stats.guard_blocked||0)>0){
      lines.push('Guarded Damage: '+numbers.format(stats.guard_blocked));
    }
    if((stats.barrier_blocked||0)>0){
      lines.push('Barrier Absorbed: '+numbers.format(stats.barrier_blocked));
    }
    const splitsText=formatRoomSplits(run.room_splits);
    if(splitsText)lines.push('Room Splits: '+splitsText);
    const famText=formatFamilyDefeats(run.monster_records);
    if(famText)lines.push('Defeated by Family: '+famText);
    const compObj=formatCompletedObjectives(run.objectives);
    if(compObj)lines.push('Completed Objectives: '+compObj);
    const failObj=formatFailedObjectives(run.objectives);
    if(failObj)lines.push('Failed Objectives: '+failObj);
    const cosmeticText=formatCosmeticMilestones(run);
    if(cosmeticText)lines.push('Cosmetic Milestones: '+cosmeticText);
    const improvedText=formatImprovedRecords(run.last_clear);
    if(improvedText)lines.push('Improved Records: '+improvedText);
    if((stats.armor_piercing_damage||0)>0){
      lines.push('Armor-Piercing Damage: '+numbers.format(stats.armor_piercing_damage));
    }
    if((stats.highest_attack_chain||0)>0){
      lines.push('Uninterrupted Attack Chain: '+numbers.format(stats.highest_attack_chain));
    }
    if((stats.combo_score||0)>0){
      lines.push('Combo Score: '+numbers.format(stats.combo_score));
    }
    if((stats.guard_breaks||0)>0){
      lines.push('Guard Breaks: '+numbers.format(stats.guard_breaks));
    }
    if((stats.heavy_attacks||0)>0){
      lines.push('Heavy Basic Attacks: '+numbers.format(stats.heavy_attacks));
    }
    if((stats.aerial_attacks||0)>0){
      lines.push('Aerial Basic Attacks: '+numbers.format(stats.aerial_attacks));
    }
    if((stats.sweep_attacks||0)>0){
      lines.push('Grounded Sweeps: '+numbers.format(stats.sweep_attacks));
    }
    if((stats.launchers||0)>0){
      lines.push('Launchers: '+numbers.format(stats.launchers));
    }
    if((stats.downed_followups||0)>0){
      lines.push('Downed Enemy Follow-ups: '+numbers.format(stats.downed_followups));
    }
    if((stats.pack_attacks||0)>0){
      lines.push('Coordinated Pack Attacks: '+numbers.format(stats.pack_attacks));
    }
    if((stats.summon_punishes||0)>0){
      lines.push('Summon Arrival Punishes: '+numbers.format(stats.summon_punishes));
    }
    if((stats.flank_attempts||0)>0){
      lines.push('Enemy Flank Maneuvers: '+numbers.format(stats.flank_attempts));
    }
    if((stats.rear_strikes||0)>0){
      lines.push('Unshielded Rear Strikes: '+numbers.format(stats.rear_strikes));
    }
    if(run.replay_seed!==undefined){
      lines.push('Replay Seed: '+run.replay_seed);
    }
    return lines.join('\n');
  }
  window.RiftHUD={contextHintsEnabled:()=>hintsEnabled,hazardDefeatHint,nearbyCover,update,duration,setRequestedRange,getRequestedRange,updateLatency,detectPlayerAreaEffects,getHealthThreshold,triggerTransientCounter,updateLastEncounter,buildResultSummary};
})();
