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
  const projectile=value=>point(value)&&finite(value.vx)&&finite(value.vy)&&text(value.kind);
  const event=value=>point(value)&&nonnegative(value.id)&&text(value.kind)&&(value.value===undefined||finite(value.value));
  const arena=value=>object(value)&&text(value.name)&&optionalList(value.obstacles,box)&&optionalList(value.hazards,h=>box(h)&&text(h.kind)&&finite(h.period)&&h.period>0&&nonnegative(h.offset)&&nonnegative(h.duration));
  function level(value){return object(value)&&Number.isInteger(value.id)&&value.id>0&&text(value.name)&&text(value.region_name)&&text(value.tactic)&&text(value.difficulty)&&Number.isInteger(value.region)&&value.region>=0&&value.region<10&&list(value.rooms,arena)&&value.rooms.length===3;}
  const splits=value=>Array.isArray(value)&&value.length===3&&value.every(seconds=>seconds===null||nonnegative(seconds));
  const attempt=value=>object(value)&&(value.splits===undefined||splits(value.splits))&&Number.isInteger(value.mission)&&value.mission>=1&&value.mission<=100&&['completed','defeated','exited','expired'].includes(value.outcome)&&Number.isSafeInteger(value.at_ms)&&value.at_ms>=0&&value.at_ms<=8640000000000000&&text(value.class)&&nonnegative(value.seconds)&&nonnegative(value.hp)&&nonnegative(value.max_hp)&&(value.hits===undefined||Number.isSafeInteger(value.hits)&&value.hits>=0);
  function run(value){
    if(!object(value))return false;
    if(value.schema!==1)throw new Error('This expedition uses an unsupported save version. Reload the page to get the current game before recovering.');
    if(!['fighting','cleared','defeated','complete','banked','expired'].includes(value.status))throw new Error('The expedition has an unrecognized state. Recover the saved expedition before continuing.');
    return text(value.id)&&value.id.length>0&&Number.isInteger(value.revision)&&value.revision>=0&&
      Number.isInteger(value.room)&&value.room>=0&&value.room<3&&typeof value.paused==='boolean'&&
      build(value.build)&&actor(value.player)&&nonnegative(value.player.mana)&&list(value.enemies,actor)&&
      list(value.projectiles,projectile)&&list(value.drops,drop)&&optionalList(value.events,event)&&
      list(value.banked_items,text)&&optionalList(value.banked_loot,item=>object(item)&&text(item.name)&&Number.isSafeInteger(item.rarity)&&item.rarity>=0)&&(value.banked_at_ms===undefined||Number.isSafeInteger(value.banked_at_ms)&&value.banked_at_ms>=0&&value.banked_at_ms<=8640000000000000)&&nonnegative(value.gold)&&nonnegative(value.banked_gold)&&nonnegative(value.clock)&&nonnegative(value.counter)&&
      object(value.skill_timers)&&Object.values(value.skill_timers).every(nonnegative)&&
      (!value.level||level(value.level))&&(value.practice===undefined||object(value.practice)&&["movement","jump","combo"].includes(value.practice.mode)&&nonnegative(value.practice.goal_x)&&Number.isSafeInteger(value.practice.hits)&&value.practice.hits>=0&&typeof value.practice.completed==="boolean"&&arena(value.practice.arena))&&optionalList(value.encounter_plan,room=>list(room,actor))&&
      optionalList(value.completed_levels,id=>Number.isInteger(id)&&id>0)&&
      (value.past_expeditions===undefined||object(value.past_expeditions)&&['enemies','bosses','treasure_goblins','gold','gear'].every(key=>Number.isSafeInteger(value.past_expeditions[key])&&value.past_expeditions[key]>=0))&&
      (value.last_clear===undefined||object(value.last_clear)&&Number.isInteger(value.last_clear.mission)&&value.last_clear.mission>=1&&value.last_clear.mission<=100&&typeof value.last_clear.first==='boolean'&&list(value.last_clear.records,key=>['time','health','hits'].includes(key)))&&
      (value.room_splits===undefined||splits(value.room_splits))&&
      (value.attempt_history===undefined||list(value.attempt_history,attempt)&&value.attempt_history.length<=50)&&
      ['clear_streak','best_clear_streak'].every(key=>value[key]===undefined||Number.isSafeInteger(value[key])&&value[key]>=0)&&
      (!value.stats||object(value.stats)&&Object.entries(value.stats).every(([key,count])=>['skill_uses','skill_hits'].includes(key)?object(count)&&Object.values(count).every(n=>Number.isSafeInteger(n)&&n>=0):['skill_mana','skill_healing','skill_barrier'].includes(key)?object(count)&&Object.values(count).every(nonnegative):nonnegative(count)));
  }
  function validate(data,method,request){
    let valid=object(data)&&data.ok===true;
    if(valid&&method==='GET')valid=build(data.build)&&list(data.rooms,text)&&list(data.levels,level)&&data.levels.length>0&&data.levels.every((l,i)=>l.id===i+1)&&list(data.bestiary,unit=>actor(unit)&&text(unit.tier)&&text(unit.art_key))&&(data.run===null||run(data.run));
    else if(valid)valid=run(data.run)&&(request.kind==='start'||data.run.id===request.run_id);
    if(!valid)throw new Error('Received an incomplete expedition update. Recover the saved expedition before continuing.');
    return data;
  }
  window.RiftProtocol={validate};
})();
