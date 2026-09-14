(function () {
  'use strict';
  const $ = id => document.getElementById(id), root = $('rift-app'), audio = window.RiftAudio, renderer = window.RiftRenderer;
  const keys = new Set(), touch = new Set(), taps = new Set(), mouse = new Set();
  const keyOrder=new Map();let keySequence=0;
  let guardLatched=false,canvasMouse=false;
  const controls=window.RiftControls;
  let starting = false, startIntent = 0, checkpointPending = false;
  let run = null, build = null, rooms = [], playing = false, busy = false, ready = false, timer = 0, currentSkillIDs = '';
  let levels = [], selectedLevel = 1, campaignKey = '', clearedAt = 0;
  try { $('rift-auto').checked = localStorage.getItem('rift-auto') !== 'false'; } catch (_) {}
  const api = '/api/abyss/rift';
  const status = message => { $('rift-status').textContent = message; };
  function silence(){try{Promise.resolve(audio.setActive(false)).catch(()=>{});}catch(_){} }
  function text(tag, value, parent, className) { const node = document.createElement(tag); node.textContent = value; if(className)node.className=className; if(parent)parent.append(node); return node; }
  function put(node,value){if(node.textContent!==String(value))node.textContent=value;}
  function message(title, copy, button, kicker) {
    $('rift-result-actions').hidden=true;
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
    const pad=window.RiftGamepad.consume();
    const held = name => touch.has(name)||taps.has(name)||mouse.has(name);
    const pressed = action => pad.actions.has(action)||controls.codes(action).some(code=>keys.has(code)||taps.has(code));
    const direction=(positive,negative)=>{const last=action=>Math.max(0,...controls.codes(action).filter(code=>keys.has(code)||taps.has(code)).map(code=>keyOrder.get(code)||0));const p=last(positive),n=last(negative);return p||n?(p>n?1:-1):0;};
    const value = {x:direction('right','left')||Number(held('right'))-Number(held('left')),
      y:direction('down','up')||Number(held('down'))-Number(held('up')),
      attack:pressed('attack')||held('attack'),guard:controls.toggleGuard?guardLatched:pressed('guard')||held('guard'),jump:window.RiftJump.input(pressed('jump')||held('jump')),
      skill:''};
    const intent=window.RiftIntents.take(run,action=>pressed(action)||held(action),value.guard);value.skill=intent.skill;if(intent.wait)value.attack=false;
    value.x=value.x||pad.x;value.y=value.y||pad.y;taps.clear();return value;
  }
  function resetInput(){window.RiftHaptics.stop();window.RiftIntents.reset();window.RiftGamepad.reset();keys.clear();keyOrder.clear();touch.clear();taps.clear();mouse.clear();guardLatched=false;root.querySelectorAll('.rift-held').forEach(n=>n.classList.remove('rift-held'));guardDisplay();}
  function guardDisplay(){const button=root.querySelector('[data-bind="guard"]');button.setAttribute('aria-pressed',String(controls.toggleGuard&&guardLatched));button.title=controls.toggleGuard?'Toggle guard · '+(guardLatched?'On':'Off'):'Hold to guard';if(controls.toggleGuard)button.classList.toggle('rift-held',guardLatched);}
  function toggleGuard(){guardLatched=!guardLatched;guardDisplay();}
  function update(value, replay) {
    if(!value)return;
    if(value.status !== 'cleared' || replay) clearedAt = 0;
    else if(!clearedAt) clearedAt = performance.now();
    run=value;window.RiftIntents.sync(run,replay);renderer.snapshot(run,replay);window.RiftFeedback.update(run,replay,playing);window.RiftHaptics.update(run,replay,playing);
    const controlsEnabled=playing&&['fighting','cleared'].includes(run.status)&&!run.paused;
    $('rift-settings-return').disabled=!controlsEnabled;
    if(run.level){if(['fighting','cleared'].includes(run.status))selectedLevel=run.level.id;rooms=run.level.rooms.map(room=>room.name);}
    updateCampaign();
    $('rift-transition').hidden=run.status!=='cleared'||!playing||!$('rift-auto').checked;
    $('rift-vitals').hidden=false;put($('rift-name'),run.build.name);put($('rift-class'),run.build.class);
    put($('rift-style'),run.build.class_name||run.build.class);
    put($('rift-resource'),run.build.resource ? run.build.resource+' '+(run.resource||0)+'/3'+(run.barrier>0?' · Barrier '+Math.ceil(run.barrier):'') : 'Class abilities unlock through Abyss progression');
    put($('rift-hp'),Math.ceil(run.player.hp)+' / '+Math.ceil(run.player.max_hp));
    $('rift-hp-fill').style.width=Math.max(0,100*run.player.hp/run.player.max_hp)+'%';
    put($('rift-mana'),Math.floor(run.player.mana)+' MP');$('rift-mana-fill').style.width=run.player.mana+'%';
    put($('rift-room'),'Mission '+(run.level?.id||1)+' · Tier '+(run.room+1)+'/3 · '+(rooms[run.room]||'Mossbound Ruins'));
    const boss=run.enemies.find(e=>e.kind==='boss'&&e.hp>0);$('rift-boss').hidden=!boss;if(boss){$('rift-boss-fill').style.width=100*boss.hp/boss.max_hp+'%';put($('rift-boss-name'),boss.name);}
    const finalBoss=run.encounter_plan?.[2]?.find(e=>e.kind==='boss')||run.enemies.find(e=>e.kind==='boss');
    put($('rift-route-boss'),finalBoss?'Defeat '+finalBoss.name:'Defeat an Abyss boss');
    window.RiftLoot.update(run);
    root.querySelectorAll('.rift-route li').forEach((li,i)=>{li.classList.toggle('current',i===run.room);li.classList.toggle('done',i<run.room);});
    const signature=run.build.skills.map(s=>s.id).join(',');
    if(signature!==currentSkillIDs||!$('rift-skills').childElementCount){
      currentSkillIDs=signature;$('rift-skills').replaceChildren();run.build.skills.forEach((s,i)=>{const btn=document.createElement('button');btn.type='button';btn.dataset.hold=s.id;btn.title=s.name+' · '+s.cost+' MP · '+s.cooldown+'s cooldown';btn.setAttribute('aria-label',s.name);text('span','',btn,'rift-skill-icon');text('kbd',String(i+1),btn);text('span',s.name,btn);text('small','Ready',btn);$('rift-skills').append(btn);hold(btn,'skill:'+s.id);});
    }
    [...$('rift-skills').children].forEach((btn,i)=>{const s=run.build.skills[i],remaining=run.skill_timers[s.id]||0;put(btn.querySelector('small'),remaining>0?remaining.toFixed(1)+'s':s.cost+' MP');btn.disabled=!controlsEnabled||remaining>0||run.player.mana<s.cost;});
    const specials=[...(run.build.signatures||[]),...(run.build.ultimate?[run.build.ultimate]:[])];
    const specialIDs=specials.map(s=>s.id).join(',');
    if($('rift-signatures').dataset.ids!==specialIDs){$('rift-signatures').dataset.ids=specialIDs;$('rift-signatures').replaceChildren();specials.forEach((s,i)=>{const btn=document.createElement('button');btn.type='button';btn.title=s.name+' · '+s.cost+' MP';text('kbd',s===run.build.ultimate?'R':i?'E':'Q',btn);text('span',s.name,btn);text('small','Ready',btn);$('rift-signatures').append(btn);hold(btn,'skill:'+s.id);});}
    [...$('rift-signatures').children].forEach((btn,i)=>{const s=specials[i],remaining=run.skill_timers[s.id]||0;put(btn.querySelector('small'),remaining>0?remaining.toFixed(1)+'s':s.cost+' MP');btn.disabled=!controlsEnabled||remaining>0||run.player.mana<s.cost;});
    [...$('rift-skills').children].forEach((button,i)=>button.dataset.bind='skill'+i);
    [...$('rift-signatures').children].forEach((button,i)=>button.dataset.bind=specials[i]===run.build.ultimate?'ultimate':'signature'+i);
    controls.prompts();window.RiftHUD.update(run,playing);
    $('rift-room-actions').hidden=run.status!=='cleared'||!playing;
    put($('rift-clear-label'),run.room===2?(finalBoss?.name||'The boss')+' has fallen':'Area secured');
    put($('rift-next'),$('rift-auto').checked?'Continue now →':run.room===2?'Bank & finish expedition':'Bank & continue →');
    $('rift-pause').disabled=!playing&&run.status!=='fighting'&&run.status!=='cleared';
    put($('rift-pause'),(playing?'Pause · ':'Resume · ')+controls.label('pause'));
    root.querySelectorAll('#rift-loadout select').forEach(el=>el.disabled=run.status==='fighting'||run.status==='cleared');
    if(['defeated','complete','banked','expired'].includes(run.status)){
      playing=false;clearTimeout(timer);resetInput();$('rift-room-actions').hidden=true;
      const lost=run.status==='defeated';
      message(lost?'The rift takes its toll.':'Returned from the ruins.',lost?'Unbanked finds were lost. Your equipped gear and banked rewards are safe.':run.banked_gold.toLocaleString()+' gold and '+run.banked_items.length.toLocaleString()+' Abyss '+(run.banked_items.length===1?'item':'items')+' safely in your inventory.','Enter a new expedition',lost?'EXPEDITION ENDED':'REWARDS SECURED');
      if(run.status==='expired')message('A new chapter begins.','This expedition belongs to an earlier economy. Start a fresh run with your current character.','Enter a new expedition','EXPEDITION EXPIRED');
      $('rift-result-actions').hidden=!run.level||run.status==='expired';
      $('rift-replay').hidden=!run.level||!['complete','banked'].includes(run.status)||run.room!==2;
      $('rift-replay').textContent='Replay mission '+(run.level?.id||1);
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
    if(await send('pause')&&['fighting','cleared'].includes(run.status)){message('A moment by the lantern.','Take your time. The expedition will wait.','Resume expedition','PAUSED');$('rift-pause').textContent='Resume · '+controls.label('pause');$('rift-room-actions').hidden=true;}
  }
  async function begin(){
    if(busy||starting||controls.opened)return;
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
        $('rift-campaign').open=false;$('rift-overlay').hidden=true;$('rift-pause').disabled=false;update(run,true);$('rift-canvas').focus();status(controls.description());clearTimeout(timer);loop();
      }
    }finally{starting=false;}
  }
  function updateCampaign(){
    const active=run&&['fighting','cleared'].includes(run.status), completed=run?.completed_levels||[];
    const key=[selectedLevel,active,completed.join(','),JSON.stringify(run?.mission_history||{})].join('|');
    if(campaignKey===key)return;campaignKey=key;
    const level=run?.level?.id===selectedLevel?run.level:levels.find(l=>l.id===selectedLevel);
    if(!level)return;
    $('rift-region-title').textContent=level.region_name;
    $('rift-level-description').textContent='Mission '+level.id+' · '+level.name+' — '+level.tactic+'. Jump cover; glowing zones warn before they strike.';
    $('rift-progress').textContent=completed.length+'/100 completed · Mission '+selectedLevel;
    root.querySelectorAll('.rift-room-name').forEach((node,i)=>node.textContent=level.rooms[i].name.split(' / ')[1]);
    root.querySelectorAll('[data-level]').forEach(button=>{
      const id=Number(button.dataset.level);button.disabled=!!active;button.setAttribute('aria-pressed',String(id===selectedLevel));if(id===selectedLevel)button.setAttribute('aria-current','true');else button.removeAttribute('aria-current');
      button.classList.toggle('completed',completed.includes(id));button.querySelector('small').textContent=completed.includes(id)?'Completed ✓':levels[id-1].difficulty;
    });
    window.RiftCampaignTools.update(run,selectedLevel);
    window.RiftMission.update(level,run&&active?run.build:build,!!active);
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
    const preferred=window.RiftMission.preferred(window.RiftCampaignTools.preferred(),levels);
    selectedLevel=run?.level&&['fighting','cleared'].includes(run.status)?run.level.id:preferred;campaignKey='';updateCampaign();renderer.preview(levels[selectedLevel-1]);
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
    let artworkFailed=false;
    try{
      const [,data]=await Promise.all([renderer.ready.catch(error=>{artworkFailed=true;throw error;}),request('GET')]);
      build=data.build;rooms=data.rooms;run=data.run;levels=data.levels||[];window.RiftLoot.init(data.rarities||[]);loadout();campaign();window.RiftBestiary.render(data.bestiary||[]);ready=true;
      if(run){update(run,true);if(['fighting','cleared'].includes(run.status))message('Your expedition awaits.','Resume from the last confirmed moment. Your expedition bag is still here.','Resume expedition','SAVED EXPEDITION');}
      else if(selectedLevel===1){$('rift-start').textContent='Enter the ruins →';$('rift-start').disabled=false;}
      else{const level=levels.find(l=>l.id===selectedLevel);message(level.name.split(' · ')[1],level.tactic+'. Three tiers, one Abyss boss.','Enter mission '+level.id,level.region_name);}
      status(root.dataset.fixture?'LOCAL PLAYTEST · Sample character and isolated rewards. No live inventory changes.':'Your Abyss character is ready. Choose up to three skills, then enter.');
      if(root.dataset.fixture)$('rift-overlay-note').textContent='Local playtest · Sample character · Isolated rewards';
    }catch(error){ready=false;$('rift-start').textContent=artworkFailed?'Reload artwork':'Retry loading';$('rift-start').dataset.retry='true';$('rift-start').dataset.artworkRetry=String(artworkFailed);$('rift-start').disabled=false;status(error.message);}
  }
  function hold(button,value){
    button.dataset.action=value;
    button.addEventListener('pointerdown',event=>{if(!playing||button.disabled)return;event.preventDefault();button.setPointerCapture(event.pointerId);touch.add(value);taps.add(value);window.RiftIntents.press(value);button.classList.add('rift-held');});
    button.addEventListener('click',event=>{if(event.detail===0&&playing&&!button.disabled){if(value==='guard'&&controls.toggleGuard)toggleGuard();else{taps.add(value);window.RiftIntents.press(value);}}});
    const release=event=>{if(event.type==='pointerup'&&touch.has(value)&&value==='guard'&&controls.toggleGuard)toggleGuard();if(event.type!=='pointerup'&&touch.has(value)){taps.delete(value);window.RiftIntents.cancel(value);}touch.delete(value);button.classList.remove('rift-held');if(value==='guard')guardDisplay();};['pointerup','pointercancel','lostpointercapture'].forEach(name=>button.addEventListener(name,release));
  }
  root.querySelectorAll('[data-hold]').forEach(button=>hold(button,button.dataset.hold));root.querySelectorAll('[data-move]').forEach(button=>hold(button,button.dataset.move));
  $('rift-controls-open').addEventListener('click',async()=>{startIntent++;if(playing)await pause();resetInput();if(!controls.opened)controls.open();});
  window.addEventListener('riftbindingschange',()=>{resetInput();if(run)update(run,true);});
  window.addEventListener('riftintentchange',resetInput);
  $('rift-canvas').addEventListener('pointerdown',event=>{canvasMouse=event.pointerType==='mouse';if(playing&&canvasMouse&&controls.pointer(event.button))$('rift-canvas').setPointerCapture(event.pointerId);});
  $('rift-canvas').addEventListener('mousedown',event=>{
    const action=controls.pointer(event.button);if(!playing||controls.opened||!canvasMouse||!action||event.ctrlKey||event.metaKey||event.altKey)return;
    event.preventDefault();$('rift-canvas').focus();window.RiftIntents.recognize(action);
    if(action==='guard'&&controls.toggleGuard)toggleGuard();else{mouse.add(action);taps.add(action);}
  });
  window.addEventListener('mouseup',event=>{const action=controls.pointer(event.button);if(action)mouse.delete(action);});
  $('rift-canvas').addEventListener('lostpointercapture',()=>{for(const action of mouse)taps.delete(action);mouse.clear();});
  $('rift-canvas').addEventListener('contextmenu',event=>{if(playing&&controls.pointer(2))event.preventDefault();});
  window.addEventListener('keydown',event=>{
    if(controls.opened||event.ctrlKey||event.metaKey||event.altKey||event.isComposing)return;
    const action=controls.action(event.code);
    if(event.code==='Escape'&&!event.repeat){if(playing)pause();else if(run&&['fighting','cleared'].includes(run.status))begin();return;}
    if(event.target.matches('input,select,textarea'))return;
    if(['Space','Enter'].includes(event.code)&&event.target.closest('button,summary,a'))return;
    if(action==='pause'&&!event.repeat){event.preventDefault();if(playing)pause();else if(run&&['fighting','cleared'].includes(run.status))begin();return;}
    if(!playing||!action)return;event.preventDefault();keys.add(event.code);if(!event.repeat){keyOrder.set(event.code,++keySequence);window.RiftIntents.press(action);if(action==='guard'&&controls.toggleGuard)toggleGuard();else taps.add(event.code);}
  });
  window.addEventListener('keyup',event=>keys.delete(event.code));window.addEventListener('blur',()=>{startIntent++;resetInput();if(playing)pause();});
  document.addEventListener('visibilitychange',()=>{if(document.hidden){startIntent++;resetInput();if(playing)pause();silence();}});
  window.addEventListener('pageshow',event=>{if(event.persisted){startIntent++;playing=false;clearTimeout(timer);resetInput();silence();load();}});
  $('rift-start').addEventListener('click',begin);$('rift-pause').addEventListener('click',()=>playing?pause():begin());
  $('rift-replay').addEventListener('click',()=>{
    if(!ready||busy||starting||!run?.level||!['complete','banked'].includes(run.status)||run.room!==2)return;
    selectedLevel=run.level.id;campaignKey='';updateCampaign();begin();
  });
  $('rift-result-region').addEventListener('click',()=>{
    if(busy||starting||!run?.level||!['complete','banked','defeated'].includes(run.status))return;
    window.RiftCampaignTools.showRegion(run.level.region);
  });
  $('rift-next').addEventListener('click',async()=>{if(await checkpoint($('rift-auto').checked?'advance':'next'))status(run.status==='complete'?'Expedition complete. Your rewards are banked.':'Checkpoint reached. Health restored by 25%; mana refilled.');});
  $('rift-auto').addEventListener('change',()=>{try{localStorage.setItem('rift-auto',String($('rift-auto').checked));}catch(_){}clearedAt=0;if(run)update(run,true);});
  $('rift-exit').addEventListener('click',()=>checkpoint('exit'));
  $('rift-fullscreen').addEventListener('click',async()=>{try{if(document.fullscreenElement)await document.exitFullscreen();else await $('rift-viewport').requestFullscreen();}catch(_){status('Fullscreen is unavailable in this browser.');}});
  function soundLabel(){$('rift-sound').textContent=audio.muted?'Sound off':'Sound on';$('rift-sound').setAttribute('aria-pressed',String(audio.muted));}
  soundLabel();$('rift-sound').addEventListener('click',async()=>{await audio.unlock();audio.set('muted',!audio.muted);soundLabel();audio.play('ui',0);});
  [['effects','rift-effects-volume'],['ambience','rift-ambience-volume']].forEach(([key,id])=>{$(id).value=audio[key]*100;$(id).addEventListener('input',()=>audio.set(key,Number($(id).value)/100));});
  const systemMotion=window.matchMedia('(prefers-reduced-motion: reduce)');let reducedOverride=null;
  try{const reduced=JSON.parse(localStorage.getItem('riftReducedMotion'));if(typeof reduced==='boolean')reducedOverride=reduced;}catch(_){}
  function motionPreference(){renderer.reduced=reducedOverride??systemMotion.matches;$('rift-reduced').checked=renderer.reduced;window.dispatchEvent(new Event('riftmotionchange'));}
  $('rift-reduced').addEventListener('change',()=>{reducedOverride=$('rift-reduced').checked;try{localStorage.setItem('riftReducedMotion',JSON.stringify(reducedOverride));}catch(_){}motionPreference();});
  $('rift-system-motion').addEventListener('click',()=>{reducedOverride=null;try{localStorage.removeItem('riftReducedMotion');}catch(_){}motionPreference();});
  systemMotion.addEventListener('change',()=>{if(reducedOverride===null)motionPreference();});motionPreference();
  window.RiftGamepad.init({playing:()=>playing,skillCount:()=>run?.build.skills.length||0,recognize:action=>window.RiftIntents.recognize(action),ability:action=>window.RiftIntents.press(action),guard:()=>{if(controls.toggleGuard)toggleGuard();},togglePause:()=>playing?pause():begin(),disconnect:()=>{startIntent++;resetInput();if(playing)pause();},});
  const settings=root.querySelector('.rift-settings'),returnToBattlefield=document.createElement('button');returnToBattlefield.id='rift-settings-return';returnToBattlefield.type='button';returnToBattlefield.textContent='Return to battlefield';returnToBattlefield.disabled=true;settings.append(returnToBattlefield);
  function focusBattlefield(){if(!playing||controls.opened)return;resetInput();$('rift-canvas').focus();}
  returnToBattlefield.addEventListener('click',()=>{if(!playing||controls.opened)return;settings.open=false;focusBattlefield();});
  settings.addEventListener('toggle',()=>{if(!settings.open)focusBattlefield();});
  load();
})();
