(function(){
 'use strict';
 const root=document.getElementById('rift-class-comparison'),left=document.getElementById('rift-compare-class-left'),right=document.getElementById('rift-compare-class-right');
 let catalog=[];
 const focus={vanguard:'Build protection, then use a charged armor-piercing finisher. Frontal perfect guards can also build charges.',berserker:'Charged finishers punish wounded targets. Low health activates passive Fury.',marksman:'Mark a target with the builder to add armor piercing to a charged finisher against that target.',beastmaster:'Equipped pets increase charged-finisher damage, capped at three pets. Aim pack shots along the target lane.',elementalist:'Hit the builder-marked target with a charged finisher to trigger extra reaction damage.',chronomancer:'Spend charges while other abilities or jump are cooling down to recover cooldown time.',oracle:'Heal yourself to build charges, even at full health, then spend them on radiant damage.',geomancer:'Build a barrier, then spend charges on a finisher with extra armor piercing.',bloodblade:'Recover health through your builder and charge spending while continuing to attack.',voidwalker:'Charged finishers spend health for damage; marking the target adds armor piercing.',runesmith:'Charge spending grants a barrier. An equipped relic also boosts charged-finisher damage.',alchemist:'Build charges and mark a target, then spend charges for healing and marked-target armor piercing.'};
 function draw(select,target){
  const item=catalog.find(item=>item.id===select.value);target.replaceChildren();if(!item)return;
  const title=document.createElement('h3');title.textContent=item.base+' · '+item.name;target.append(title);
  const list=document.createElement('dl');for(const [label,value] of [['Resource',item.resource],['Builder',item.builder],['Finisher',item.finisher],['Brawl focus',focus[item.id]||'See equipped ability details for this combat style.']]){const term=document.createElement('dt'),description=document.createElement('dd');term.textContent=label;description.textContent=value;list.append(term,description);}target.append(list);
 }
 function render(){draw(left,document.getElementById('rift-compare-class-card-left'));draw(right,document.getElementById('rift-compare-class-card-right'));}
 function init(classes,current){
  catalog=(classes||[]).flatMap(base=>(base.subclasses||[]).map(sub=>({...sub,base:base.name})));
  for(const select of [left,right]){select.replaceChildren();for(const item of catalog){const option=document.createElement('option');option.value=item.id;option.textContent=item.base+' · '+item.name;select.append(option);}}
  left.value=catalog.some(item=>item.id===current)?current:catalog[0]?.id||'';right.value=catalog.find(item=>item.id!==left.value)?.id||left.value;render();root.hidden=!catalog.length;
 }
 left.addEventListener('change',render);right.addEventListener('change',render);
 window.RiftClassCompare={init,visibility(active){root.hidden=active||!catalog.length;}};
})();
