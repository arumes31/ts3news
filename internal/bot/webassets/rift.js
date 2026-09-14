(function () {
  'use strict';
  const $ = id => document.getElementById(id), root = $('rift-app'), audio = window.RiftAudio, renderer = window.RiftRenderer;
  const keys = new Set(), touch = new Set(), taps = new Set();
  let starting = false, startIntent = 0, checkpointPending = false;
  let run = null, build = null, rooms = [], playing = false, busy = false, ready = false, timer = 0, currentSkillIDs = '';
  let levels = [], selectedLevel = 1, campaignKey = '', clearedAt = 0;
  try { $('rift-auto').checked = localStorage.getItem('rift-auto') !== 'false'; } catch (_) {}
  const api = '/api/abyss/rift';
  const status = message => { $('rift-status').textContent = message; };
  function silence(){try{Promise.resolve(audio.setActive(false)).catch(()=>{});}catch(_){} }
  function text(tag, value, parent, className) { const node = document.createElement(tag); node.textContent = value; if(className)node.className=className; if(parent)parent.append(node); return node; }
  function message(title, copy, button, kicker) {
    $('rift-overlay').hidden = false; $('rift-overlay-title').textContent = title; $('rift-overlay-copy').textContent = copy;
    $('rift-overlay-kicker').textContent = kicker || 'MOSSBOUND RUINS'; $('rift-start').textContent = button; $('rift-start').disabled = !ready || busy;
  }
  async function request(method, body) {
    const controller = new AbortController(), timeout = setTimeout(() => controller.abort(), 10000);
    try {
      const response = await fetch(api,{method,credentials:'same-origin',cache:'no-store',headers:body?{'Content-Type':'application/json'}:undefined,body:body?JSON.stringify(body):undefined,signal:controller.signal});
      if(response.status===401)throw new Error('Your session expired. Sign in again, then resume this expedition.');
      if(response.status===409)throw new Error('The saved expedition changed. Recover it before continuing.');
      if(!response.ok)throw new Error('Connection interrupted. Recover the saved expedition before continuing.');
      let data;try{data=await response.json();}catch(_){throw new Error('The expedition response was interrupted. Recover the saved expedition before continuing.');}
      if(data?.ok===false)throw new Error(typeof data.error==='string'?data.error:'Could not confirm the expedition.');return window.RiftProtocol.validate(data,method,body);
    } finally { clearTimeout(timeout); }
  }
  function input() {
    const held = name => touch.has(name)||taps.has(name);
    const pressed = code => keys.has(code)||taps.has(code);
    const value = {x:Number(keys.has('KeyD')||keys.has('ArrowRight')||held('right'))-Number(keys.has('KeyA')||keys.has('ArrowLeft')||held('left')),
      y:Number(keys.has('KeyS')||keys.has('ArrowDown')||held('down'))-Number(keys.has('KeyW')||keys.has('ArrowUp')||held('up')),
      attack:pressed('KeyJ')||held('attack'),guard:pressed('KeyL')||held('guard'),jump:pressed('KeyK')||pressed('Space')||held('jump'),
      skill:(run?.build.signatures||[]).find((s,i)=>pressed(i?'KeyE':'KeyQ')||held(s.id))?.id || (run?.build.ultimate && (pressed('KeyR')||held(run.build.ultimate.id)) ? run.build.ultimate.id : '') || (run?.build.skills||[]).find((s,i)=>pressed('Digit'+(i+1))||held(s.id))?.id||''};
    taps.clear();return value;
  }
  function resetInput(){keys.clear();touch.clear();taps.clear();root.querySelectorAll('.rift-held').forEach(n=>n.classList.remove('rift-held'));}
  function update(value, replay) {
    if(!value)return;
    if(value.status !== 'cleared' || replay) clearedAt = 0;
    else if(!clearedAt) clearedAt = performance.now();
    run=value;renderer.snapshot(run,replay);
    const controlsEnabled=playing&&['fighting','cleared'].includes(run.status)&&!run.paused;
    if(run.level){if(['fighting','cleared'].includes(run.status))selectedLevel=run.level.id;rooms=run.level.rooms.map(room=>room.name);}
    updateCampaign();
    $('rift-transition').hidden=run.status!=='cleared'||!playing||!$('rift-auto').checked;
    $('rift-vitals').hidden=false;$('rift-name').textContent=run.build.name;$('rift-class').textContent=run.build.class;
    $('rift-style').textContent=run.build.class_name||run.build.class;
    $('rift-resource').textContent=run.build.resource ? run.build.resource+' '+(run.resource||0)+'/3'+(run.barrier>0?' · Barrier '+Math.ceil(run.barrier):'') : 'Class abilities unlock through Abyss progression';
    $('rift-hp').textContent=Math.ceil(run.player.hp)+' / '+Math.ceil(run.player.max_hp);
    $('rift-hp-fill').style.width=Math.max(0,100*run.player.hp/run.player.max_hp)+'%';
    $('rift-mana').textContent=Math.floor(run.player.mana)+' MP';$('rift-mana-fill').style.width=run.player.mana+'%';
    $('rift-room').textContent='Mission '+(run.level?.id||1)+' · Tier '+(run.room+1)+'/3 · '+(rooms[run.room]||'Mossbound Ruins');
    const boss=run.enemies.find(e=>e.kind==='boss'&&e.hp>0);$('rift-boss').hidden=!boss;if(boss){$('rift-boss-fill').style.width=100*boss.hp/boss.max_hp+'%';$('rift-boss-name').textContent=boss.name;}
    const finalBoss=run.encounter_plan?.[2]?.find(e=>e.kind==='boss')||run.enemies.find(e=>e.kind==='boss');
    $('rift-route-boss').textContent=finalBoss?'Defeat '+finalBoss.name:'Defeat an Abyss boss';
    window.RiftLoot.update(run);
    root.querySelectorAll('.rift-route li').forEach((li,i)=>{li.classList.toggle('current',i===run.room);li.classList.toggle('done',i<run.room);});
    const signature=run.build.skills.map(s=>s.id).join(',');
    if(signature!==currentSkillIDs||!$('rift-skills').childElementCount){
      currentSkillIDs=signature;$('rift-skills').replaceChildren();run.build.skills.forEach((s,i)=>{const btn=document.createElement('button');btn.type='button';btn.dataset.hold=s.id;btn.title=s.name+' · '+s.cost+' MP · '+s.cooldown+'s cooldown';btn.setAttribute('aria-label',s.name);text('span','',btn,'rift-skill-icon');text('kbd',String(i+1),btn);text('span',s.name,btn);text('small','Ready',btn);$('rift-skills').append(btn);hold(btn,s.id);});
    }
    [...$('rift-skills').children].forEach((btn,i)=>{const s=run.build.skills[i],remaining=run.skill_timers[s.id]||0;btn.querySelector('small').textContent=remaining>0?remaining.toFixed(1)+'s':s.cost+' MP';btn.disabled=!controlsEnabled||remaining>0||run.player.mana<s.cost;});
    const specials=[...(run.build.signatures||[]),...(run.build.ultimate?[run.build.ultimate]:[])];
    const specialIDs=specials.map(s=>s.id).join(',');
    if($('rift-signatures').dataset.ids!==specialIDs){$('rift-signatures').dataset.ids=specialIDs;$('rift-signatures').replaceChildren();specials.forEach((s,i)=>{const btn=document.createElement('button');btn.type='button';btn.title=s.name+' · '+s.cost+' MP';text('kbd',s===run.build.ultimate?'R':i?'E':'Q',btn);text('span',s.name,btn);text('small','Ready',btn);$('rift-signatures').append(btn);hold(btn,s.id);});}
    [...$('rift-signatures').children].forEach((btn,i)=>{const s=specials[i],remaining=run.skill_timers[s.id]||0;btn.querySelector('small').textContent=remaining>0?remaining.toFixed(1)+'s':s.cost+' MP';btn.disabled=!controlsEnabled||remaining>0||run.player.mana<s.cost;});
    window.RiftHUD.update(run,playing);
    $('rift-room-actions').hidden=run.status!=='cleared'||!playing;
    $('rift-clear-label').textContent=run.room===2?(finalBoss?.name||'The boss')+' has fallen':'Area secured';
    $('rift-next').textContent=$('rift-auto').checked?'Continue now →':run.room===2?'Bank & finish expedition':'Bank & continue →';
    $('rift-pause').disabled=!playing&&run.status!=='fighting'&&run.status!=='cleared';
    $('rift-pause').textContent=playing?'Pause · Esc':'Resume · Esc';
    root.querySelectorAll('#rift-loadout select').forEach(el=>el.disabled=run.status==='fighting'||run.status==='cleared');
    if(['defeated','complete','banked','expired'].includes(run.status)){
      playing=false;clearTimeout(timer);resetInput();$('rift-room-actions').hidden=true;
      const lost=run.status==='defeated';
      message(lost?'The rift takes its toll.':'Returned from the ruins.',lost?'Unbanked finds were lost. Your equipped gear and banked rewards are safe.':run.banked_gold.toLocaleString()+' gold and '+run.banked_items.length.toLocaleString()+' Abyss '+(run.banked_items.length===1?'item':'items')+' safely in your inventory.','Enter a new expedition',lost?'EXPEDITION ENDED':'REWARDS SECURED');
      if(run.status==='expired')message('A new chapter begins.','This expedition belongs to an earlier economy. Start a fresh run with your current character.','Enter a new expedition','EXPEDITION EXPIRED');
      root.querySelectorAll('#rift-loadout select').forEach(el=>el.disabled=false);
      setTimeout(()=>{if(!playing)silence();},1500);
    }
  }
  async function send(kind) {
    if(busy)return false;
    busy=true;
    const body={kind,run_id:run?.id||'',request_id:crypto.randomUUID(),revision:(run?.revision||0)+1,input:kind==='step'?input():{}};
    if(kind==='start'){body.level_id=selectedLevel;body.skills=[...root.querySelectorAll('#rift-loadout select')].map(el=>el.value).filter(Boolean);}
    root.querySelectorAll(kind==='step'?'#rift-start':'#rift-next,#rift-exit,#rift-start').forEach(btn=>btn.disabled=true);
    try {
      const data=await request('POST',body);update(data.run,false);
      if(['bank','exit','next','advance'].includes(kind))audio.play('bank',0);
      return true;
    } catch(error){
      playing=false;resetInput();clearTimeout(timer);silence();if(run)update(run,true);status(error.message);
      message('Your expedition is saved.',error.message,'Recover expedition','CONNECTION PAUSED');$('rift-start').dataset.recover='true';return false;
    } finally {busy=false;$('rift-start').disabled=!ready;root.querySelectorAll('#rift-next,#rift-exit').forEach(btn=>btn.disabled=!ready||checkpointPending);}
  }
  async function checkpoint(kind){
    if(checkpointPending||!playing||run?.status!=='cleared')return false;
    const identity=run.id,room=run.room,mission=run.level?.id;
    checkpointPending=true;root.querySelectorAll('#rift-next,#rift-exit').forEach(btn=>btn.disabled=true);
    try{
      while(busy)await new Promise(resolve=>setTimeout(resolve,20));
      if(!playing||document.hidden||run?.id!==identity||run.status!=='cleared'||run.room!==room||run.level?.id!==mission)return false;
      return await send(kind);
    }finally{checkpointPending=false;root.querySelectorAll('#rift-next,#rift-exit').forEach(btn=>btn.disabled=!ready);}
  }
  async function loop(){
    if(!playing)return;
    if(!busy&&!checkpointPending){
      if(run?.status==='cleared' && $('rift-auto').checked){
        if(!clearedAt)clearedAt=performance.now();
        const remaining=Math.max(0,1.2-(performance.now()-clearedAt)/1000);
        const next=run.room===2?levels.find(level=>level.id===(run.level?.id||0)+1)?.name:rooms[run.room+1];
        $('rift-transition').textContent=remaining>0?'Next: '+(next||'campaign complete')+' · '+remaining.toFixed(1)+'s':'Banking rewards…';
        if(remaining===0)await send('advance');
      }else await send('step');
    }
    if(playing)timer=setTimeout(loop,85);
  }
  async function pause(){
    if(!playing)return;
    playing=false;clearTimeout(timer);resetInput();silence();
    clearedAt=0;$('rift-transition').hidden=true;
    // Wait for the single pending input request, then persist the pause.
    while(busy)await new Promise(resolve=>setTimeout(resolve,20));
    if(!run||!['fighting','cleared'].includes(run.status))return;
    if(await send('pause')&&['fighting','cleared'].includes(run.status)){message('A moment by the lantern.','Take your time. The expedition will wait.','Resume expedition','PAUSED');$('rift-pause').textContent='Resume · Esc';$('rift-room-actions').hidden=true;}
  }
  async function begin(){
    if(busy||starting)return;
    if($('rift-start').dataset.retry){
      if($('rift-start').dataset.artworkRetry==='true'){location.reload();return;}
      delete $('rift-start').dataset.retry;$('rift-start').disabled=true;await load();return;
    }
    if(!ready)return;
    if($('rift-start').dataset.recover){delete $('rift-start').dataset.recover;await load();return;}
    starting=true;const intent=++startIntent;
    try{
      try{await audio.setActive(true,(run?.level?.region||0)*3+(run?.room||0));}catch(_){silence();}
      if(intent!==startIntent||document.hidden){silence();return;}
      const resume=run&&(run.status==='fighting'||run.status==='cleared');
      if(await send(resume?'resume':'start')){
        playing=true;
        if(intent!==startIntent||document.hidden){await pause();return;}
        $('rift-campaign').open=false;$('rift-overlay').hidden=true;$('rift-pause').disabled=false;$('rift-pause').textContent='Pause · Esc';update(run,true);$('rift-canvas').focus();status('WASD moves · Space jumps · J attacks · L guards · Q / E class abilities · 1–3 skills · R ultimate.');clearTimeout(timer);loop();
      }
    }finally{starting=false;}
  }
  function updateCampaign(){
    const active=run&&['fighting','cleared'].includes(run.status), completed=run?.completed_levels||[];
    const key=[selectedLevel,active,completed.join(',')].join('|');
    if(campaignKey===key)return;campaignKey=key;
    const level=run?.level?.id===selectedLevel?run.level:levels.find(l=>l.id===selectedLevel);
    if(!level)return;
    $('rift-region-title').textContent=level.region_name;
    $('rift-level-description').textContent='Mission '+level.id+' · '+level.name+' — '+level.tactic+'. Jump cover; glowing zones warn before they strike.';
    $('rift-progress').textContent=completed.length+'/100 completed · Mission '+selectedLevel;
    root.querySelectorAll('.rift-room-name').forEach((node,i)=>node.textContent=level.rooms[i].name.split(' / ')[1]);
    root.querySelectorAll('[data-level]').forEach(button=>{
      const id=Number(button.dataset.level);button.disabled=!!active;button.setAttribute('aria-pressed',String(id===selectedLevel));
      button.classList.toggle('completed',completed.includes(id));button.querySelector('small').textContent=completed.includes(id)?'Completed ✓':levels[id-1].difficulty;
    });
    window.RiftCampaignTools.update(run,selectedLevel);
  }
  function campaign(){
    $('rift-levels').replaceChildren();
    $('rift-region').querySelectorAll('option:not(:first-child)').forEach(n=>n.remove());
    levels.forEach(level=>{
      if((level.id-1)%10===0)text('option',level.region_name,$('rift-region')).value=String(level.region);
      const button=text('button','',$('rift-levels'));button.type='button';button.dataset.level=level.id;button.dataset.region=level.region;
      button.style.setProperty('--region-color',level.color);
      const art=text('span','',button,'rift-level-art');art.style.backgroundImage='url("'+root.dataset.regions+'")';art.style.backgroundPosition=(level.region%2*100)+'% '+(Math.floor(level.region/2)*25)+'%';
      text('span',String(level.id).padStart(3,'0'),button,'rift-level-number');text('strong',level.name.split(' · ')[1],button);text('span',level.region_name,button,'rift-level-region');text('small',level.difficulty,button);
      button.setAttribute('aria-label','Mission '+level.id+': '+level.name);
      button.addEventListener('click',()=>{selectedLevel=level.id;updateCampaign();renderer.preview(level);message(level.name.split(' · ')[1],level.tactic+'. Three tiers, one Abyss boss.','Enter mission '+level.id,level.region_name);});
    });
    $('rift-campaign').insertBefore($('rift-campaign-tools-extra'),$('rift-level-description'));
    window.RiftCampaignTools.init(levels);
    selectedLevel=run?.level&&['fighting','cleared'].includes(run.status)?run.level.id:window.RiftCampaignTools.preferred();campaignKey='';updateCampaign();renderer.preview(levels[selectedLevel-1]);
  }
  function loadout(){
    renderer.build(build);
    $('rift-build').textContent=build.name+' · Level '+build.level+' · '+build.class+'\n'+build.weapon;
    $('rift-sequence').textContent=build.sequence||'';
    $('rift-style').textContent=build.class_name||build.class;$('rift-resource').textContent=build.resource?build.resource+' · Q builds / E spends':'Class abilities unlock through Abyss progression';
    $('rift-gear').replaceChildren();build.gear.forEach(name=>text('li',name,$('rift-gear')));
    if(!build.gear.length)text('li','No equipment yet',$('rift-gear'));
    $('rift-loadout').replaceChildren();
    for(let i=0;i<Math.min(3,build.skills.length);i++){
      const label=text('label','Skill '+(i+1),$('rift-loadout')),select=document.createElement('select');select.setAttribute('aria-label','Expedition skill '+(i+1));
      text('option','None',select).value='';build.skills.forEach(s=>{text('option',s.name,select).value=s.id;});select.value=run?(run.build.skills[i]?.id||''):build.skills[i].id;label.append(select);
      select.addEventListener('change',()=>{const others=[...root.querySelectorAll('#rift-loadout select')].filter(el=>el!==select);if(select.value&&others.some(el=>el.value===select.value)){select.value='';status('Choose each skill only once.');}});
    }
  }
  async function load(){
    let artworkReady=false;
    try{
      await renderer.ready;
      artworkReady=true;
      const data=await request('GET');build=data.build;rooms=data.rooms;run=data.run;levels=data.levels||[];window.RiftLoot.init(data.rarities||[]);loadout();campaign();window.RiftBestiary.render(data.bestiary||[]);ready=true;
      if(run){update(run,true);if(['fighting','cleared'].includes(run.status))message('Your expedition awaits.','Resume from the last confirmed moment. Your expedition bag is still here.','Resume expedition','SAVED EXPEDITION');}
      else if(selectedLevel===1){$('rift-start').textContent='Enter the ruins →';$('rift-start').disabled=false;}
      else{const level=levels.find(l=>l.id===selectedLevel);message(level.name.split(' · ')[1],level.tactic+'. Three tiers, one Abyss boss.','Enter mission '+level.id,level.region_name);}
      status(root.dataset.fixture?'LOCAL PLAYTEST · Sample character and isolated rewards. No live inventory changes.':'Your Abyss character is ready. Choose up to three skills, then enter.');
      if(root.dataset.fixture)$('rift-overlay-note').textContent='Local playtest · Sample character · Isolated rewards';
    }catch(error){ready=false;$('rift-start').textContent=artworkReady?'Retry loading':'Reload artwork';$('rift-start').dataset.retry='true';$('rift-start').dataset.artworkRetry=String(!artworkReady);$('rift-start').disabled=false;status(error.message);}
  }
  function hold(button,value){
    button.dataset.action=value;
    button.addEventListener('pointerdown',event=>{if(!playing||button.disabled)return;event.preventDefault();button.setPointerCapture(event.pointerId);touch.add(value);taps.add(value);button.classList.add('rift-held');});
    button.addEventListener('click',event=>{if(event.detail===0&&playing&&!button.disabled)taps.add(value);});
    const release=event=>{if(event.type!=='pointerup'&&touch.has(value))taps.delete(value);touch.delete(value);button.classList.remove('rift-held');};['pointerup','pointercancel','lostpointercapture'].forEach(name=>button.addEventListener(name,release));
  }
  root.querySelectorAll('[data-hold]').forEach(button=>hold(button,button.dataset.hold));root.querySelectorAll('[data-move]').forEach(button=>hold(button,button.dataset.move));
  const controlKeys=new Set(['KeyW','KeyA','KeyS','KeyD','ArrowUp','ArrowDown','ArrowLeft','ArrowRight','KeyJ','KeyK','KeyL','KeyQ','KeyE','KeyR','Space','Digit1','Digit2','Digit3']);
  window.addEventListener('keydown',event=>{
    if(event.ctrlKey||event.metaKey||event.altKey||event.isComposing)return;
    if(event.code==='Escape'&&!event.repeat){if(playing)pause();else if(run&&['fighting','cleared'].includes(run.status))begin();return;}
    if(event.target.matches('input,select,textarea'))return;
    if(event.code==='Space'&&event.target.closest('button[data-action]'))return;
    if(!playing||!controlKeys.has(event.code))return;event.preventDefault();keys.add(event.code);if(!event.repeat)taps.add(event.code);
  });
  window.addEventListener('keyup',event=>keys.delete(event.code));window.addEventListener('blur',()=>{startIntent++;resetInput();if(playing)pause();});
  document.addEventListener('visibilitychange',()=>{if(document.hidden){startIntent++;resetInput();if(playing)pause();silence();}});
  window.addEventListener('pageshow',event=>{if(event.persisted){startIntent++;playing=false;clearTimeout(timer);resetInput();silence();load();}});
  $('rift-start').addEventListener('click',begin);$('rift-pause').addEventListener('click',()=>playing?pause():begin());
  $('rift-next').addEventListener('click',async()=>{if(await checkpoint($('rift-auto').checked?'advance':'next'))status(run.status==='complete'?'Expedition complete. Your rewards are banked.':'Checkpoint reached. Health restored by 25%; mana refilled.');});
  $('rift-auto').addEventListener('change',()=>{try{localStorage.setItem('rift-auto',String($('rift-auto').checked));}catch(_){}clearedAt=0;if(run)update(run,true);});
  $('rift-exit').addEventListener('click',()=>checkpoint('exit'));
  $('rift-fullscreen').addEventListener('click',async()=>{try{if(document.fullscreenElement)await document.exitFullscreen();else await $('rift-viewport').requestFullscreen();}catch(_){status('Fullscreen is unavailable in this browser.');}});
  function soundLabel(){$('rift-sound').textContent=audio.muted?'Sound off':'Sound on';$('rift-sound').setAttribute('aria-pressed',String(audio.muted));}
  soundLabel();$('rift-sound').addEventListener('click',async()=>{await audio.unlock();audio.set('muted',!audio.muted);soundLabel();audio.play('ui',0);});
  [['effects','rift-effects-volume'],['ambience','rift-ambience-volume']].forEach(([key,id])=>{$(id).value=audio[key]*100;$(id).addEventListener('input',()=>audio.set(key,Number($(id).value)/100));});
  try{const reduced=JSON.parse(localStorage.getItem('riftReducedMotion'));if(typeof reduced==='boolean')renderer.reduced=reduced;}catch(_){}
  $('rift-reduced').checked=renderer.reduced;$('rift-reduced').addEventListener('change',()=>{renderer.reduced=$('rift-reduced').checked;try{localStorage.setItem('riftReducedMotion',JSON.stringify(renderer.reduced));}catch(_){} });
  load();
})();
