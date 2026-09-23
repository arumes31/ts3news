(function(){
  'use strict';
  const $=id=>document.getElementById(id),numbers=new Intl.NumberFormat(undefined,{maximumFractionDigits:0});
  let announced='',summaryKey='',lastAnnouncedBossAttack='';
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
  function reason(skill,run,playing){
    if(!['fighting','cleared'].includes(run.status))return 'Expedition ended';
    if(!playing||run.paused)return 'Paused';
    const remaining=run.skill_timers[skill.id]||0;
    if(remaining>0)return remaining.toFixed(1)+' seconds cooldown';
    if(run.player.mana<skill.cost)return Math.ceil(skill.cost-run.player.mana)+' more mana needed';
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
    if(ref.target==='self'){
      put(signal,'Range: '+skill.name+' · Self ('+(ref.barrier?'Barrier':'Healing')+')');
    }else if(ref.target==='area'){
      put(signal,'Range: '+skill.name+' · Area ('+ref.horizontal+'h × '+ref.depth+'d)');
    }else{
      const facing=(run?.player?.facing??1)<0?'←':'→';
      put(signal,'Range: '+skill.name+' · Projectile (lane ±'+ref.depth+'d, facing '+facing+')');
    }
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
    const p=run.player;
    const arena=run.practice?.arena||run.level?.rooms?.[run.room];
    if(arena&&Array.isArray(arena.hazards)){
      for(let i=0;i<arena.hazards.length;i++){
        const h=arena.hazards[i];
        const inside=p.x>=h.x&&p.x<=h.x+h.w&&p.y>=h.y&&p.y<=h.y+h.h;
        if(inside){
          const phase=(run.clock+h.offset)%h.period;
          const warning=phase<1.2;
          const active=phase>=1.2&&phase<1.2+h.duration;
          const kindName=h.kind.charAt(0).toUpperCase()+h.kind.slice(1);
          if(active){
            const evading=(p.jump||0)>0.1;
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
      if(e.hp>0&&e.windup>0&&e.kind==='boss'){
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
        name:'Slowed',
        label:'Area effect: Slowed ('+run.skill_timers.slowed.toFixed(1)+'s)',
      };
    }
    return null;
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

    let encounter=run.last_encounter;
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
      ['Enemies defeated',String(encounter.enemies||0)+(encounter.boss_name?' (Boss: '+encounter.boss_name+')':'')],
      ['Damage dealt',numbers.format(encounter.damage_dealt||0)],
      ['Damage taken',numbers.format(encounter.damage_taken||0)+' ('+(encounter.hits_taken||0)+' '+((encounter.hits_taken===1)?'hit':'hits')+')'],
      ['Guarded damage',numbers.format(encounter.guard_blocked||0)],
      ['Barrier absorbed',numbers.format(encounter.barrier_blocked||0)]
    ];

    if(isDefeated&&encounter.defeated_by_boss){
      rows.push(['Defeated by boss',encounter.defeated_by_boss]);
    }

    if((encounter.treasure_escaped||0)>0){
      rows.push(['Treasure goblins escaped',String(encounter.treasure_escaped)+' · No loot or defeat credit']);
    }

    if((encounter.healing||0)>0){
      rows.push(['Healing received',numbers.format(encounter.healing)]);
    }

    if((encounter.gold_gained||0)>0||(encounter.loot_items||0)>0){
      const lootText=numbers.format(encounter.gold_gained||0)+' gold'+((encounter.loot_items||0)>0?' · '+(encounter.loot_items)+' '+((encounter.loot_items===1)?'item':'items'):'');
      rows.push([isDefeated?'Unbanked loot lost':'Loot collected',lootText]);
    }

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
  }
  function update(run,playing,replay=false){
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
        triggerTransientCounter($('rift-resource-gain'),'+'+diff+' '+(diff===1?resName:resName+'s'),'resource');
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
    recentDamage(run,replay);window.RiftOnboarding.update(run);window.RiftRecords.update(run);updateLastEncounter(run);
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
    const guardNode=$('rift-guard-reduction');
    put(guardNode,run.player.guard&&['fighting','cleared'].includes(run.status)?(playing&&!run.paused?'Guard: 82% frontal reduction after armor':'Guard paused'):'Guard inactive');
    const recentGuard=(run.events||[]).some(e=>(e.kind==='block'||e.kind==='perfect_guard')&&(run.counter-e.id)<6);
    if(recentGuard)guardNode.dataset.guardedHit='';else delete guardNode.dataset.guardedHit;
    const guardBtn=document.querySelector('.rift-basics button[data-bind="guard"]');
    if(guardBtn){if(recentGuard)guardBtn.dataset.guardedHit='';else delete guardBtn.dataset.guardedHit;}
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
    put($('rift-objective'),run.status==='fighting'?(boss?'Defeat '+boss.name+' and its defenders':'Clear the enemy patrol'):run.status==='cleared'?'Room secured · Loot ready to bank':run.status==='defeated'?'Expedition ended':'Rewards secured');
    put($('rift-jump-ready'),(run.skill_timers.jump||0)>0?'Jump '+run.skill_timers.jump.toFixed(1)+'s':'Jump ready');
    put($('rift-combo-step'),'Strike '+(run.combo||0)+'/3');
    put($('rift-barrier-state'),run.barrier>0?'Barrier '+numbers.format(run.barrier):'No barrier');
    put($('rift-slow-state'),run.skill_timers.slowed>0?'Slowed '+run.skill_timers.slowed.toFixed(1)+'s':'Normal speed');
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
    const marked=living.find(e=>e.id===run.marked);put($('rift-mark-state'),marked?'Marked: '+marked.name:'No marked target');
    put($('rift-ultimate-state'),run.build.ultimate?'Ultimate: '+run.build.ultimate.name+' · '+reason(run.build.ultimate,run,playing):'No ultimate in this expedition');
    const finisher=run.build.signatures?.find(s=>s.role==='finisher');
    put($('rift-finisher-state'),!finisher?'Class abilities unlock in Abyss':run.resource>0?'Finisher: '+reason(finisher,run,playing)+' · '+run.resource+' charges':'Build charges with '+window.RiftControls.label('signature0'));
    $('rift-health-cost').hidden=run.build.class!=='voidwalker'||!finisher;
    if(!$('rift-health-cost').hidden)put($('rift-health-cost'),'Finisher health cost: '+window.RiftAbilities.healthCost(run,finisher).toFixed(1)+' HP'+(run.resource>0?' · leaves at least 1 HP':' · no charges to spend'));
    const builder=run.build.signatures?.find(s=>s.role==='builder');
    coaching.hidden=coachingDismissed||!builder||!finisher||(stats.empty_finishers||0)<3;
    if(!coaching.hidden)put(coaching.querySelector('p'),'Three or more finishers used no charges. Practice '+builder.name+' ('+window.RiftControls.label('signature0')+') before '+finisher.name+' ('+window.RiftControls.label('signature1')+'). Build up to three charges, then spend them with your finisher.');
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
          put(bossAttackNode,'⚡ '+attack+' · '+boss.windup.toFixed(1)+'s windup');
          bossAttackNode.hidden=false;
          if(playing&&!run.paused){
            const attackKey=(boss.attacks||0)+':'+attack;
            if(lastAnnouncedBossAttack!==attackKey){
              lastAnnouncedBossAttack=attackKey;
              put($('rift-announcer'),'Boss preparing '+attack+' · '+Math.ceil(boss.windup)+'s windup.');
            }
          }
        }else{
          bossAttackNode.hidden=true;
          lastAnnouncedBossAttack='';
        }
      }
    }else{
      if($('rift-boss-attack'))$('rift-boss-attack').hidden=true;
      lastAnnouncedBossAttack='';
    }
    for(const [id,skills] of [['rift-skills',run.build.skills],['rift-signatures',[...(run.build.signatures||[]),...(run.build.ultimate?[run.build.ultimate]:[])]]]){
      [...$(id).children].forEach((button,index)=>{const skill=skills[index];if(!skill)return;const why=reason(skill,run,playing);window.RiftAbilities.update(button,skill,run,why,skill===run.build.ultimate);});
    }
    const values=[['Current mission clear streak',run.clear_streak],['Best mission clear streak',run.best_clear_streak],['Paused seconds (completed pauses)',stats.paused_seconds],['Enemies defeated',stats.kills],['Bosses defeated',stats.bosses],['Treasure goblins defeated',stats.treasure_goblins],['Rooms cleared',stats.rooms_cleared],['Damage dealt',stats.damage_dealt],['Damage taken',stats.damage_taken],['Damaging hits taken',stats.hits_taken],['Healing received',stats.healing],['Guard prevented',stats.guard_blocked],['Barrier prevented',stats.barrier_blocked],['Armor prevented',stats.armor_blocked],['Largest hit',stats.largest_hit],['Mana spent',stats.mana_spent],['Skills cast',stats.skills_cast],['Charged finishers',stats.charged_finishers],['Finishers without charges',stats.empty_finishers],['Charges spent',stats.charges_spent],['Highest basic combo strike (of 3)',stats.highest_combo],['Basic attacks',stats.attacks],['Successful guards',stats.guards],['Jumps',stats.jumps]];
    const abilityUses=[...run.build.skills.map(skill=>[skill,'optional']),...(run.build.signatures||[]).map(skill=>[skill,skill.role||'class']),...(run.build.ultimate?[[run.build.ultimate,'ultimate']]:[])];
    let attributed=0,attributedMana=0,attributedHealing=0,attributedBarrier=0;for(const [skill,kind] of abilityUses){const casts=stats.skill_uses?.[skill.id]||0,mana=stats.skill_mana?.[skill.id]||0,healing=stats.skill_healing?.[skill.id]||0,barrier=stats.skill_barrier?.[skill.id]||0;attributed+=casts;attributedMana+=mana;attributedHealing+=healing;attributedBarrier+=barrier;const name=skill.name+' ('+kind+')';values.push(['Casts · '+name,casts],['Mana · '+name,mana],['Hits · '+name,stats.skill_hits?.[skill.id]||0],['Healing · '+name,healing],['Absorbed · '+name,barrier]);}
    if((stats.skills_cast||0)>attributed)values.push(['Earlier casts without per-skill records',stats.skills_cast-attributed]);
    if((stats.mana_spent||0)>attributedMana+.001)values.push(['Earlier mana without per-skill records',stats.mana_spent-attributedMana]);
    if((stats.healing||0)>attributedHealing+.001)values.push(['Healing without per-skill records',stats.healing-attributedHealing]);
    if((stats.barrier_blocked||0)>attributedBarrier+.001)values.push(['Absorption without per-skill records',stats.barrier_blocked-attributedBarrier]);
    const past=run.past_expeditions||{};
    const career=[['Enemies defeated',(past.enemies||0)+(stats.kills||0)],['Bosses defeated',(past.bosses||0)+(stats.bosses||0)],['Treasure goblins defeated',(past.treasure_goblins||0)+(stats.treasure_goblins||0)],['Gold banked',(past.gold||0)+(run.banked_gold||0)],['Gear pieces banked',(past.gear||0)+(run.banked_items?.length||0)]];
    const careerKey=JSON.stringify(career);if($('rift-career-statistics').dataset.values!==careerKey){$('rift-career-statistics').dataset.values=careerKey;$('rift-career-statistics').replaceChildren();for(const [label,value] of career){const dt=document.createElement('dt'),dd=document.createElement('dd');dt.textContent=label;dd.textContent=numbers.format(value);$('rift-career-statistics').append(dt,dd);}}
    const key=JSON.stringify(values);
    if(summaryKey!==key){summaryKey=key;$('rift-statistics').replaceChildren();for(const [label,value] of values){const dt=document.createElement('dt'),dd=document.createElement('dd');dt.textContent=label;dd.textContent=numbers.format(value||0);if(['Current mission clear streak','Best mission clear streak','Highest basic combo strike (of 3)','Largest hit'].includes(label)){dt.dataset.personalRecord='true';dd.dataset.personalRecord='true';}$('rift-statistics').append(dt,dd);}}
    const state=[run.id,run.level?.id,run.room,run.status,run.paused].join(':');
    if(state!==announced){
      announced=state;
      if(run.status==='cleared')put($('rift-announcer'),'Room '+(run.room+1)+' cleared. Loot is ready to bank.');
      else if(run.status==='defeated')put($('rift-announcer'),'Expedition ended. Banked rewards are safe.');
      else if(['complete','banked'].includes(run.status))put($('rift-announcer'),'Expedition finished. '+numbers.format(run.banked_gold)+' gold and '+numbers.format(run.banked_items.length)+' '+(run.banked_items.length===1?'item':'items')+' banked.');
      else if(run.paused)put($('rift-announcer'),'Expedition paused.');
    }
  }
  window.RiftHUD={update,duration,setRequestedRange,getRequestedRange,updateLatency,detectPlayerAreaEffects,getHealthThreshold,triggerTransientCounter,updateLastEncounter};
})();
