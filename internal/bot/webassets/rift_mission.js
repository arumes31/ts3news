(function(){
  'use strict';
  const $=id=>document.getElementById(id),namespace='http://www.w3.org/2000/svg';
  const section=document.createElement('section');section.className='rift-mission-planning';section.setAttribute('aria-label','Selected mission plan');
  const add=(tag,id,text,parent=section)=>{const node=document.createElement(tag);if(id)node.id=id;if(text)node.textContent=text;parent.append(node);return node;};
  add('p','rift-mission-class');
  add('p','rift-mission-difficulty');
  add('p','rift-mission-loot');
  const controls=add('div','','',section);controls.className='rift-mission-link-tools';
  const label=add('label','','Mission link',controls),link=add('input','rift-mission-link','',label);link.readOnly=true;link.type='text';
  const copy=add('button','rift-copy-mission','Copy mission link',controls);copy.type='button';
  const preference=add('label','','',controls),automatic=add('input','rift-preview-default','',preference);automatic.type='checkbox';preference.append(document.createTextNode('Open route previews by default'));
  const status=add('p','rift-mission-link-status');status.setAttribute('role','status');
  const details=add('details','rift-mission-preview');add('summary','','Three-room route and terrain',details);
  add('p','','Top-down terrain: solid blocks are low cover; outlined zones are hazards. Walk above or below cover, or jump it. Hazard timing continues during combat.',details);
  const route=add('div','rift-room-previews','',details);
  $('rift-campaign').append(section);
  try{automatic.checked=localStorage.getItem('riftPreviewOpen')==='true';}catch(_){}details.open=automatic.checked;
  automatic.onchange=()=>{details.open=automatic.checked;try{localStorage.setItem('riftPreviewOpen',String(automatic.checked));}catch(_){} };
  copy.onclick=async()=>{try{await navigator.clipboard.writeText(link.value);status.textContent='Mission link copied.';}catch(_){link.focus();link.select();status.textContent='Copy the selected mission link.';}};
  const parameters=new URLSearchParams(location.search),requested=parameters.getAll('mission');let linked=null,invalid=false,stamp='';
  function preferred(fallback,catalog){
    invalid=requested.length>0&&(requested.length!==1||!/^\d+$/.test(requested[0])||!catalog.some(level=>level.id===Number(requested[0])));
    linked=!invalid&&requested.length===1?Number(requested[0]):null;
    if(invalid)status.textContent='The mission link is invalid. Showing mission '+fallback+'.';
    return linked??fallback;
  }
  function update(level,build,active){
    if(!level)return;
    const next=JSON.stringify([level,build?.class_name,build?.class,active]);if(stamp===next)return;stamp=next;
    $('rift-mission-class').textContent='Mission '+level.id+' · '+(build?.class_name||build?.class||'Your Abyss class');
    const lootKnown=level.rooms.every(room=>typeof room.loot_rarity_ceiling==='string'&&room.loot_rarity_ceiling.length>0);
    $('rift-mission-loot').textContent=lootKnown?'Loot rarity ceiling: '+level.rooms.map((room,index)=>'Tier '+(index+1)+' — '+room.loot_rarity_ceiling).join(' · ')+'. These are upper limits, not guaranteed rarities. Bank collected loot at a checkpoint to keep it.':'Loot rarity limits were not saved with this expedition.';
    const encounters=level.rooms.map(room=>room.encounter),known=encounters.every(encounter=>encounter&&Number.isInteger(encounter.enemies)&&encounter.enemies>=0&&Number.isFinite(encounter.health_multiplier)&&Number.isFinite(encounter.damage_multiplier));
    $('rift-mission-difficulty').textContent=known?level.difficulty+' · '+Math.min(...encounters.map(encounter=>encounter.enemies))+'–'+Math.max(...encounters.map(encounter=>encounter.enemies))+' initial enemies per tier · '+encounters.reduce((sum,encounter)=>sum+encounter.enemies,0)+' across the mission. Enemy identities vary; multipliers below apply to each monster’s Brawl template before defenses and combat effects.':level.difficulty+' · Encounter estimates were not saved with this expedition.';
    const url=new URL(location.pathname,location.origin);url.searchParams.set('mission',level.id);link.value=url.href;
    if(active&&linked!==null&&linked!==level.id)status.textContent='Your saved expedition takes priority. Finish or leave it before choosing mission '+linked+'.';
    else if(!invalid)status.textContent='Share this mission without including character or account details.';
    route.replaceChildren();
    level.rooms.forEach((room,index)=>{
      const obstacles=room.obstacles||[],zones=room.hazards||[];
      const card=add('article','','',route);add('h4','','Tier '+(index+1)+' · '+room.name,card);
      if(known){const encounter=room.encounter;add('p','','Expected initial defenders: '+encounter.enemies+(index===2?' (includes the guardian)':'')+' · Health ×'+encounter.health_multiplier.toFixed(3)+' · Damage ×'+encounter.damage_multiplier.toFixed(3),card);}
      const terrainSummary=obstacles.length+' cover '+(obstacles.length===1?'block':'blocks')+' · '+zones.length+' hazard '+(zones.length===1?'zone':'zones');
      const svg=document.createElementNS(namespace,'svg');svg.setAttribute('viewBox','0 300 1600 210');svg.setAttribute('role','img');svg.setAttribute('aria-label',room.name+': '+terrainSummary);card.append(svg);
      const rectangle=(terrain,kind)=>{const rect=document.createElementNS(namespace,'rect');for(const [attribute,value] of Object.entries({x:terrain.x,y:terrain.y,width:terrain.w,height:terrain.h}))rect.setAttribute(attribute,value);rect.dataset.terrain=kind;const title=document.createElementNS(namespace,'title');title.textContent=kind==='cover'?'Low cover':terrain.kind+' hazard';rect.append(title);svg.append(rect);};
      zones.forEach(hazard=>rectangle(hazard,'hazard'));obstacles.forEach(obstacle=>rectangle(obstacle,'cover'));
      const hazards=[...new Set(zones.map(hazard=>hazard.kind))];add('p','',terrainSummary+' · '+(hazards.join(', ')||'No hazards'),card);
      const timings=[...new Set(zones.map(hazard=>hazard.kind+': '+hazard.period.toFixed(1)+'s cycle, '+hazard.duration.toFixed(1)+'s active'))];add('small','',timings.join(' · '),card);
      add('p','',index===2?'Defeat the guardian and its defenders.':'Clear the patrol, then bank and continue.',card);
    });
  }
  window.RiftMission={preferred,update};
})();
