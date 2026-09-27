(function(){
  'use strict';
  const root=document.getElementById('rift-minimap'),svg=root.querySelector('svg'),copy=root.querySelector('p'),ns='http://www.w3.org/2000/svg';
  const progress=document.createElement('ol');progress.id='rift-minimap-progress';progress.setAttribute('aria-label','Current mission tier progress');root.append(progress);
  let progressKey='';
  function cleared(run,index){
    if(run.practice||index>run.room)return false;
    return index<run.room||['cleared','complete','banked'].includes(run.status)||(Number.isFinite(run.room_splits?.[index])&&run.room_splits[index]>=0);
  }
  function tierProgress(run){
    progress.hidden=!!run.practice;if(progress.hidden)return;
    const tiers=(run.level?.rooms||[]).map((room,index)=>({name:room.name,current:index===run.room,state:cleared(run,index)?'cleared':index>run.room?'ahead':['defeated','expired'].includes(run.status)?'ended':'current'}));
    const key=JSON.stringify([run.level?.id,tiers]);if(key===progressKey)return;progressKey=key;
    const labels={cleared:'✓ Cleared',ahead:'Ahead',current:'Current',ended:'Not cleared'};
    progress.replaceChildren(...tiers.map((tier,index)=>{const item=document.createElement('li'),title=document.createElement('strong'),status=document.createElement('span');item.dataset.state=tier.state;if(tier.current)item.setAttribute('aria-current','step');title.textContent='Tier '+(index+1);status.textContent=labels[tier.state];item.append(title,status);item.setAttribute('aria-label','Tier '+(index+1)+': '+tier.name+' · '+labels[tier.state]+(tier.current?' · current location':''));return item;}));
  }
  const mx=x=>Math.round(x*.2*10)/10,my=y=>Math.round((6+(y-315)*.24)*10)/10;
  function node(tag,attrs,label){const el=document.createElementNS(ns,tag);for(const [key,value] of Object.entries(attrs))el.setAttribute(key,String(value));if(label){const title=document.createElementNS(ns,'title');title.textContent=label;el.append(title);}return el;}
  function update(run){
    const arena=run?.practice?.arena||run?.level?.rooms?.[run.room];root.hidden=!arena;if(!arena)return;
    tierProgress(run);
    const currentCleared=cleared(run,run.room),drawing=document.createDocumentFragment();
    drawing.append(node('rect',{x:7,y:6,width:306,height:42,class:'map-floor','data-cleared':currentCleared},currentCleared?'Cleared arena':'Walkable arena bounds'));
    const rect=(o,kind,label,phase)=>drawing.append(node('rect',{x:mx(o.x),y:my(o.y),width:mx(o.w),height:Math.round(o.h*.24*10)/10,'data-kind':kind,...(phase?{'data-phase':phase}:{})},label));
    for(const p of arena.platforms||[])rect(p,'platform','Raised platform with sloped edges');
    for(const o of arena.obstacles||[])rect(o,'cover','Low cover');
    for(const o of arena.high_cover||[])rect(o,'stone','Tall cover');
    for(const o of arena.cover||[])if(o.material==='stone'||o.hp>0)rect(o,o.material,o.material==='wood'?'Breakable wood':'Permanent stone');
    for(const edge of arena.drop_edges||[])rect({...edge,h:edge.landing_y-edge.y},'ledge','One-way descent; return around an end');
    if(arena.hazard_switch){const s=arena.hazard_switch;rect({x:s.x-10,y:s.y-10,w:20,h:20},'switch',s.used?'Hazards off':'Hazard switch: hold guard nearby');}
    let active=0;
    for(const h of arena.hazards||[]){
      const phase=(run.clock+h.offset)%h.period,state=h.disabled||run.status!=='fighting'?'off':phase<1.2?'warning':phase<1.2+h.duration?'active':'safe';if(state==='active')active++;
      let box=h;
      if(state==='active'&&h.kind==='sweeping_flame'){const width=Math.min(36,h.w);box={...h,x:h.x+(h.w-width)*Math.max(0,Math.min(1,(phase-1.2)/h.duration)),w:width};}
      if(state==='active'&&h.kind==='rotating_blade'){const width=Math.min(20,h.w),height=Math.min(20,h.h),angle=Math.max(0,Math.min(1,(phase-1.2)/h.duration))*Math.PI*2;box={...h,x:h.x+(h.w-width)/2*(1+Math.cos(angle)),y:h.y+(h.h-height)/2*(1+Math.sin(angle)),w:width,h:height};}
      if(state==='active'&&h.kind==='moving_poison'){const width=Math.min(80,h.w),progress=Math.max(0,Math.min(1,(phase-1.2)/h.duration));box={...h,x:h.x+(h.w-width)*(1-Math.abs(2*progress-1)),w:width};}
      rect(box,'hazard',h.kind+' · '+state,state);
    }
    const actors=(run.enemies||[]).filter(e=>e.hp>0),enemies=actors.filter(e=>!['totem','generator','cage'].includes(e.kind));
    for(const e of actors.filter(e=>['totem','generator','cage'].includes(e.kind)))drawing.append(node('rect',{x:mx(e.x)-2,y:my(e.y)-2,width:4,height:4,'data-kind':'objective'},e.name||'Objective prop'));
    for(const e of enemies)drawing.append(node('path',{d:'M -2.5 2 L 0 -3 L 2.5 2 Z',transform:'translate('+mx(e.x)+' '+my(e.y)+')','data-kind':'enemy'},e.name||'Enemy'));
    const goal=run.room_objective,exit=goal&&['escape_collapse','carry_relic','escort_spirit'].includes(goal.kind)?goal.zone:null;
    if(exit)drawing.append(node('path',{d:'M -3 -4 L 3 -4 L 3 4 L -3 4 Z M -1 0 L 5 0 M 3 -2 L 5 0 L 3 2',transform:'translate('+mx(exit.x)+' '+my(exit.y)+')','data-kind':'exit'},'Exit · '+(goal.complete?'objective complete':'reach the marked destination')));
    drawing.append(node('circle',{cx:mx(run.player.x),cy:my(run.player.y),r:2.6,'data-kind':'player'},'You'));
    svg.replaceChildren(drawing);
    const guidance=run.practice?'Practice arena: no checkpoint banking.':exit?'Exit marked on map.':run.status==='cleared'?'Bank & leave is available from anywhere.':['complete','banked','defeated','expired'].includes(run.status)?'Expedition finished.':'Clear the tier, then bank and leave from anywhere.';
    svg.setAttribute('aria-label','Current arena: '+arena.name+(currentCleared?' · cleared. ':'. ')+enemies.length+' enemies, '+active+' active hazards. Player '+Math.round(run.player.x)+', '+Math.round(run.player.y)+(exit?'. Exit '+Math.round(exit.x)+', '+Math.round(exit.y):'. '+guidance));
    const title=document.createElement('strong');title.textContent=arena.name;
    copy.replaceChildren(title,document.createElement('br'),document.createTextNode('○ You · △ Enemies · Red zones: active hazards. '+guidance));
  }
  window.RiftMinimap={update};
})();
