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
  const skill=value=>object(value)&&text(value.id)&&text(value.name)&&nonnegative(value.cost)&&nonnegative(value.cooldown);
  const actor=value=>object(value)&&text(value.id)&&text(value.name)&&text(value.kind)&&finite(value.x)&&finite(value.y)&&nonnegative(value.hp)&&nonnegative(value.max_hp)&&value.max_hp>0&&finite(value.facing);
  const build=value=>object(value)&&text(value.name)&&text(value.class)&&list(value.skills,skill)&&list(value.gear,text)&&(!value.signatures||list(value.signatures,skill))&&(!value.ultimate||skill(value.ultimate));
  const drop=value=>point(value)&&text(value.id)&&nonnegative(value.gold)&&(!value.gear||object(value.gear)&&text(value.gear.Name)&&text(value.gear.Slot));
  const projectile=value=>point(value)&&finite(value.vx)&&finite(value.vy)&&text(value.kind);
  const event=value=>point(value)&&nonnegative(value.id)&&text(value.kind)&&(value.value===undefined||finite(value.value));
  const arena=value=>object(value)&&text(value.name)&&optionalList(value.obstacles,box)&&optionalList(value.hazards,h=>box(h)&&text(h.kind)&&finite(h.period)&&h.period>0&&nonnegative(h.offset)&&nonnegative(h.duration));
  function level(value){return object(value)&&Number.isInteger(value.id)&&value.id>0&&text(value.name)&&text(value.region_name)&&text(value.tactic)&&text(value.difficulty)&&Number.isInteger(value.region)&&value.region>=0&&value.region<10&&list(value.rooms,arena)&&value.rooms.length===3;}
  function run(value){
    if(!object(value))return false;
    if(value.schema!==1)throw new Error('This expedition uses an unsupported save version. Reload the page to get the current game before recovering.');
    if(!['fighting','cleared','defeated','complete','banked','expired'].includes(value.status))throw new Error('The expedition has an unrecognized state. Recover the saved expedition before continuing.');
    return text(value.id)&&value.id.length>0&&Number.isInteger(value.revision)&&value.revision>=0&&
      Number.isInteger(value.room)&&value.room>=0&&value.room<3&&typeof value.paused==='boolean'&&
      build(value.build)&&actor(value.player)&&nonnegative(value.player.mana)&&list(value.enemies,actor)&&
      list(value.projectiles,projectile)&&list(value.drops,drop)&&optionalList(value.events,event)&&
      list(value.banked_items,text)&&nonnegative(value.gold)&&nonnegative(value.banked_gold)&&nonnegative(value.clock)&&nonnegative(value.counter)&&
      object(value.skill_timers)&&Object.values(value.skill_timers).every(nonnegative)&&
      (!value.level||level(value.level))&&optionalList(value.encounter_plan,room=>list(room,actor))&&
      optionalList(value.completed_levels,id=>Number.isInteger(id)&&id>0)&&
      (!value.stats||object(value.stats)&&Object.values(value.stats).every(nonnegative));
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
