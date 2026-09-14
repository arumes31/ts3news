(function(){
  'use strict';
  const captions=document.createElement('section');captions.id='rift-captions';captions.hidden=true;captions.setAttribute('aria-label','Recent combat cues');
  const list=document.createElement('ul'),live=document.createElement('p');live.className='rift-visually-hidden';live.setAttribute('role','status');live.setAttribute('aria-live','polite');captions.append(list,live);document.querySelector('.rift-actionbar').before(captions);
  const label=document.createElement('label'),checkbox=document.createElement('input');checkbox.type='checkbox';checkbox.id='rift-combat-captions';label.append(checkbox,document.createTextNode('Captions for important combat cues'));document.querySelector('.rift-settings').append(label);
  try{const saved=JSON.parse(localStorage.getItem('riftCaptions'));checkbox.checked=saved?.version===1&&saved.enabled===true;}catch(_){}
  let previous=null,seen=0,lastLow=-Infinity,ultimateArmed=false,finisherArmed=false,items=[],expiry=0;
  const labels={boss_roar:'Boss roar',slam:'Ground slam',boss_death:'Boss defeated',clear:'Room secured',defeat:'Expedition ended',pickup:'Loot collected',ultimate:'Ultimate unleashed'};
  function paint(){
    clearTimeout(expiry);const now=performance.now();items=items.filter(item=>item.until>now);list.replaceChildren();
    for(const item of items){const li=document.createElement('li');li.textContent=item.text;list.append(li);}
    captions.hidden=!checkbox.checked||items.length===0;
    if(items.length)expiry=setTimeout(paint,Math.max(1,Math.min(...items.map(item=>item.until))-now));
  }
  function clear(){items=[];live.textContent='';paint();}
  checkbox.onchange=()=>{try{localStorage.setItem('riftCaptions',JSON.stringify({version:1,enabled:checkbox.checked}));}catch(_){}clear();};
  function update(run,replay,playing){
    const identity=[run.id,run.level?.id,run.room].join(':'),p=run.player;
    const eligible=skill=>!!skill&&p.mana>=skill.cost&&!(run.skill_timers[skill.id]>0);
    const ultimate=eligible(run.build.ultimate),finisher=eligible(run.build.signatures?.find(skill=>skill.role==='finisher'))&&run.resource>0;
    const fresh=!previous||previous.identity!==identity;
    if(fresh||replay||!playing||run.paused){previous={identity,hp:p.hp};seen=run.counter;if(fresh)lastLow=-Infinity;ultimateArmed=!ultimate;finisherArmed=!finisher;clear();return;}
    const announcements=[];
    function cue(kind,text,sound){
      if(sound)window.RiftAudio.play(kind,0);
      if(!checkbox.checked)return;
      const now=performance.now();if(items.some(item=>item.kind===kind&&item.until>now))return;
      items.push({kind,text,until:now+4000});items=items.slice(-3);announcements.push(text);
    }
    for(const event of run.events||[]){if(event.id<=seen)continue;seen=event.id;if(labels[event.kind])cue(event.kind,labels[event.kind],false);}
    seen=Math.max(seen,run.counter);
    if(!ultimate)ultimateArmed=true;if(!finisher)finisherArmed=true;
    if(run.status==='fighting'&&p.hp>0){
      if(p.hp<=p.max_hp*.25&&p.hp<previous.hp&&run.clock-lastLow>=12){lastLow=run.clock;cue('low_health','Low health (25% or less)',true);}
      if(!(p.cooldown>0||p.knockdown>0)){
        if(ultimate&&ultimateArmed){ultimateArmed=false;cue('ultimate_ready','Ultimate ready',true);}
        if(finisher&&finisherArmed){finisherArmed=false;cue('finisher_ready','Charged finisher ready',true);}
      }
    }
    previous={identity,hp:p.hp};
    if(announcements.length){paint();live.textContent=announcements.slice(-3).join('. ')+'.';}
  }
  window.RiftFeedback={update};
})();
