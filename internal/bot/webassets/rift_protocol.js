(function(){
  'use strict';
  const object=value=>value!==null&&typeof value==='object'&&!Array.isArray(value);
  const finite=value=>typeof value==='number'&&Number.isFinite(value);
  const nonnegative=value=>finite(value)&&value>=0;
  const text=value=>typeof value==='string';
  const list=(value,valid)=>Array.isArray(value)&&value.every(valid);
  const optionalList=(value,valid)=>value==null||list(value,valid);
  const point=value=>object(value)&&finite(value.x)&&finite(value.y);
  const box=value=>point(value)&&nonnegative(value.w)&&nonnegative(value.h);
  const reference=value=>object(value)&&['self','area','projectile'].includes(value.target)&&nonnegative(value.horizontal)&&nonnegative(value.depth)&&nonnegative(value.healing)&&typeof value.barrier==='boolean';
  const skill=value=>object(value)&&text(value.id)&&text(value.name)&&nonnegative(value.cost)&&nonnegative(value.cooldown)&&(value.reference===undefined||reference(value.reference));
  const actor=value=>object(value)&&text(value.id)&&text(value.name)&&text(value.kind)&&finite(value.x)&&finite(value.y)&&nonnegative(value.hp)&&nonnegative(value.max_hp)&&value.max_hp>0&&finite(value.facing);
  const build=value=>object(value)&&text(value.name)&&text(value.class)&&list(value.skills,skill)&&list(value.gear,text)&&optionalList(value.owned_ultimates,text)&&(!value.signatures||list(value.signatures,skill))&&(!value.ultimate||skill(value.ultimate));
  const drop=value=>point(value)&&text(value.id)&&nonnegative(value.gold)&&(!value.gear||object(value.gear)&&text(value.gear.Name)&&text(value.gear.Slot));
  const projectile=value=>point(value)&&finite(value.vx)&&finite(value.vy)&&text(value.kind)&&(value.enemy===undefined||typeof value.enemy==='boolean');
  const event=value=>point(value)&&nonnegative(value.id)&&text(value.kind)&&(value.value===undefined||finite(value.value));
  const arena=value=>object(value)&&text(value.name)&&optionalList(value.obstacles,box)&&optionalList(value.hazards,h=>box(h)&&text(h.kind)&&finite(h.period)&&h.period>0&&nonnegative(h.offset)&&nonnegative(h.duration)&&(h.disabled===undefined||typeof h.disabled==='boolean')&&(h.generator_id===undefined||text(h.generator_id)&&h.generator_id.length>0));
  function level(value){return object(value)&&Number.isInteger(value.id)&&value.id>0&&text(value.name)&&text(value.region_name)&&text(value.tactic)&&text(value.difficulty)&&Number.isInteger(value.region)&&value.region>=0&&value.region<10&&list(value.rooms,arena)&&value.rooms.length===3;}
  const splits=value=>Array.isArray(value)&&value.length===3&&value.every(seconds=>seconds===null||nonnegative(seconds));
  const attempt=value=>object(value)&&(value.splits===undefined||splits(value.splits))&&Number.isInteger(value.mission)&&value.mission>=1&&value.mission<=100&&['completed','defeated','exited','expired'].includes(value.outcome)&&Number.isSafeInteger(value.at_ms)&&value.at_ms>=0&&value.at_ms<=8640000000000000&&text(value.class)&&nonnegative(value.seconds)&&nonnegative(value.hp)&&nonnegative(value.max_hp)&&(value.hits===undefined||Number.isSafeInteger(value.hits)&&value.hits>=0);
  const objective=value=>object(value)&&text(value.id)&&text(value.name)&&text(value.description)&&nonnegative(value.current)&&nonnegative(value.target)&&["active","failed","complete"].includes(value.status)&&(value.reason===undefined||text(value.reason));
  const objectives=value=>object(value)&&Number.isInteger(value.mission)&&value.mission>=1&&value.mission<=100&&typeof value.finished==="boolean"&&(value.banked===undefined||typeof value.banked==="boolean")&&(value.difficulty===undefined||text(value.difficulty))&&list(value.entries,objective);
  const sigilObjective=value=>object(value)&&value.kind==='sigils'&&text(value.name)&&text(value.description)&&value.target===3&&Number.isInteger(value.collected)&&value.collected>=0&&value.collected<=3&&value.complete===(value.collected===3)&&list(value.pickups,p=>point(p)&&Number.isInteger(p.id)&&p.id>=1&&p.id<=3&&typeof p.collected==='boolean')&&value.pickups.length===3&&new Set(value.pickups.map(p=>p.id)).size===3&&value.pickups.filter(p=>p.collected).length===value.collected;
  const circleObjective=value=>object(value)&&value.kind==='hold_circle'&&text(value.name)&&text(value.description)&&value.target===15&&nonnegative(value.seconds)&&value.seconds<=15&&value.complete===(value.seconds===15)&&typeof value.contested==='boolean'&&typeof value.charging==='boolean'&&point(value.zone)&&value.zone.radius_x===80&&value.zone.radius_y===44;
  const waveObjective=value=>object(value)&&value.kind==='survive_waves'&&text(value.name)&&text(value.description)&&value.target===3&&Number.isInteger(value.wave)&&value.wave>=1&&value.wave<=3&&typeof value.complete==='boolean'&&(!value.complete||value.wave===3)&&nonnegative(value.next_wave_seconds)&&value.next_wave_seconds<=2.5&&(value.wave<3||value.next_wave_seconds===0)&&list(value.waves,group=>list(group,actor)&&group.length>0)&&value.waves.length===3&&new Set(value.waves.flat().map(enemy=>enemy.id)).size===value.waves.flat().length;
  const totemObjective=value=>object(value)&&value.kind==='destroy_totems'&&text(value.name)&&text(value.description)&&value.target===3&&Number.isInteger(value.collected)&&value.collected>=0&&value.collected<=3&&value.complete===(value.collected===3);
  const relicObjective=value=>object(value)&&value.kind==='carry_relic'&&text(value.name)&&text(value.description)&&value.target===1&&[0,1].includes(value.collected)&&value.complete===(value.collected===1)&&typeof value.carrying==='boolean'&&point(value.relic)&&value.relic.id===1&&typeof value.relic.collected==='boolean'&&value.relic.collected===(value.carrying||value.complete)&&!(value.carrying&&value.complete)&&point(value.zone)&&value.zone.radius_x===45&&value.zone.radius_y===28;
  const generatorObjective=value=>object(value)&&value.kind==='disable_generators'&&text(value.name)&&text(value.description)&&Number.isInteger(value.target)&&value.target>=1&&value.target<=3&&Number.isInteger(value.collected)&&value.collected>=0&&value.collected<=value.target&&value.complete===(value.collected===value.target);
  const roomObjective=value=>generatorObjective(value)||relicObjective(value)||totemObjective(value)||sigilObjective(value)||circleObjective(value)||waveObjective(value);
  function generatorLinks(run){
    if(run.room_objective?.kind!=='disable_generators')return true;
    const hazards=run.level?.rooms?.[run.room]?.hazards;
    if(!Array.isArray(hazards)||!list(run.enemies,actor))return false;
    const generators=run.enemies.filter(e=>e.kind==='generator');
    return hazards.length===run.room_objective.target&&generators.length===hazards.length&&new Set(generators.map(e=>e.id)).size===generators.length&&new Set(hazards.map(h=>h?.generator_id)).size===hazards.length&&hazards.every(h=>object(h)&&generators.some(e=>e.id===h.generator_id&&(h.disabled===true)===(e.hp===0)))&&hazards.filter(h=>h.disabled).length===run.room_objective.collected;
  }
  function run(value){
    if(!object(value))return false;
    if(value.schema!==1)throw new Error('This expedition uses an unsupported save version. Reload the page to get the current game before recovering.');
    if(!['fighting','cleared','defeated','complete','banked','expired'].includes(value.status))throw new Error('The expedition has an unrecognized state. Recover the saved expedition before continuing.');
    return text(value.id)&&value.id.length>0&&Number.isInteger(value.revision)&&value.revision>=0&&
      Number.isInteger(value.room)&&value.room>=0&&value.room<3&&typeof value.paused==='boolean'&&
      (value.room_objective?.kind!=='destroy_totems'||list(value.enemies,actor)&&value.enemies.filter(e=>e.kind==='totem').length===3&&new Set(value.enemies.filter(e=>e.kind==='totem').map(e=>e.id)).size===3&&value.enemies.filter(e=>e.kind==='totem'&&e.hp===0).length===value.room_objective.collected)&&
      generatorLinks(value)&&build(value.build)&&actor(value.player)&&nonnegative(value.player.mana)&&list(value.enemies,actor)&&
      list(value.projectiles,projectile)&&list(value.drops,drop)&&optionalList(value.events,event)&&
      list(value.banked_items,text)&&optionalList(value.banked_loot,item=>object(item)&&text(item.name)&&Number.isSafeInteger(item.rarity)&&item.rarity>=0)&&(value.banked_at_ms===undefined||Number.isSafeInteger(value.banked_at_ms)&&value.banked_at_ms>=0&&value.banked_at_ms<=8640000000000000)&&nonnegative(value.gold)&&nonnegative(value.banked_gold)&&nonnegative(value.clock)&&nonnegative(value.counter)&&
      object(value.skill_timers)&&Object.values(value.skill_timers).every(nonnegative)&&
      (!value.level||level(value.level))&&(value.practice===undefined||object(value.practice)&&["movement","jump","combo","guard","hazard","boss"].includes(value.practice.mode)&&(value.practice.mode!=="boss"||actor(value.practice.boss_start))&&(value.practice.slow_telegraphs===undefined||typeof value.practice.slow_telegraphs==="boolean")&&nonnegative(value.practice.goal_x)&&Number.isSafeInteger(value.practice.hits)&&value.practice.hits>=0&&typeof value.practice.completed==="boolean"&&arena(value.practice.arena)&&(value.practice.dodges===undefined||Number.isSafeInteger(value.practice.dodges)&&value.practice.dodges>=0))&&optionalList(value.encounter_plan,room=>list(room,actor))&&
      (value.objective_history===undefined||object(value.objective_history)&&Object.values(value.objective_history).every(counts=>object(counts)&&Object.values(counts).every(count=>Number.isSafeInteger(count)&&count>=0)))&&
      (value.room_objective===undefined||roomObjective(value.room_objective))&&(value.objectives===undefined||objectives(value.objectives))&&(value.last_objectives===undefined||objectives(value.last_objectives))&&
      optionalList(value.completed_levels,id=>Number.isInteger(id)&&id>0)&&
      (value.monster_records===undefined||object(value.monster_records)&&Object.values(value.monster_records).every(record=>object(record)&&Number.isSafeInteger(record.first_seen_ms)&&record.first_seen_ms>=0&&record.first_seen_ms<=8640000000000000&&Number.isSafeInteger(record.defeats)&&record.defeats>=0&&(record.fastest_clear_seconds===undefined||nonnegative(record.fastest_clear_seconds))))&&
      (value.past_expeditions===undefined||object(value.past_expeditions)&&['enemies','bosses','treasure_goblins','gold','gear'].every(key=>Number.isSafeInteger(value.past_expeditions[key])&&value.past_expeditions[key]>=0))&&
      (value.last_clear===undefined||object(value.last_clear)&&Number.isInteger(value.last_clear.mission)&&value.last_clear.mission>=1&&value.last_clear.mission<=100&&typeof value.last_clear.first==='boolean'&&list(value.last_clear.records,key=>['time','health','hits'].includes(key)))&&
      (value.last_encounter===undefined||object(value.last_encounter)&&Number.isInteger(value.last_encounter.mission)&&value.last_encounter.mission>=1&&value.last_encounter.mission<=100&&text(value.last_encounter.room_name)&&['cleared','defeated','completed'].includes(value.last_encounter.outcome)&&nonnegative(value.last_encounter.seconds)&&nonnegative(value.last_encounter.player_hp)&&nonnegative(value.last_encounter.player_max_hp))&&
      (value.room_baseline===undefined||object(value.room_baseline))&&
      (value.room_splits===undefined||splits(value.room_splits))&&
      (value.attempt_history===undefined||list(value.attempt_history,attempt)&&value.attempt_history.length<=50)&&
      ['clear_streak','best_clear_streak'].every(key=>value[key]===undefined||Number.isSafeInteger(value[key])&&value[key]>=0)&&
      (value.catchup===undefined||typeof value.catchup==='boolean')&&
      (value.saved_at_ms===undefined||Number.isSafeInteger(value.saved_at_ms)&&value.saved_at_ms>=0&&value.saved_at_ms<=8640000000000000)&&
      (!value.stats||object(value.stats)&&Object.entries(value.stats).every(([key,count])=>['skill_uses','skill_hits'].includes(key)?object(count)&&Object.values(count).every(n=>Number.isSafeInteger(n)&&n>=0):['skill_mana','skill_healing','skill_barrier'].includes(key)?object(count)&&Object.values(count).every(nonnegative):nonnegative(count)));
  }
  function validate(data,method,request){
    let valid=object(data)&&data.ok===true;
    if(valid&&method==='GET')valid=optionalList(data.objective_options,objective)&&build(data.build)&&list(data.rooms,text)&&list(data.levels,level)&&data.levels.length>0&&data.levels.every((l,i)=>l.id===i+1)&&list(data.bestiary,unit=>actor(unit)&&text(unit.tier)&&text(unit.art_key)&&(unit.training===undefined||object(unit.training)&&nonnegative(unit.training.windup_seconds)&&typeof unit.training.resists_knockdown==="boolean"&&typeof unit.training.interruptible==="boolean"))&&(data.run===null||run(data.run));
    else if(valid)valid=run(data.run)&&(request.kind==='start'||data.run.id===request.run_id);
    if(!valid)throw new Error('Received an incomplete expedition update. Recover the saved expedition before continuing.');
    return data;
  }
  window.RiftProtocol={validate};
})();
