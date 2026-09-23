(function(){
  'use strict';
  const $=id=>document.getElementById(id),namespace='http://www.w3.org/2000/svg';
  const section=document.createElement('section');section.className='rift-mission-planning';section.setAttribute('aria-label','Selected mission plan');
  const add=(tag,id,text,parent=section)=>{const node=document.createElement(tag);if(id)node.id=id;if(text)node.textContent=text;parent.append(node);return node;};
  add('p','rift-mission-class');
  add('p','rift-mission-difficulty');
  add('p','rift-mission-loot');
  const challengeStatus=add('p','rift-mission-challenge');
  const controls=add('div','','',section);controls.className='rift-mission-link-tools';
  const label=add('label','','Mission link',controls),link=add('input','rift-mission-link','',label);link.readOnly=true;link.type='text';label.htmlFor=link.id;
  const copy=add('button','rift-copy-mission','Copy mission link',controls);copy.type='button';
  const preference=add('label','','',controls),automatic=add('input','rift-preview-default','',preference);automatic.type='checkbox';preference.htmlFor=automatic.id;preference.append(document.createTextNode('Open route previews by default'));
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
  function update(level,build,active,challenge){
    if(!level)return;
    const next=JSON.stringify([level,build?.class_name,build?.class,active,challenge?.key]);if(stamp===next)return;stamp=next;
    $('rift-mission-class').textContent='Mission '+level.id+' · '+(build?.class_name||build?.class||'Your Abyss class');
    const lootKnown=level.rooms.every(room=>typeof room.loot_rarity_ceiling==='string'&&room.loot_rarity_ceiling.length>0);
    $('rift-mission-loot').textContent=lootKnown?'Loot rarity ceiling: '+level.rooms.map((room,index)=>'Tier '+(index+1)+' — '+room.loot_rarity_ceiling).join(' · ')+'. These are upper limits, not guaranteed rarities. Bank collected loot at a checkpoint to keep it.':'Loot rarity limits were not saved with this expedition.';
    const encounters=level.rooms.map(room=>room.encounter),known=encounters.every(encounter=>encounter&&Number.isInteger(encounter.enemies)&&encounter.enemies>=0&&Number.isFinite(encounter.health_multiplier)&&Number.isFinite(encounter.damage_multiplier));
    $('rift-mission-difficulty').textContent=known?level.difficulty+' · '+Math.min(...encounters.map(encounter=>encounter.enemies))+'–'+Math.max(...encounters.map(encounter=>encounter.enemies))+' planned enemies per tier · '+encounters.reduce((sum,encounter)=>sum+encounter.enemies,0)+' across the mission. Enemy identities vary; multipliers below apply to each monster’s Brawl template before defenses and combat effects.':level.difficulty+' · Encounter estimates were not saved with this expedition.';
    if(challenge){
      challengeStatus.hidden=false;
      challengeStatus.replaceChildren();
      const compatible=window.RiftCampaignTools?window.RiftCampaignTools.isChallengeCompatible(level,challenge):false;
      const statusSpan=document.createElement('span');
      statusSpan.className='rift-challenge-status '+(compatible?'compatible':'incompatible');
      statusSpan.textContent=compatible?'✓ Compatible':'✗ Incompatible';
      challengeStatus.append(
        document.createTextNode('Active challenge: '+challenge.label+' ('+challenge.criteria+') — '),
        statusSpan
      );
    }else{
      challengeStatus.hidden=true;
    }
    const url=new URL(location.pathname,location.origin);url.searchParams.set('mission',level.id);link.value=url.href;
    if(active&&linked!==null&&linked!==level.id)status.textContent='Your saved expedition takes priority. Finish or leave it before choosing mission '+linked+'.';
    else if(!invalid)status.textContent='Share this mission without including character or account details.';
    route.replaceChildren();
    level.rooms.forEach((room,index)=>{
      const obstacles=room.obstacles||[],zones=room.hazards||[];
      const card=add('article','','',route);add('h4','','Tier '+(index+1)+' · '+room.name,card);
      if(known){const encounter=room.encounter;add('p','','Planned defenders: '+encounter.enemies+(room.objective==='survive_waves'?' across three waves':index===2?' (includes the guardian)':'')+' · Health ×'+encounter.health_multiplier.toFixed(3)+' · Damage ×'+encounter.damage_multiplier.toFixed(3),card);}
      const terrainSummary=obstacles.length+' cover '+(obstacles.length===1?'block':'blocks')+' · '+zones.length+' hazard '+(zones.length===1?'zone':'zones');
      const svg=document.createElementNS(namespace,'svg');svg.setAttribute('viewBox','0 300 1600 210');svg.setAttribute('role','img');svg.setAttribute('aria-label',room.name+': '+terrainSummary);card.append(svg);
      const rectangle=(terrain,kind)=>{const rect=document.createElementNS(namespace,'rect');for(const [attribute,value] of Object.entries({x:terrain.x,y:terrain.y,width:terrain.w,height:terrain.h}))rect.setAttribute(attribute,value);rect.dataset.terrain=kind;const title=document.createElementNS(namespace,'title');title.textContent=kind==='cover'?'Low cover':terrain.kind+' hazard';rect.append(title);svg.append(rect);};
      zones.forEach(hazard=>rectangle(hazard,'hazard'));obstacles.forEach(obstacle=>rectangle(obstacle,'cover'));
      const hazards=[...new Set(zones.map(hazard=>hazard.kind))];add('p','',terrainSummary+' · '+(hazards.join(', ')||'No hazards'),card);
      const timings=[...new Set(zones.map(hazard=>hazard.kind+': '+hazard.period.toFixed(1)+'s cycle, '+hazard.duration.toFixed(1)+'s active'))];add('small','',timings.join(' · '),card);
      const objectives={
        rune_gate:'Clear the patrol, then step on rune seals in the displayed order to open the gate. Wrong seals reset the sequence without damage. Floor hazards switch off for the puzzle.',
        split_defense:'Defend upper and lower lane wards. Intercept approaching enemies; nearby fighters draw them into combat. Both wards must survive until the patrol is defeated.',
        protect_lantern:'Keep enemies outside the lantern ring or defeat them. Nearby enemies drain its light; if the lantern goes out, the expedition fails. Clear the patrol to protect it.',
        rescue_companions:'Break both cages with attacks or spells to rescue the captive spirits, then clear the patrol. Cages grant no monster kills or loot.',
        linked_guardians:'Linked guardians take 50% less damage within 240 units of each other. Separate them or defeat one to break the bond, then defeat both guardians and the patrol.',
        escape_collapse:'Escape through the exit seal before the collapse catches you. It advances after three seconds and damages fighters behind its edge. Surviving enemies grant no rewards.',
        interrupt_ritual:'Hit channelers to reset their eight-second charge. Leave their pulse rings before discharge. Defeat all channelers and the remaining patrol.',
        escort_spirit:'Stay near the spirit and clear nearby threats so it reaches the exit. It waits when left behind or threatened. Defeat the patrol before banking.',
        moving_beacons:'Capture three moving beacons by staying grounded inside each ring for three seconds, then defeat the patrol. Leaving keeps capture progress.',
        sigils:'Collect all three sigils and defeat the patrol before banking.',
        hold_circle:'Hold the circle for 15 uncontested seconds, then defeat the patrol. Leaving keeps your progress.',
        survive_waves:'Defeat three waves. Reinforcements arrive after a short warning; bank only after the final wave.',
        destroy_totems:'Destroy all three ritual totems and defeat the patrol. Totems grant no monster kills or loot.',
        carry_relic:'Carry the relic to the exit seal, then defeat the patrol. Movement is 30% slower while carrying; attacks and jumps remain available.',
        disable_generators:'Destroy the generators to permanently disable their linked floor hazards, then defeat the patrol. Generators grant no monster kills or loot.',
        marked_hunt:'Defeat the marked targets to secure the tier. Surviving unmarked enemies retreat without granting kills or loot.'
      };
      add('p','',objectives[room.objective]||(index===2?'Defeat the guardian and its defenders.':'Clear the patrol, then bank and continue.'),card);
    });
  }
  window.RiftMission={preferred,update};
})();
