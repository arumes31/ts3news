(function(){
  'use strict';
  const captions=document.createElement('section');captions.id='rift-captions';captions.hidden=true;captions.setAttribute('aria-label','Recent combat cues');
  const list=document.createElement('ul'),live=document.createElement('p');live.className='rift-visually-hidden';live.setAttribute('role','status');live.setAttribute('aria-live','polite');captions.append(list,live);document.querySelector('.rift-actionbar').before(captions);
  const label=document.createElement('label'),checkbox=document.createElement('input');checkbox.type='checkbox';checkbox.id='rift-combat-captions';label.htmlFor='rift-combat-captions';label.append(checkbox,document.createTextNode('Captions for important combat cues'));document.querySelector('.rift-settings').append(label);
  try{const saved=JSON.parse(localStorage.getItem('riftCaptions'));checkbox.checked=saved?.version===1&&saved.enabled===true;}catch(_){}
  const dirLabel=document.createElement('label'),dirCheckbox=document.createElement('input');dirCheckbox.type='checkbox';dirCheckbox.id='rift-directional-threat-captions';dirLabel.htmlFor='rift-directional-threat-captions';dirLabel.append(dirCheckbox,document.createTextNode('Directional captions for off-screen threats'));document.querySelector('.rift-settings').append(dirLabel);
  try{const savedDir=JSON.parse(localStorage.getItem('riftDirectionalCaptions'));dirCheckbox.checked=savedDir?.version===1&&savedDir.enabled===true;}catch(_){}
  let previous=null,seen=0,lastLow=-Infinity,ultimateArmed=false,finisherArmed=false,items=[],expiry=0,lastThreatLeftTime=0,lastThreatLeftCount=0,lastThreatRightTime=0,lastThreatRightCount=0;
  const labels={boss_roar:'Boss roar',slam:'Ground slam',boss_death:'Boss defeated',treasure_death:'Treasure goblin defeated',clear:'Room secured',defeat:'Expedition ended',pickup:'Loot collected',ultimate:'Ultimate unleashed',block:'Attack guarded',perfect_guard:'Perfect guard!',empty_mana:'Not enough mana',cooldown_rejection:'Ability on cooldown',dodge:'Dodged!',hazard_warning:'Hazard charging',hazard_deactivation:'Hazard cleared',treasure_escape:'Treasure goblin escaped',rare_item:'Rare item discovered',rare_discovery:'Rare item discovered',land_heavy:'Hard impact landing'};
  function paint(){
    clearTimeout(expiry);const now=performance.now();items=items.filter(item=>item.until>now);list.replaceChildren();
    for(const item of items){const li=document.createElement('li');li.textContent=item.text;list.append(li);}
    captions.hidden=(!checkbox.checked&&!dirCheckbox.checked)||items.length===0;
    if(items.length)expiry=setTimeout(paint,Math.max(1,Math.min(...items.map(item=>item.until))-now));
  }
  function clear(){items=[];live.textContent='';paint();}
  checkbox.onchange=()=>{try{localStorage.setItem('riftCaptions',JSON.stringify({version:1,enabled:checkbox.checked}));}catch(_){}clear();};
  dirCheckbox.onchange=()=>{try{localStorage.setItem('riftDirectionalCaptions',JSON.stringify({version:1,enabled:dirCheckbox.checked}));}catch(_){}clear();};
  function update(run,replay,playing){
    const identity=[run.id,run.level?.id,run.room].join(':'),p=run.player;
    const eligible=skill=>!!skill&&p.mana>=skill.cost&&!(run.skill_timers[skill.id]>0);
    const ultimate=eligible(run.build.ultimate),finisher=eligible(run.build.signatures?.find(skill=>skill.role==='finisher'))&&run.resource>0;
    const fresh=!previous||previous.identity!==identity;
    if(fresh||replay||!playing||run.paused){
      previous={identity,hp:p.hp};seen=run.counter;if(fresh)lastLow=-Infinity;ultimateArmed=!ultimate;finisherArmed=!finisher;
      lastThreatLeftTime=0;lastThreatLeftCount=0;lastThreatRightTime=0;lastThreatRightCount=0;clear();return;
    }
    const announcements=[];
    function cue(kind,text,sound,isDirectional=false){
      if(sound)window.RiftAudio.play(kind,0);
      if(!checkbox.checked&&!(dirCheckbox.checked&&isDirectional))return;
      const now=performance.now();if(items.some(item=>item.kind===kind&&item.until>now))return;
      items.push({kind,text,until:now+4000});items=items.slice(-3);announcements.push(text);
    }
    const cam=Math.max(0,Math.min(640,(p?.x||0)-350));
    for(const event of run.events||[]){
      if(event.id<=seen)continue;
      seen=event.id;
      if(labels[event.kind] || (event.kind==='land'&&event.value>=0.85)){
        let baseLabel=event.kind==='land' ? 'Hard impact landing' : labels[event.kind];
        if(event.kind==='boss_death'&&event.actor_name)baseLabel=event.actor_name+' defeated';
        if(event.kind==='rare_item'||event.kind==='rare_discovery'){
          if(event.value>=4)baseLabel='Legendary item discovered';
          else if(event.value>=3)baseLabel='Epic item discovered';
        }
        let text=baseLabel,isDir=false;
        if(dirCheckbox.checked&&Number.isFinite(event.x)){
          if(event.x<cam){text='← '+baseLabel+' (off-screen left)';isDir=true;}
          else if(event.x>cam+960){text=baseLabel+' (off-screen right) →';isDir=true;}
        }
        let cueKey = isDir ? (event.kind==='land'?'land_heavy':event.kind)+'_dir' : (event.kind==='land'?'land_heavy':event.kind);
        if(event.kind==='rare_item'||event.kind==='rare_discovery')cueKey+='_'+Math.round(event.value||0);
        if(event.kind==='boss_death')cueKey+='_'+event.id;
        cue(cueKey,text,false,isDir);
      }
    }
    seen=Math.max(seen,run.counter);
    if(!ultimate)ultimateArmed=true;if(!finisher)finisherArmed=true;
    if(run.status==='fighting'&&p.hp>0){
      if(p.hp<=p.max_hp*.25&&p.hp<previous.hp&&run.clock-lastLow>=12){lastLow=run.clock;cue('low_health','Low health (25% or less)',true);}
      if(!(p.cooldown>0||p.knockdown>0)){
        if(ultimate&&ultimateArmed){ultimateArmed=false;cue('ultimate_ready','Ultimate ready',true);}
        if(finisher&&finisherArmed){finisherArmed=false;cue('finisher_ready','Charged finisher ready',true);}
      }
      if(dirCheckbox.checked){
        const living=(run.enemies||[]).filter(e=>e.hp>0);
        const offLeft=living.filter(e=>e.x<cam),offRight=living.filter(e=>e.x>cam+960);
        const now=performance.now();
        if(offLeft.length>0&&(!lastThreatLeftTime||now-lastThreatLeftTime>=6000||offLeft.length>lastThreatLeftCount)){
          lastThreatLeftTime=now;lastThreatLeftCount=offLeft.length;
          const boss=offLeft.some(e=>e.kind==='boss');
          const text=boss?'← Boss threat (off-screen left)':'← '+offLeft.length+' '+(offLeft.length===1?'threat':'threats')+' (off-screen left)';
          cue('threat_left',text,false,true);
        }else if(offLeft.length===0){
          lastThreatLeftCount=0;
        }
        if(offRight.length>0&&(!lastThreatRightTime||now-lastThreatRightTime>=6000||offRight.length>lastThreatRightCount)){
          lastThreatRightTime=now;lastThreatRightCount=offRight.length;
          const boss=offRight.some(e=>e.kind==='boss');
          const text=boss?'Boss threat (off-screen right) →':offRight.length+' '+(offRight.length===1?'threat':'threats')+' (off-screen right) →';
          cue('threat_right',text,false,true);
        }else if(offRight.length===0){
          lastThreatRightCount=0;
        }
      }
    }
    previous={identity,hp:p.hp};
    if(announcements.length){paint();live.textContent=announcements.slice(-3).join('. ')+'.';}
  }
  window.RiftFeedback={update};
})();
