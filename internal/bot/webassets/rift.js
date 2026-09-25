(function () {
  'use strict';
  const $ = id => document.getElementById(id), root = $('rift-app'), audio = window.RiftAudio, renderer = window.RiftRenderer;
  const keys = new Set(), touch = new Set(), taps = new Set(), mouse = new Set();
  const touchPointers=new Map();let touchJoystick={x:0,y:0};
  const keyOrder=new Map();let keySequence=0;
  let guardLatched=false,canvasMouse=false,practiceToolPending=false;
  const controls=window.RiftControls;
  const coarsePointer=window.matchMedia('(any-pointer: coarse)');
  const payloadDiagnostics=new URLSearchParams(location.search).get('riftPayloadDebug')==='1'?{count:0,totalResponseBytes:0,maxResponseBytes:0,samples:[]}:null;
  const payloadEncoder=payloadDiagnostics?new TextEncoder():null;
  if(payloadDiagnostics)window.RiftPayloadDiagnostics=payloadDiagnostics;
  const inputDiagnostics=new URLSearchParams(location.search).get('riftInputDebug')==='1'?{count:0,samples:[]}:null;
  const inputMarks=inputDiagnostics?new Map():null;
  let inputEpoch=0,inputOverlay=null;
  if(inputDiagnostics){
    window.RiftInputDiagnostics=inputDiagnostics;
    inputOverlay=document.createElement('div');inputOverlay.id='rift-input-diagnostics';
    inputOverlay.style.cssText='position:absolute;left:8px;bottom:8px;z-index:20;pointer-events:none;padding:6px;background:#071813eb;color:#d9f3ce;font:11px monospace;white-space:pre-line';
    inputOverlay.textContent='Input confirmation: waiting for a control press';$('rift-canvas').parentElement.append(inputOverlay);
  }
  function markInput(action){if(inputMarks&&playing)inputMarks.set(action,performance.now());}
  function takeInputTiming(value){
    if(!inputMarks)return null;
    const now=performance.now(),actions=[];let started=now;
    for(const [action,at] of inputMarks){
      const skill=action.startsWith('skill:')?action.slice(6):action==='ultimate'?run?.build.ultimate?.id:action.startsWith('signature')?run?.build.signatures?.[Number(action.slice(9))]?.id:action.startsWith('skill')?run?.build.skills?.[Number(action.slice(5))]?.id:null;
      const matches=action==='right'?value.x>0:action==='left'?value.x<0:action==='down'?value.y>0:action==='up'?value.y<0:action==='stop'?value.x===0&&value.y===0:skill?value.skill===skill:['attack','guard','jump'].includes(action)&&value[action];
      if(matches){actions.push(action);started=Math.min(started,at);inputMarks.delete(action);}
    }
    return actions.length?{actions,started,epoch:inputEpoch}:null;
  }
  function confirmInput(timing){
    if(!timing||timing.epoch!==inputEpoch)return;
    const total=performance.now()-timing.started,queue=timing.sent-timing.started,request=timing.received-timing.sent;
    const sample={actions:timing.actions,total,queue,request};
    inputDiagnostics.count++;inputDiagnostics.samples.push(sample);if(inputDiagnostics.samples.length>120)inputDiagnostics.samples.shift();
    const values=inputDiagnostics.samples.map(value=>value.total).sort((a,b)=>a-b);
    inputOverlay.textContent='Input confirmation '+total.toFixed(1)+' ms · p95 '+values[Math.ceil(values.length*.95)-1].toFixed(1)+' ms\nQueue '+queue.toFixed(1)+' ms · response '+request.toFixed(1)+' ms · '+values.length+'/120 samples';
  }
  let starting = false, startIntent = 0, checkpointPending = false;
  let run = null, build = null, rooms = [], playing = false, busy = false, ready = false, timer = 0, currentSkillIDs = '';
  let levels = [], selectedLevel = 1, campaignKey = '', clearedAt = 0, challenge = null, countdownAnnounced = -1;
  try{$('rift-confirm-boss').checked=localStorage.getItem('riftConfirmBoss')==='true';}catch(_){}
  try{$('rift-pause-boss-room').checked=localStorage.getItem('riftPauseBossRoom')==='true';}catch(_){}
  try{$('rift-pause-new-region').checked=localStorage.getItem('riftPauseNewRegion')==='true';}catch(_){}
  try{$('rift-fullscreen-controls').checked=localStorage.getItem('riftFullscreenControls')==='true';}catch(_){}
  function awaitingBossConfirmation(){return $('rift-auto').checked&&$('rift-confirm-boss').checked&&run?.room===2;}
  function awaitingBossRoomPause(){return $('rift-auto').checked&&$('rift-pause-boss-room').checked&&run?.room===1&&run?.status==='cleared';}
  function nextRegionEntering(){if(run?.room!==2)return null;const next=levels.find(l=>l.id===(run.level?.id||0)+1);return next&&next.region!==run.level?.region?next:null;}
  function awaitingNewRegionPause(){return $('rift-auto').checked&&$('rift-pause-new-region').checked&&run?.room===2&&run?.status==='cleared'&&!!nextRegionEntering();}
  $('rift-confirm-boss').addEventListener('change',()=>{try{localStorage.setItem('riftConfirmBoss',String($('rift-confirm-boss').checked));}catch(_){}clearedAt=0;countdownAnnounced=-1;if(run)update(run,true);});
  $('rift-pause-boss-room').addEventListener('change',()=>{try{localStorage.setItem('riftPauseBossRoom',String($('rift-pause-boss-room').checked));}catch(_){}clearedAt=0;countdownAnnounced=-1;if(run)update(run,true);});
  $('rift-pause-new-region').addEventListener('change',()=>{try{localStorage.setItem('riftPauseNewRegion',String($('rift-pause-new-region').checked));}catch(_){}clearedAt=0;countdownAnnounced=-1;if(run)update(run,true);});
  $('rift-fullscreen-controls').addEventListener('change',()=>{try{localStorage.setItem('riftFullscreenControls',String($('rift-fullscreen-controls').checked));}catch(_){}});
  let transitionDelay=1.2;
  try{const saved=Number(localStorage.getItem('riftTransitionDelay'));if([1.2,3,5,10].includes(saved))transitionDelay=saved;}catch(_){}
  $('rift-transition-delay').value=String(transitionDelay);
  $('rift-transition-delay').addEventListener('change',()=>{const value=Number($('rift-transition-delay').value);if(![1.2,3,5,10].includes(value))return;transitionDelay=value;clearedAt=0;countdownAnnounced=-1;try{localStorage.setItem('riftTransitionDelay',String(value));}catch(_){} });
  try { $('rift-auto').checked = localStorage.getItem('rift-auto') !== 'false'; } catch (_) {}
  const practice=root.dataset.practice||'', drillNames={touch:'Touch control calibration',banking:'Checkpoint banking',pickup:'Safe loot pickup',resource:'Resource management',ultimate:'Ultimate timing lane',ranged:'Ranged aiming lane',perfect_guard:'Perfect-guard timing',skills:'Skill testing lane',class:'Your class sequence',boss:'Boss phase practice',movement:'Movement lane',jump:'Jump over cover',combo:'Three-hit combo',guard:'Directional guard',hazard:'Read the warning zone'};
  const challengeParam=new URLSearchParams(location.search).get('challenge');
  const api = '/api/abyss/rift'+(practice?'?practice='+encodeURIComponent(practice):challengeParam?'?challenge='+encodeURIComponent(challengeParam):'');
  // Core status-line and request-error copy. Parameters remain plain text.
  const statusCopy=Object.freeze({
    uniqueSkill:"Choose each skill only once.",
    fixtureReady:"LOCAL PLAYTEST · Sample character and isolated rewards. No live inventory changes.",
    characterReady:"Your Abyss character is ready. Choose up to three skills, then enter.",
    screenshotEnabled:"Clean screenshot mode enabled. HUD hidden.",
    screenshotDisabled:"HUD restored.",
    rangeHidden:"Equipped skill range preview hidden.",
    expeditionComplete:"Expedition complete. Your rewards are banked.",
    checkpointReached:"Checkpoint reached. Health restored by 25%; mana refilled.",
    fullscreenUnavailable:"Fullscreen is unavailable in this browser.",
    fullscreenReview:"Review your controls before entering fullscreen.",
    sessionExpired:"Your session expired. Sign in again, then resume this expedition.",
    saveConflict:"The saved expedition changed. Recover it before continuing.",
    connectionInterrupted:"Connection interrupted. Recover the saved expedition before continuing.",
    responseInterrupted:"The expedition response was interrupted. Recover the saved expedition before continuing.",
    unconfirmed:"Could not confirm the expedition.",
    wrongDrill:"The saved drill does not match this page.",
    practiceReady:instructions=>'Practice is ready. '+instructions,
    showRange:name=>'Showing range for '+name
  });
  const status = message => { $('rift-status').textContent = message; };
  const transitionText=message=>message+(run?.build?.resource?' · '+run.build.resource+' '+(run.resource||0)+'/3':'');
  let lastAudioArea = -1;
  function silence(){lastAudioArea=-1;audio.stopBossMusic?.(0);audio.silence?.();try{Promise.resolve(audio.setActive(false)).catch(()=>{});}catch(_){} }
  function text(tag, value, parent, className) { const node = document.createElement(tag); node.textContent = value; if(className)node.className=className; if(parent)parent.append(node); return node; }
  function put(node,value){if(node&&node.textContent!==String(value))node.textContent=value;}
  function setSafeDisabled(node, disabled, explain=false) {
    if(!node)return;
    node.toggleAttribute('data-explain-disabled',disabled&&explain);
    const isFocused=document.activeElement===node;
    if(disabled){
      node.setAttribute('aria-disabled','true');
      if(explain){node.disabled=false;}else if(isFocused){
        node.disabled=false;
        const onBlur=()=>{
          if(node.getAttribute('aria-disabled')==='true'&&!node.hasAttribute('data-explain-disabled'))node.disabled=true;
          node.removeEventListener('blur',onBlur);
        };
        node.addEventListener('blur',onBlur);
      }else{
        node.disabled=true;
      }
    }else{
      node.removeAttribute('aria-disabled');
      node.disabled=false;
    }
  }
  function replacePreservingFocus(container, builder) {
    const active = document.activeElement;
    const isInside = container && active && container.contains(active);
    const activeBind = isInside ? active.dataset?.bind : null;
    const activeHold = isInside ? active.dataset?.hold : null;
    container.replaceChildren();
    builder();
    if(isInside){
      const target = (activeBind && container.querySelector(`[data-bind="${activeBind}"]`))
        || (activeHold && container.querySelector(`[data-hold="${activeHold}"]`))
        || container.querySelector('button');
      if(target) target.focus();
    }
  }
  function message(title, copy, button, kicker) {
    $('rift-result-actions').hidden=true;
    if($('rift-result-banner'))$('rift-result-banner').hidden=true;
    if($('rift-result-rewards'))$('rift-result-rewards').hidden=true;
    $('rift-overlay').hidden = false; $('rift-overlay-title').textContent = title; $('rift-overlay-copy').textContent = copy;
    $('rift-overlay-kicker').textContent = practice?(String(kicker||'').startsWith('PRACTICE')?kicker:'PRACTICE · '+(kicker||drillNames[practice])):kicker || 'MOSSBOUND RUINS'; $('rift-start').textContent = button; $('rift-start').disabled = !ready || busy;
  }
  function updatePauseButton(isPlaying){
    const btn=$('rift-pause');
    if(!btn)return;
    const action=isPlaying?'Pause':'Resume';
    const key=controls?.label('pause')||'Esc';
    let labelSpan=btn.querySelector('.rift-action-label');
    if(!labelSpan){
      labelSpan=document.createElement('span');
      labelSpan.className='rift-action-label';
      btn.replaceChildren(labelSpan);
    }
    labelSpan.textContent=action;
    let kbd=btn.querySelector('kbd');
    if(!kbd){
      kbd=document.createElement('kbd');
      btn.append(document.createTextNode(' '),kbd);
    }
    kbd.textContent=key;
    kbd.setAttribute('aria-hidden','true');
    btn.setAttribute('aria-label',action+' expedition');
    btn.setAttribute('aria-keyshortcuts',key);
  }
  let pendingRead=null,pendingRequest=null,loadGeneration=0;
  async function request(method, body, timing) {
    const previousRequest=pendingRequest;
    let settleRequest;
    const settled=new Promise(resolve=>{settleRequest=resolve;});pendingRequest=settled;
    await previousRequest;
    const started = performance.now(),requestBody=body?JSON.stringify(body):undefined;
    if(timing)timing.sent=started;
    const controller = new AbortController(), timeout = setTimeout(() => controller.abort(), 10000);
    if(method==='GET')pendingRead=controller;
    try {
      const response = await fetch(api,{method,credentials:'same-origin',cache:'no-store',headers:body?{'Content-Type':'application/json'}:undefined,body:requestBody,signal:controller.signal});
      if(response.status===401)throw new Error(statusCopy.sessionExpired);
      if(response.status===409)throw new Error(statusCopy.saveConflict);
      if(!response.ok)throw new Error(statusCopy.connectionInterrupted);
      let data,responseBytes=0,runBytes=0;
      try{
        if(payloadDiagnostics){const raw=await response.text();responseBytes=payloadEncoder.encode(raw).byteLength;data=JSON.parse(raw);runBytes=data.run?payloadEncoder.encode(JSON.stringify(data.run)).byteLength:0;}
        else data=await response.json();
      }catch(error){if(method==='GET'&&error?.name==='AbortError')throw error;throw new Error(statusCopy.responseInterrupted);}
      if(data?.ok===false)throw new Error(typeof data.error==='string'?data.error:statusCopy.unconfirmed);const result=window.RiftProtocol.validate(data,method,body);if(result.run&&(result.run.practice?.mode||'')!==practice)throw new Error(statusCopy.wrongDrill);
      if(payloadDiagnostics){
        payloadDiagnostics.count++;payloadDiagnostics.totalResponseBytes+=responseBytes;payloadDiagnostics.maxResponseBytes=Math.max(payloadDiagnostics.maxResponseBytes,responseBytes);
        payloadDiagnostics.samples.push({method,action:body?.kind||'load',requestBytes:requestBody?payloadEncoder.encode(requestBody).byteLength:0,responseBytes,runBytes});
        if(payloadDiagnostics.samples.length>32)payloadDiagnostics.samples.shift();
      }
      if(timing)timing.received=performance.now();
      const duration = performance.now() - started;
      if(window.RiftHUD?.updateLatency)window.RiftHUD.updateLatency(duration);
      return result;
    } finally { clearTimeout(timeout);if(pendingRead===controller)pendingRead=null;if(pendingRequest===settled)pendingRequest=null;settleRequest(); }
  }
  function input() {
    const pad=window.RiftGamepad.consume();
    const held = name => touch.has(name)||taps.has(name)||mouse.has(name);
    const pressed = action => pad.actions.has(action)||controls.codes(action).some(code=>keys.has(code)||taps.has(code));
    const direction=(positive,negative)=>{const last=action=>Math.max(0,...controls.codes(action).filter(code=>keys.has(code)||taps.has(code)).map(code=>keyOrder.get(code)||0));const p=last(positive),n=last(negative);return p||n?(p>n?1:-1):0;};
    const value = {x:direction('right','left')||Number(held('right'))-Number(held('left')),
      y:direction('down','up')||Number(held('down'))-Number(held('up')),
      attack:pressed('attack')||held('attack'),guard:controls.toggleGuard?guardLatched:pressed('guard')||held('guard'),jump:window.RiftJump.input(pressed('jump')||held('jump')),
      dodge:pressed('dodge')||held('dodge'),
      skill:''};
    const intent=window.RiftIntents.take(run,action=>pressed(action)||held(action),value.guard);value.skill=intent.skill;if(intent.wait)value.attack=false;
    value.x=value.x||touchJoystick.x||pad.x;value.y=value.y||touchJoystick.y||pad.y;taps.clear();return value;
  }
  function resetInput(){const captured=[...touchPointers];touchPointers.clear();for(const [id,{button}] of captured)if(button.hasPointerCapture(id))button.releasePointerCapture(id);touchJoystick={x:0,y:0};const stick=$('rift-joystick-stick');if(stick)stick.style.transform='translate(0px, 0px)';const base=$('rift-joystick-base');if(base)base.classList.remove('rift-active');if(inputMarks){inputMarks.clear();inputEpoch++;}window.RiftHaptics.stop();window.RiftIntents.reset();window.RiftGamepad.reset();keys.clear();keyOrder.clear();touch.clear();taps.clear();mouse.clear();guardLatched=false;root.querySelectorAll('.rift-held, [data-pressed="true"]').forEach(n=>{n.classList.remove('rift-held');delete n.dataset.pressed;if(n.dataset.bind!=='guard'||!controls.toggleGuard)n.setAttribute('aria-pressed','false');const m=n.dataset.move;if(m&&moveLabels[m])n.setAttribute('aria-label',moveLabels[m][0]);});guardDisplay();}
  function guardDisplay(){const button=root.querySelector('[data-bind="guard"]');if(!button)return;const isGuarding=Boolean((controls.toggleGuard&&guardLatched)||touch.has('guard'));button.setAttribute('aria-pressed',String(isGuarding));button.classList.toggle('rift-held',isGuarding);if(isGuarding)button.dataset.pressed='true';else delete button.dataset.pressed;}
  function toggleGuard(){guardLatched=!guardLatched;guardDisplay();}
  function practiceToolButtons(){const freeze=$('rift-practice-freeze');if(freeze)freeze.setAttribute('aria-pressed',String(!!run?.practice?.freeze_movement));root.querySelectorAll('[data-practice-action]').forEach(button=>setSafeDisabled(button,!practice||!ready||starting||practiceToolPending||run?.status!=='fighting'||button.dataset.practiceAction==='practice_bank'&&!run?.practice?.checkpoint_ready));}
  function hazardPracticePhase(run){const hazard=run.practice.arena.hazards[0],phase=(run.clock+hazard.offset)%hazard.period;return phase<1.2?'Warning: move or prepare to jump':phase<1.2+hazard.duration?'Active hazard':'Wait for the next warning';}
  const abilityAbbreviations = {
    'Frost Detonation': 'Frost Det.',
    'Resolute Bash': 'Res. Bash',
    'Rending Strike': 'Rend. Strike',
    'Rage Execution': 'Rage Exec.',
    'Sighting Shot': 'Sight. Shot',
    'Piercing Volley': 'Pierce. Volley',
    'Pack Assault': 'Pack Assault',
    'Pack Mark': 'Pack Mark',
    'Ember Seed': 'Ember Seed',
    'Time Bolt': 'Time Bolt',
    'Temporal Release': 'Temp. Release',
    'Mending Light': 'Mend. Light',
    'Grace Flare': 'Grace Flare',
    'Stone Aegis': 'Stone Aegis',
    'Seismic Break': 'Seismic Break',
    'Leech Cut': 'Leech Cut',
    'Crimson Reap': 'Crim. Reap',
    'Void Hex': 'Void Hex',
    'Oblivion Burst': 'Obliv. Burst',
    'Rune Inscription': 'Rune Inscr.',
    'Runic Discharge': 'Runic Disch.',
    'Volatile Mixture': 'Vol. Mixture',
    'Catalytic Burst': 'Cat. Burst',
    'Iron Guard': 'Iron Guard',
  };
  let abbreviateAbilities = false;
  try { abbreviateAbilities = localStorage.getItem('riftAbbreviateAbilities') === 'true'; } catch(_) {}
  function formatAbilityName(name){
    if(!abbreviateAbilities) return name;
    if(abilityAbbreviations[name]) return abilityAbbreviations[name];
    if(name.length > 12) {
      const parts = name.split(' ');
      if(parts.length > 1) return parts[0].slice(0, 5) + '. ' + parts.slice(1).join(' ');
      return name.slice(0, 10) + '.';
    }
    return name;
  }
  function updateAbilityLabels(){
    [...$('rift-skills').children, ...$('rift-signatures').children].forEach(btn => {
      const full = btn.dataset.abilityName || (btn.getAttribute('aria-label')||'').split(' · ')[0];
      const label = btn.querySelector('.rift-action-label');
      if(label && full) label.textContent = formatAbilityName(full);
    });
  }
  function update(value, replay) {

    if(!value)return;
    if(run&&(run.id!==value.id||run.room!==value.room||run.level?.id!==value.level?.id))resetInput();
    if(value.status !== 'cleared' || replay) { clearedAt = 0; countdownAnnounced = -1; }
    else if(!clearedAt) { clearedAt = performance.now(); countdownAnnounced = -1; }
    run=value;classPrimer();window.RiftBossIntro.update(run);window.RiftBestiary.update(run);window.RiftIntents.sync(run,replay);renderer.snapshot(run,replay);window.RiftFeedback.update(run,replay,playing);window.RiftHaptics.update(run,replay,playing);
    const controlsEnabled=playing&&['fighting','cleared'].includes(run.status)&&!run.paused;
    const gamePaused=!playing&&['fighting','cleared'].includes(run.status)||run.paused;
    window.RiftObjectives.update(run,gamePaused);
    const roomGoal=run.room_objective;
    $('rift-room-objective').hidden=!roomGoal;
    $('rift-room-objective').dataset.contested=String(!!roomGoal?.contested);
    if(roomGoal?.kind==='rune_gate'){
      const ended=!['fighting','cleared'].includes(run.status),combat=run.enemies.some(e=>e.hp>0),next=roomGoal.pickups.find(p=>p.id===roomGoal.sequence[roomGoal.collected]);
      put($('rift-room-objective-progress'),'Rune gate '+roomGoal.collected+'/3 · '+(ended?'Expedition ended':roomGoal.complete?'Open':gamePaused?'Paused':combat?'Defeat the patrol':'Next seal: '+next.id));
      put($('rift-room-objective-help'),ended?'This tier was not secured.':roomGoal.complete?'Gate open. Bank collected loot to continue.':'Order: '+roomGoal.sequence.join(' → ')+'. '+(combat?'Seals activate after the patrol is defeated.':'Step on each seal in order. A wrong seal resets progress without damage. Floor hazards are off.'));
      put($('rift-room-objective-directions'),ended||roomGoal.complete||combat?'':'Seal '+next.id+': '+(next.x<run.player.x?'left':'right')+(Math.abs(next.y-run.player.y)<24?'':next.y<run.player.y?', up':', down'));
    }else if(roomGoal?.kind==='split_defense'){
      const failed=roomGoal.lanes.some(l=>l.ward.hp<=0),ended=!['fighting','cleared'].includes(run.status);
      put($('rift-room-objective-progress'),roomGoal.lanes.map(l=>l.ward.name+': '+Math.ceil(l.ward.hp)+'%').join(' · ')+(failed?' · Breached':roomGoal.complete?' · Protected':gamePaused?' · Paused':''));
      put($('rift-room-objective-help'),failed?'A lane ward fell. The expedition ended.':ended?'This tier was not secured.':roomGoal.complete?'Both wards survived. Bank the loot to continue.':'Intercept enemies in both lanes. Each attacker at a ward drains five light per second, up to two attackers. Both wards must survive.');
      put($('rift-room-objective-directions'),ended||roomGoal.complete?'':roomGoal.lanes.map(l=>l.ward.name+': '+(l.contested?'UNDER ATTACK': 'holding')+' · '+(l.ward.x<run.player.x?'left':'right')+(l.ward.y<run.player.y?', up':', down')).join(' | '));
    }else if(roomGoal?.kind==='protect_lantern'){
      const ended=!['fighting','cleared'].includes(run.status),lamp=roomGoal.lantern;
      put($('rift-room-objective-progress'),'Lantern '+Math.ceil(lamp.hp)+'% · '+(lamp.hp<=0?'Extinguished':ended?'Expedition ended':roomGoal.complete?'Protected':gamePaused?'Paused':roomGoal.contested?'Under threat':'Keep enemies away'));
      put($('rift-room-objective-help'),lamp.hp<=0?'The lantern went out. The expedition ended.':ended?'This tier was not secured.':roomGoal.complete?'Lantern protected and patrol defeated. Bank the loot to continue.':'Enemies inside the ring drain five light per second each, up to three enemies. Draw them away or defeat them. Losing all light ends the expedition.');
      put($('rift-room-objective-directions'),ended||roomGoal.complete?'':'Lantern: '+(lamp.x<run.player.x?'left':'right')+(Math.abs(lamp.y-run.player.y)<24?'':lamp.y<run.player.y?', up':', down'));
    }else if(roomGoal?.kind==='rescue_companions'){
      const ended=!['fighting','cleared'].includes(run.status),captives=roomGoal.captives.filter(c=>!c.freed);
      put($('rift-room-objective-progress'),'Companions '+roomGoal.collected+'/2 · '+(ended?'Expedition ended':roomGoal.complete?'Rescued':gamePaused?'Paused':'Break the cages'));
      put($('rift-room-objective-help'),ended?'This tier was not secured.':roomGoal.complete?'Companions rescued. Clear the remaining patrol and bank the loot.':'Attacks and spells break the cages without harming the captive spirits. Cages grant no monster kills or loot.');
      put($('rift-room-objective-directions'),ended||roomGoal.complete?'':captives.map(c=>c.name+': '+(c.x<run.player.x?'left':'right')+(Math.abs(c.y-run.player.y)<24?'':c.y<run.player.y?', up':', down')).join(' · '));
    }else if(roomGoal?.kind==='linked_guardians'){
      const ended=!['fighting','cleared'].includes(run.status),targets=run.enemies.filter(e=>roomGoal.targets.includes(e.id)&&e.hp>0);
      put($('rift-room-objective-progress'),'Guardians '+roomGoal.collected+'/2 · '+(ended?'Expedition ended':roomGoal.complete?'Defeated':gamePaused?'Paused':roomGoal.bond_active?'Linked · 50% damage reduction':'Bond broken'));
      put($('rift-room-objective-help'),ended?'This tier was not secured.':roomGoal.complete?'Guardians defeated. Clear the remaining patrol and bank the loot.':'Separate the guardians by more than 240 units or defeat one to remove their protection. They can reform the bond when close together.');
      put($('rift-room-objective-directions'),ended||roomGoal.complete?'':targets.map(e=>e.name+': '+(e.x<run.player.x?'left':'right')+(Math.abs(e.y-run.player.y)<24?'':e.y<run.player.y?', up':', down')).join(' · '));
    }else if(roomGoal?.kind==='escape_collapse'){
      const ended=!['fighting','cleared'].includes(run.status),caught=run.player.x<=roomGoal.collapse_x;
      put($('rift-room-objective-progress'),'Collapse · '+(ended?'Expedition ended':roomGoal.complete?'Escaped':gamePaused?'Paused':roomGoal.seconds<3?'Starts in '+(3-roomGoal.seconds).toFixed(1)+'s':caught?'Caught in the collapse':'Keep moving to the exit'));
      put($('rift-room-objective-help'),ended?'This tier was not secured.':roomGoal.complete?'You escaped. Bank collected loot to continue.':'Stay ahead of the advancing edge and land inside the exit seal. Fight for loot if time permits; surviving enemies grant no rewards.');
      put($('rift-room-objective-directions'),ended||roomGoal.complete?'':'Exit: '+(roomGoal.zone.x<run.player.x?'left':'right')+(Math.abs(roomGoal.zone.y-run.player.y)<24?'':roomGoal.zone.y<run.player.y?', up':', down'));
    }else if(roomGoal?.kind==='interrupt_ritual'){
      const ended=!['fighting','cleared'].includes(run.status),channels=roomGoal.channels.map(c=>({...c,enemy:run.enemies.find(e=>e.id===c.enemy_id)})).filter(c=>c.enemy.hp>0),next=channels.length?Math.min(...channels.map(c=>8-c.seconds)):0;
      put($('rift-room-objective-progress'),'Ritual '+roomGoal.collected+'/'+roomGoal.target+' · '+(ended?'Expedition ended':roomGoal.complete?'Interrupted':gamePaused?'Paused':'Next pulse in '+next.toFixed(1)+'s'));
      put($('rift-room-objective-help'),ended?'This tier was not secured.':roomGoal.complete?'Ritual ended. Defeat the remaining patrol and bank the tier loot.':'Hit a channeler to reset its eight-second charge. Leave the pulse ring before discharge. Defeat every channeler.');
      put($('rift-room-objective-directions'),ended||roomGoal.complete?'':channels.map(c=>c.enemy.name+': '+(c.enemy.x<run.player.x?'left':'right')+(Math.abs(c.enemy.y-run.player.y)<24?'':c.enemy.y<run.player.y?', up':', down')).join(' · '));
    }else if(roomGoal?.kind==='escort_spirit'){
      const ended=!['fighting','cleared'].includes(run.status),spirit=roomGoal.escort,percent=Math.min(100,Math.floor((spirit.x-350)/1100*100));
      put($('rift-room-objective-progress'),'Spirit '+percent+'% · '+(ended?'Expedition ended':roomGoal.complete?'Safe at the exit':gamePaused?'Paused':roomGoal.contested?'Clear nearby enemies':roomGoal.escort_moving?'Escorting':'Stay near the spirit'));
      put($('rift-room-objective-help'),ended?'This tier was not secured.':roomGoal.complete?(run.enemies.some(e=>e.hp>0)?'The spirit is safe. Defeat the remaining patrol.':'Spirit escorted and patrol cleared. Bank the tier loot to continue.'):'Stay inside the spirit’s support ring. It waits when you move away or enemies get close. Saved progress is kept.');
      put($('rift-room-objective-directions'),ended||roomGoal.complete?'':'Spirit: '+(spirit.x<run.player.x?'left':'right')+(Math.abs(spirit.y-run.player.y)<24?'':spirit.y<run.player.y?', up':', down'));
    }else if(roomGoal?.kind==='moving_beacons'){
      const ended=!['fighting','cleared'].includes(run.status),zone=roomGoal.zone;
      put($('rift-room-objective-progress'),'Beacons '+roomGoal.collected+'/3 · '+(ended?'Expedition ended':roomGoal.complete?'Captured':gamePaused?'Paused':'Beacon '+(roomGoal.collected+1)+': '+Math.floor(roomGoal.seconds)+'/3 s · '+(roomGoal.charging?'Capturing':'Follow the ring')));
      put($('rift-room-objective-help'),ended?'This tier was not secured.':roomGoal.complete?(run.enemies.some(e=>e.hp>0)?'All beacons captured. Defeat the patrol to secure the tier.':'Beacons and patrol cleared. Bank the tier loot to continue.'):'Stay grounded inside the moving ring for three seconds. Leaving keeps your capture progress.');
      put($('rift-room-objective-directions'),ended||roomGoal.complete?'':'Beacon: '+(zone.x<run.player.x?'left':'right')+(Math.abs(zone.y-run.player.y)<24?'':zone.y<run.player.y?', up':', down'));
    }else if(roomGoal?.kind==='marked_hunt'){
      const ended=!['fighting','cleared'].includes(run.status),targets=run.enemies.filter(e=>roomGoal.targets.includes(e.id)&&e.hp>0);
      put($('rift-room-objective-progress'),'Targets '+roomGoal.collected+'/'+roomGoal.target+' · '+(ended?'Expedition ended':roomGoal.complete?'Hunt complete':gamePaused?'Paused':'Defeat the marked enemies'));
      put($('rift-room-objective-help'),ended?'This tier was not secured.':roomGoal.complete?'Targets defeated. Survivors retreated without granting kills or loot. Bank the tier loot to continue.':'Look for gold diamond markers. Defeat those enemies to secure the tier; surviving unmarked enemies retreat without rewards.');
      put($('rift-room-objective-directions'),ended?'':targets.map(e=>e.name+': '+(e.x<run.player.x?'left':'right')+(Math.abs(e.y-run.player.y)<24?'':e.y<run.player.y?', up':', down')).join(' · '));
    }else if(roomGoal?.kind==='disable_generators'){
      const ended=!['fighting','cleared'].includes(run.status),remaining=run.enemies.filter(e=>e.kind==='generator'&&e.hp>0);
      put($('rift-room-objective-progress'),'Generators '+roomGoal.collected+'/'+roomGoal.target+' · '+(ended?'Expedition ended':gamePaused?'Paused':roomGoal.complete?'All hazards off':'Shut down the hazards'));
      put($('rift-room-objective-help'),ended?'This tier was not secured.':roomGoal.complete?(run.enemies.some(e=>e.kind!=='generator'&&e.hp>0)?'Hazards disabled. Defeat the remaining patrol.':'Generators and patrol cleared. Bank the tier loot to continue.'):'Destroy each generator with attacks or spells. Its linked floor hazard stays off. Generators grant no monster loot or kill credit.');
      put($('rift-room-objective-directions'),ended?'':remaining.map(e=>e.name+': '+(e.x<run.player.x?'left':'right')+(Math.abs(e.y-run.player.y)<24?'':e.y<run.player.y?', up':', down')).join(' · '));
    }else if(roomGoal?.kind==='carry_relic'){
      const ended=!['fighting','cleared'].includes(run.status),target=roomGoal.carrying?roomGoal.zone:roomGoal.relic;
      put($('rift-room-objective-progress'),'Relic · '+(ended?'Expedition ended':roomGoal.complete?'Delivered':roomGoal.carrying?'Carrying · 30% slower':'Find the relic')+(gamePaused&&!ended&&!roomGoal.complete?' · Paused':''));
      put($('rift-room-objective-help'),ended?'This tier was not secured.':roomGoal.complete?(run.enemies.some(e=>e.hp>0)?'Movement restored. Defeat the patrol to secure this tier.':'Relic delivered and patrol cleared. Bank the tier loot to continue.'):roomGoal.carrying?'Carry it to the exit seal. Attacks and jumps remain available. Land inside the seal to deliver.':'Walk over the relic to pick it up. Carrying reduces movement speed by 30%.');
      put($('rift-room-objective-directions'),ended||roomGoal.complete?'':(roomGoal.carrying?'Exit seal: ':'Relic: ')+(target.x<run.player.x?'left':'right')+(Math.abs(target.y-run.player.y)<24?'':target.y<run.player.y?', up':', down'));
    }else if(roomGoal?.kind==='destroy_totems'){
      const ended=!['fighting','cleared'].includes(run.status),remaining=run.enemies.filter(e=>e.kind==='totem'&&e.hp>0),patrol=run.enemies.filter(e=>e.kind!=='totem'&&e.hp>0).length;
      put($('rift-room-objective-progress'),'Totems '+roomGoal.collected+'/3 · '+(ended?'Expedition ended':gamePaused?'Paused':roomGoal.complete?'Shattered':'Destroy the ritual props'));
      put($('rift-room-objective-help'),ended?'This tier was not secured.':roomGoal.complete?(patrol?'Defeat the remaining patrol to secure this tier.':'All totems and defenders cleared. Bank the tier loot to continue.'):'Use attacks or damaging spells. Totems do not grant monster loot or kill credit.');
      put($('rift-room-objective-directions'),ended?'':remaining.map(e=>e.name+': '+(e.x<run.player.x?'left':'right')+(Math.abs(e.y-run.player.y)<24?'':e.y<run.player.y?', up':', down')).join(' · '));
    }else if(roomGoal?.kind==='survive_waves'){
      const ended=!['fighting','cleared'].includes(run.status), waiting=roomGoal.next_wave_seconds>0;
      put($('rift-room-objective-progress'),'Wave '+roomGoal.wave+'/3 · '+(ended?'Expedition ended':roomGoal.complete?'Survived':gamePaused?'Paused':waiting?'Reinforcements in '+Math.ceil(roomGoal.next_wave_seconds)+'s':'Defeat the attackers'));
      put($('rift-room-objective-help'),ended?'This tier was not secured.':roomGoal.complete?'All three waves defeated. Bank the tier loot to continue.':waiting?'A new group is approaching. Reposition before they arrive.':roomGoal.wave===3?'Defeat the final group to secure this tier and bank its loot.':'Defeat this group to trigger the next wave. Loot stays available throughout the fight.');
      put($('rift-room-objective-directions'),roomGoal.complete||ended?'':run.enemies.filter(enemy=>enemy.hp>0).length+' enemies remaining in this wave');
    }else if(roomGoal?.kind==='hold_circle'){
      const zone=roomGoal.zone,inside=((run.player.x-zone.x)/zone.radius_x)**2+((run.player.y-zone.y)/zone.radius_y)**2<=1;
      const ended=!['fighting','cleared'].includes(run.status);
      const state=ended?'Expedition ended':roomGoal.complete?'Charged':gamePaused?'Paused':roomGoal.contested?'Contested':inside&&run.player.jump<=.1?'Charging':'Move onto the circle';
      put($('rift-room-objective-progress'),roomGoal.name+' · '+Math.floor(roomGoal.seconds)+' / '+roomGoal.target+' s · '+state);
      put($('rift-room-objective-help'),ended?'This tier was not secured.':roomGoal.complete?(run.enemies.some(e=>e.hp>0)?'Circle charged. Defeat the remaining enemies.':'Circle charged and enemies defeated. Tier secured.'):'Stay grounded inside the circle while no enemy occupies it. Leaving keeps the charge you have earned.');
      put($('rift-room-objective-directions'),inside?'Inside the circle':('Circle: '+(run.player.x<zone.x?'right':'left')+(Math.abs(run.player.y-zone.y)<20?'':run.player.y<zone.y?', down':', up')));
    }else if(roomGoal){
      put($('rift-room-objective-progress'),roomGoal.name+' · '+roomGoal.collected+' / '+roomGoal.target+(roomGoal.complete?' · Gathered':''));
      put($('rift-room-objective-help'),roomGoal.complete?(run.enemies.some(e=>e.hp>0)?'Sigils gathered. Defeat the remaining enemies.':'Sigils gathered and enemies defeated. Tier secured.'):'Walk over the marked sigils on the ground. Both the sigils and enemy defeats are required to clear this tier.');
      put($('rift-room-objective-directions'),roomGoal.pickups.filter(p=>!p.collected).map(p=>'Sigil '+p.id+': '+(Math.abs(p.x-run.player.x)<28?'aligned':p.x<run.player.x?'left':'right')+(Math.abs(p.y-run.player.y)<24?'':p.y<run.player.y?', up':', down')).join(' · '));
    }
    if(gamePaused)root.dataset.paused='';else delete root.dataset.paused;
    setSafeDisabled($('rift-settings-return'),!controlsEnabled);
    if(run.level){
      if(['fighting','cleared'].includes(run.status))selectedLevel=run.level.id;
      rooms=run.level.rooms.map(room=>room.name);
      if(playing&&['fighting','cleared'].includes(run.status)&&!run.paused){
        const areaIdx=(run.level?.region||0)*3+(run.room||0);
        if(areaIdx!==lastAudioArea){lastAudioArea=areaIdx;audio.area(areaIdx);}
      }
    }
    updateCampaign();
    $('rift-transition').hidden=run.status!=='cleared'||!playing||!$('rift-auto').checked;
    $('rift-vitals').hidden=false;put($('rift-name'),run.build.name);put($('rift-class'),run.build.class);
    put($('rift-style'),window.RiftRecords.classIdentity(run.build));
    put($('rift-resource'),run.build.resource ? run.build.resource+' '+(run.resource||0)+'/3'+(run.barrier>0?' · Barrier '+Math.ceil(run.barrier):'') : 'Class abilities unlock through Abyss progression');
    put($('rift-hp'),Math.ceil(run.player.hp)+' / '+Math.ceil(run.player.max_hp));
    $('rift-hp-fill').style.width=Math.max(0,100*run.player.hp/run.player.max_hp)+'%';
    put($('rift-mana'),Math.floor(run.player.mana)+' MP');$('rift-mana-fill').style.width=run.player.mana+'%';
    put($('rift-room'),'Mission '+(run.level?.id||1)+' · Tier '+(run.room+1)+'/3 · '+(rooms[run.room]||'Mossbound Ruins'));
    const boss=run.enemies.find(e=>e.kind==='boss'&&e.hp>0);$('rift-boss').hidden=!boss;if(boss){$('rift-boss-fill').style.width=100*boss.hp/boss.max_hp+'%';put($('rift-boss-name'),boss.name);$('rift-boss-name').title=boss.name;if(playing&&['fighting'].includes(run.status)&&!run.paused&&!audio.bossMusicActive)audio.startBossMusic?.();}else if(audio.bossMusicActive){audio.fadeBossMusic?.(1.8);}
    const finalBoss=run.encounter_plan?.[2]?.find(e=>e.kind==='boss')||run.enemies.find(e=>e.kind==='boss');
    put($('rift-route-boss'),finalBoss?'Defeat '+finalBoss.name:'Defeat an Abyss boss');
    window.RiftLoot.update(run,replay);
    root.querySelectorAll('.rift-route li').forEach((li,i)=>{li.classList.toggle('current',i===run.room);li.classList.toggle('done',i<run.room);});
    const signature=run.build.skills.map(s=>s.id).join(',');
    if(signature!==currentSkillIDs||$('rift-skills').childElementCount!==run.build.skills.length){
      currentSkillIDs=signature;
      replacePreservingFocus($('rift-skills'),()=>{
        run.build.skills.forEach((s,i)=>{
          const btn=document.createElement('button');btn.type='button';btn.dataset.hold=s.id;btn.dataset.abilityName=s.name;btn.dataset.bind='skill'+i;btn.setAttribute('aria-label',s.name);btn.setAttribute('aria-keyshortcuts',String(i+1));text('span','',btn,'rift-skill-icon');const kbd=text('kbd',String(i+1),btn);kbd.setAttribute('aria-hidden','true');text('span',formatAbilityName(s.name),btn,'rift-action-label');text('small','Ready',btn);btn.addEventListener('pointerenter',()=>window.RiftHUD.setRequestedRange(s));btn.addEventListener('pointerleave',()=>{window.RiftHUD.setRequestedRange(pinnedRangeSkill); });btn.addEventListener('focus',()=>window.RiftHUD.setRequestedRange(s));btn.addEventListener('blur',()=>{window.RiftHUD.setRequestedRange(pinnedRangeSkill); });$('rift-skills').append(btn);hold(btn,'skill:'+s.id);
        });
      });
    }
    [...$('rift-skills').children].forEach((btn,i)=>{
      const s=run.build.skills[i],remaining=run.skill_timers[s.id]||0;
      put(btn.querySelector('small'),remaining>0?remaining.toFixed(1)+'s':s.cost+' MP');
      setSafeDisabled(btn,!controlsEnabled||remaining>0||run.player.mana<s.cost,controlsEnabled&&coarsePointer.matches);
    });
    const specials=[...(run.build.signatures||[]),...(run.build.ultimate?[run.build.ultimate]:[])];
    const specialIDs=specials.map(s=>s.id).join(',');
    if($('rift-signatures').dataset.ids!==specialIDs){
      $('rift-signatures').dataset.ids=specialIDs;
      replacePreservingFocus($('rift-signatures'),()=>{
        specials.forEach((s,i)=>{
          const btn=document.createElement('button');btn.type='button';btn.dataset.bind=s===run.build.ultimate?'ultimate':'signature'+i;btn.dataset.abilityName=s.name;const defaultKey=s===run.build.ultimate?'R':i?'E':'Q';btn.setAttribute('aria-label',s.name);btn.setAttribute('aria-keyshortcuts',defaultKey);const kbd=text('kbd',defaultKey,btn);kbd.setAttribute('aria-hidden','true');text('span',formatAbilityName(s.name),btn,'rift-action-label');text('small','Ready',btn);btn.addEventListener('pointerenter',()=>window.RiftHUD.setRequestedRange(s));btn.addEventListener('pointerleave',()=>{window.RiftHUD.setRequestedRange(pinnedRangeSkill); });btn.addEventListener('focus',()=>window.RiftHUD.setRequestedRange(s));btn.addEventListener('blur',()=>{window.RiftHUD.setRequestedRange(pinnedRangeSkill); });$('rift-signatures').append(btn);hold(btn,'skill:'+s.id);
        });
      });
    }
    [...$('rift-signatures').children].forEach((btn,i)=>{
      const s=specials[i],remaining=run.skill_timers[s.id]||0;
      put(btn.querySelector('small'),remaining>0?remaining.toFixed(1)+'s':s.cost+' MP');
      setSafeDisabled(btn,!controlsEnabled||remaining>0||run.player.mana<s.cost,controlsEnabled&&coarsePointer.matches);
    });
    [...$('rift-skills').children].forEach((button,i)=>button.dataset.bind='skill'+i);
    [...$('rift-signatures').children].forEach((button,i)=>button.dataset.bind=specials[i]===run.build.ultimate?'ultimate':'signature'+i);
    controls.prompts();window.RiftHUD.update(run,playing,replay);
    $('rift-room-actions').hidden=run.status!=='cleared'||!playing;
    put($('rift-clear-label'),run.room===2?(finalBoss?.name||'The boss')+' has fallen':'Area secured');
    if(awaitingNewRegionPause()){const nextReg=nextRegionEntering();put($('rift-transition'),transitionText('Approaching '+(nextReg?.region_name||'new region')+' · Prepare and confirm when ready.'));}
    else if(awaitingBossConfirmation())put($('rift-transition'),transitionText('Boss tier cleared · Confirm to bank rewards and continue.'));
    if(awaitingBossRoomPause())put($('rift-transition'),transitionText('Boss room ahead · Prepare and confirm when ready.'));
    put($('rift-next'),awaitingNewRegionPause()?'Enter next region →':awaitingBossConfirmation()?'Confirm & continue →':awaitingBossRoomPause()?'Enter boss room →':$('rift-auto').checked?'Continue now →':run.room===2?'Bank & finish expedition':'Bank & continue →');
    setSafeDisabled($('rift-pause'),!playing&&run.status!=='fighting'&&run.status!=='cleared');
    updatePauseButton(playing);
    root.querySelectorAll('#rift-loadout select').forEach(el=>el.disabled=run.status==='fighting'||run.status==='cleared');
    if(practice){
      put($('rift-room'),'Practice · '+drillNames[practice]);put($('rift-objective'),run.practice.completed?'Drill complete':$('rift-practice-instructions').textContent);
      for(const id of ['rift-skills','rift-signatures','rift-class-coaching'])$(id).hidden=!['boss','class','skills','ranged','ultimate','resource'].includes(practice);
      put($('rift-practice-progress'),run.practice.completed?'Drill complete':practice==='banking'?(run.practice.checkpoint_ready?'Checkpoint reached · Review and bank practice tokens':(run.drops||[]).filter(drop=>drop.collected).length+'/3 tokens · Collect all, then reach the checkpoint'):practice==='pickup'?(run.drops||[]).filter(drop=>drop.collected).length+'/3 practice tokens collected':practice==='resource'?(run.resource||0)+'/3 charges · '+(run.practice.resource_cycles||0)+'/2 full-charge cycles':practice==='ultimate'?ultimatePracticeCue():practice==='skills'?'Free practice · '+(run.practice.hits||0)+' basic target hits · No time limit':practice==='class'?(run.resource||0)+'/3 charges · '+(run.practice.class_hits||0)+'/1 charged finisher hits':practice==='boss'?(run.enemies[0]?.name||'Boss')+' · Phase '+(run.enemies[0]?.phase||1)+' · '+Math.ceil(run.enemies[0]?.hp||0)+' HP'+(run.practice.slow_telegraphs?' · Longer warnings (2×)':''):practice==='hazard'?(run.practice.dodges||0)+'/3 clean pulses · '+(run.practice.hazard_intensity||'standard')+' · '+hazardPracticePhase(run):practice==='ranged'?(run.practice.ranged_hits||0)+'/3 projectile hits':practice==='perfect_guard'?(run.practice.perfect_guards||0)+'/3 perfect guards · Release guard between strikes':practice==='guard'?(run.stats.guards||0)+'/3 attacks blocked':practice==='combo'?run.practice.hits+' target hits · Finish a three-hit combo':Math.min(100,Math.round(run.player.x/run.practice.goal_x*100))+'% to finish');
      if(practice==='banking'){const secured=(run.drops||[]).filter(drop=>drop.banked).length,pending=(run.drops||[]).filter(drop=>drop.collected&&!drop.banked).length;put($('rift-practice-bank-receipt'),pending+' unbanked · '+secured+' secured practice tokens · No real rewards.');}
      if(practice==='ultimate'&&!run.practice.completed)put($('rift-objective'),ultimatePracticeCue());
      setSafeDisabled($('rift-practice-reset'),!ready||starting||practiceToolPending);practiceToolButtons();
      if(['complete','expired','defeated'].includes(run.status)){playing=false;clearTimeout(timer);resetInput();message(run.status==='complete'?'Drill complete.':run.status==='defeated'?'Try facing the attacker.':'Start a fresh drill.', 'Practice earns no loot or campaign records.', 'Try again',drillNames[practice]);silence();}
      return;
    }
    if(['defeated','complete','banked','expired'].includes(run.status)){
      playing=false;clearTimeout(timer);resetInput();$('rift-room-actions').hidden=true;
      if(audio.bossMusicActive)audio.fadeBossMusic?.(1.8);
      const lost=run.status==='defeated';
      message(lost?(run.room_objective?.kind==='split_defense'&&run.room_objective.lanes.some(l=>l.ward.hp<=0)?'A lane ward fell.':run.room_objective?.kind==='protect_lantern'&&run.room_objective.lantern.hp<=0?'The lantern went out.':'The rift takes its toll.'):'Returned from the ruins.',lost?'Unbanked finds were lost. This includes collected bag items and uncollected floor drops. Kept: '+run.banked_gold.toLocaleString()+' gold and '+run.banked_items.length.toLocaleString()+' banked '+(run.banked_items.length===1?'item':'items')+'. Your equipped gear is safe.':run.banked_gold.toLocaleString()+' gold and '+run.banked_items.length.toLocaleString()+' Abyss '+(run.banked_items.length===1?'item':'items')+' safely in your inventory.','Enter a new expedition',lost?'EXPEDITION ENDED':'REWARDS SECURED');
      if(run.status==='expired')message('A new chapter begins.','This expedition belongs to an earlier economy. Start a fresh run with your current character.','Enter a new expedition','EXPEDITION EXPIRED');
      const banner=$('rift-result-banner'),headingEl=$('rift-result-heading'),causeEl=$('rift-result-cause'),clearResult=$('rift-clear-result');
      if(run.status==='complete'){
        const fullCampaign=(run.completed_levels||[]).length>=100;
        if(fullCampaign){
          if(headingEl)headingEl.textContent='Full Campaign Cleared — The Abyss Conquered';
          if(causeEl)causeEl.textContent='All 100 campaign missions cleared across all regions!';
          if(clearResult){clearResult.textContent='Campaign complete: All 100 missions cleared';clearResult.hidden=false;}
        }else{
          const name=run.level?.name||('Mission '+(run.level?.id||1));
          if(headingEl)headingEl.textContent='Mission '+(run.level?.id||1)+' Cleared: '+name;
          if(causeEl)causeEl.textContent='All 3 encounter tiers secured.';
          if(clearResult){clearResult.textContent='Mission '+(run.level?.id||1)+' cleared: '+name;clearResult.hidden=false;}
        }
        if(banner)banner.hidden=false;
      }else if(run.status==='banked'){
        if(headingEl)headingEl.textContent='Voluntary Exit at Checkpoint';
        if(causeEl)causeEl.textContent='Safely banked rewards before venturing deeper.';
        if(clearResult){clearResult.textContent='Expedition banked: voluntary exit at checkpoint';clearResult.hidden=false;}
        if(banner)banner.hidden=false;
      }else if(run.status==='defeated'){
        const finalHit=run.defeated_by_hazard?('Hazard: '+run.defeated_by_hazard.kind+(run.defeated_by_hazard.jumpable?' (jump to evade)':' (move clear)')):run.defeated_by_boss?('Boss: '+run.defeated_by_boss):run.defeated_by_enemy?('Enemy: '+run.defeated_by_enemy):(run.room_objective?.kind==='split_defense'&&run.room_objective.lanes.some(l=>l.ward.hp<=0)?'Lane ward destroyed':(run.room_objective?.kind==='protect_lantern'&&run.room_objective.lantern.hp<=0?'Lantern extinguished':'Combat damage'));
        const exactCause=run.defeat_cause||finalHit;
        if(headingEl)headingEl.textContent='Expedition Defeat';
        if(causeEl)causeEl.textContent='Cause: '+exactCause+' · Final hit: '+(run.defeated_by_boss||(run.defeated_by_hazard?run.defeated_by_hazard.kind:run.defeated_by_enemy||'Fatal strike'));
        if(clearResult){clearResult.textContent='Expedition defeat: '+exactCause;clearResult.hidden=false;}
        if(banner)banner.hidden=false;
      }else{
        if(banner)banner.hidden=true;
        if(clearResult)clearResult.hidden=true;
      }
      const rewardsEl=$('rift-result-rewards'),bankedVal=$('rift-rewards-banked-val'),lostVal=$('rift-rewards-lost-val'),lostGroup=$('rift-rewards-lost-group');
      if(rewardsEl&&bankedVal&&lostVal&&lostGroup){
        bankedVal.textContent=(run.banked_gold||0).toLocaleString()+' gold · '+(run.banked_items?.length||0)+' '+((run.banked_items?.length===1)?'item':'items');
        if(lost){
          const lostGold=run.gold||0;
          const lostDrops=(run.drops||[]).filter(d=>!d.banked).length;
          lostVal.textContent=lostGold.toLocaleString()+' gold · '+lostDrops+' '+((lostDrops===1)?'item':'items');
          lostGroup.hidden=false;
        }else{
          lostGroup.hidden=true;
        }
        rewardsEl.hidden=false;
      }
      $('rift-result-actions').hidden=!run.level||run.status==='expired';
      const retryMissionBtn=$('rift-retry-mission');
      if(retryMissionBtn){
        const missionId=run.level?.id||selectedLevel;
        retryMissionBtn.textContent='Retry '+(run.level?.name||('Mission '+missionId));
        retryMissionBtn.hidden=!run.level||run.status==='expired';
      }
      $('rift-retry-boss').hidden=!lost||!run.level||!run.encounter_plan?.[run.room]?.some(enemy=>enemy.kind==='boss');
      $('rift-replay').hidden=!run.level||!['complete','banked'].includes(run.status)||run.room!==2;
      $('rift-replay').textContent='Replay mission '+(run.level?.id||1);
      const campaignBtn=$('rift-result-campaign');
      if(campaignBtn)campaignBtn.hidden=run.status==='expired';
      const buildBtn=$('rift-result-build');
      if(buildBtn)buildBtn.hidden=!lost||run.status==='expired';
      const practiceBossBtn=$('rift-practice-boss-link');
      if(practiceBossBtn){
        const bossName=run.defeated_by_boss||run.encounter_plan?.[run.room]?.find(enemy=>enemy.kind==='boss')?.name;
        practiceBossBtn.hidden=!lost||!bossName;
        if(bossName)practiceBossBtn.textContent='Practice '+bossName;
      }
      root.querySelectorAll('#rift-loadout select').forEach(el=>el.disabled=false);
      setTimeout(()=>{if(!playing)silence();},1500);
    }
  }
  async function send(kind) {
    if(busy)return false;
    busy=true;if(kind!=='step')setSafeDisabled($('rift-practice-reset'),true);practiceToolButtons();
    const previousReceipt=run?{id:run.id,banked_gold:run.banked_gold,banked_items:[...run.banked_items]}:null;
    const banking=['bank','exit','next','advance'].includes(kind);if(banking)window.RiftLoot.banking('pending');
    const body={kind,run_id:run?.id||'',request_id:crypto.randomUUID(),revision:(run?.revision||0)+1,input:kind==='step'?input():{}};
    const inputTiming=kind==='step'?takeInputTiming(body.input):null;
    if(practice==='hazard'&&['start','practice_reset'].includes(kind))body.hazard_intensity=$('rift-hazard-intensity').value;
    if(kind==='practice_spawn')body.enemy_name=$('rift-practice-enemy').value;
    if(practice==='boss'&&['start','practice_reset'].includes(kind)){body.boss_name=$('rift-practice-boss').value;body.boss_phase=Number($('rift-practice-phase').value);body.slow_telegraphs=$('rift-practice-slow').checked;}
    if(kind==='start'){body.level_id=selectedLevel;body.skills=[...root.querySelectorAll('#rift-loadout select')].map(el=>el.value).filter(Boolean);}
    root.querySelectorAll(kind==='step'?'#rift-start':'#rift-next,#rift-exit,#rift-start').forEach(btn=>setSafeDisabled(btn,true));
    try {
      const data=await request('POST',body,inputTiming);await renderer.prepareRun(data.run);update(data.run,false);confirmInput(inputTiming);
      if(banking){if(window.RiftLoot.confirmBank(previousReceipt,data.run))audio.play('bank',0);window.RiftLoot.banking('confirmed');}
      return true;
    } catch(error){
      playing=false;resetInput();clearTimeout(timer);silence();if(run)update(run,true);status(error.message);
      if(banking)window.RiftLoot.banking('uncertain');
      message('Your expedition is saved.',banking?'Reward delivery is unconfirmed. Recover the saved expedition to check what was banked. '+error.message:error.message,'Recover expedition','CONNECTION PAUSED');$('rift-start').dataset.recover='true';return false;
    } finally {busy=false;practiceToolButtons();setSafeDisabled($('rift-practice-reset'),!practice||!ready||!run);window.RiftLoadouts.refresh();setSafeDisabled($('rift-start'),!ready);root.querySelectorAll('#rift-next,#rift-exit').forEach(btn=>setSafeDisabled(btn,!ready||checkpointPending));}
  }
  async function checkpoint(kind){
    if(checkpointPending||!playing||run?.status!=='cleared')return false;
    const identity=run.id,room=run.room,mission=run.level?.id;
    checkpointPending=true;root.querySelectorAll('#rift-next,#rift-exit').forEach(btn=>setSafeDisabled(btn,true));
    try{
      while(busy)await new Promise(resolve=>setTimeout(resolve,20));
      if(!playing||document.hidden||run?.id!==identity||run.status!=='cleared'||run.room!==room||run.level?.id!==mission)return false;
      return await send(kind);
    }finally{checkpointPending=false;root.querySelectorAll('#rift-next,#rift-exit').forEach(btn=>setSafeDisabled(btn,!ready));}
  }
  async function loop(){
    if(!playing)return;
    audio.tick?.();
    if(!busy&&!checkpointPending){
      if(run?.status==='cleared' && $('rift-auto').checked){
        if(awaitingBossConfirmation()||awaitingBossRoomPause()||awaitingNewRegionPause()){timer=setTimeout(loop,85);return;}
        if(!clearedAt){clearedAt=performance.now();countdownAnnounced=-1;}
        const remaining=Math.max(0,transitionDelay-(performance.now()-clearedAt)/1000);
        const next=run.room===2?levels.find(level=>level.id===(run.level?.id||0)+1)?.name:rooms[run.room+1];
        const nextLabel=next||'campaign complete';
        $('rift-transition').textContent=transitionText(remaining>0?'Next: '+nextLabel+' · '+remaining.toFixed(1)+'s':'Banking rewards…');
        if(countdownAnnounced===-1){
          countdownAnnounced=Math.ceil(remaining);
          put($('rift-announcer'),'Next tier: '+(next||'next chamber')+' in '+Math.ceil(transitionDelay)+' seconds. Automatic transition enabled.');
        }else if(remaining===0&&countdownAnnounced!==0){
          countdownAnnounced=0;
          put($('rift-announcer'),'Banking rewards and advancing to '+(next||'next tier')+'…');
        }else if(transitionDelay>=5){
          const sec=Math.ceil(remaining);
          if((sec===3||sec===1)&&sec<countdownAnnounced){
            countdownAnnounced=sec;
            put($('rift-announcer'),sec+' seconds until next tier.');
          }
        }
        if(remaining===0)await send('advance');
      }else{countdownAnnounced=-1;await send('step');}
    }
    if(playing)timer=setTimeout(loop,85);
  }
  async function pause(){
    if(!playing)return;
    playing=false;clearTimeout(timer);resetInput();silence();
    if(run)window.RiftHUD.update(run,false);
    clearedAt=0;countdownAnnounced=-1;$('rift-transition').hidden=true;
    // Wait for the single pending input request, then persist the pause.
    while(busy)await new Promise(resolve=>setTimeout(resolve,20));
    if(!run||!['fighting','cleared'].includes(run.status))return;
    if(await send('pause')&&['fighting','cleared'].includes(run.status)){message('A moment by the lantern.','Take your time. The expedition will wait.','Resume expedition','PAUSED');updatePauseButton(false);$('rift-room-actions').hidden=true;}
  }
  function dismissVirtualKeyboard(){
    const el=document.activeElement;
    if(el&&typeof el.blur==='function'&&['INPUT','TEXTAREA'].includes(el.tagName)&&!['checkbox','radio','range','button','submit'].includes(el.type)){
      el.blur();
    }
  }
  async function begin(){
    dismissVirtualKeyboard();
    if(busy||starting||practiceToolPending||controls.opened)return;
    if($('rift-start').dataset.retry){
      if($('rift-start').dataset.artworkRetry==='true'){location.reload();return;}
      delete $('rift-start').dataset.retry;$('rift-start').disabled=true;await load();return;
    }
    if(!ready)return;
    if(['class','resource'].includes(practice)&&!classPracticeInstructions())return;
    if($('rift-start').dataset.recover){delete $('rift-start').dataset.recover;silence();await load();return;}
    starting=true;const intent=++startIntent;
    try{
      const areaIdx=(run?.level?.region||0)*3+(run?.room||0);lastAudioArea=areaIdx;
      try{await audio.setActive(true,areaIdx);}catch(_){silence();}
      const bossAlive=run?.enemies?.some(e=>e.kind==='boss'&&e.hp>0)||run?.room===2;
      if(bossAlive&&!audio.bossMusicActive)audio.startBossMusic?.();
      if(intent!==startIntent||document.hidden){silence();return;}
      const resume=run&&(run.status==='fighting'||run.status==='cleared');
      if(await send(resume?'resume':practice&&run&&run.status!=='expired'?'practice_reset':'start')){
        playing=true;
        dismissVirtualKeyboard();
        if(intent!==startIntent||document.hidden){await pause();return;}
        $('rift-campaign').open=false;$('rift-overlay').hidden=true;$('rift-pause').disabled=false;update(run,true);$('rift-canvas').focus();status(controls.description());clearTimeout(timer);loop();
      }
    }finally{starting=false;}
  }
  function updateCampaign(){
    if(practice)return;
    const active=run&&['fighting','cleared'].includes(run.status), completed=run?.completed_levels||[];
    const key=[selectedLevel,active,completed.join(','),JSON.stringify(run?.mission_history||{}),challenge?.key].join('|');
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
    window.RiftMission.update(level,run&&active?run.build:build,!!active,challenge);
  }
  function campaign(){
    if(practice)return;
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
    window.RiftCampaignTools.init(levels,challenge);
    const preferred=window.RiftMission.preferred(window.RiftCampaignTools.preferred(),levels);
    selectedLevel=run?.level&&['fighting','cleared'].includes(run.status)?run.level.id:preferred;campaignKey='';updateCampaign();renderer.preview(levels[selectedLevel-1]);
  }
  function ultimatePracticeCue(){return run?.practice?.ultimate_window?'Opening active · Use your ultimate now. Projectiles must arrive before it closes.':'Prepare · Wait for the two-second opening. It begins three seconds into each six-second cycle.';}
  function refreshPracticeInstructions(){
    if(!practice)return;
    $('rift-practice-instructions').textContent=practice==='touch'?'Calibrate on-screen touch controls, test directional dragging and practice simultaneous movement and combat actions against the training target.':practice==='pickup'?'Move with '+['up','left','down','right'].map(action=>controls.label(action)).join(', ')+'. Bring each practice token inside the gold pickup circle. Collect all three. These tokens have no gold or gear value and never enter your inventory.':practice==='ranged'?'Face the distant target and line up in its lane. Land three projectiles using your equipped ranged abilities. Melee attacks and missed shots do not count. Use the recovery controls to refill mana or reset cooldowns.':practice==='skills'?'Test your equipped skills, class abilities and ultimate on the immortal target. You start at 60% health to test healing. Use the recovery controls or reset to try again. The default target never attacks. Spawn a selected Abyss monster to practice against a live opponent, or clear the arena. This lane has no finish line.':practice==='class'?'Build charges with your equipped class builder, then land your finisher.':practice==='boss'?'Defeat the selected Abyss boss. Jump or move clear of slams; evade volleys or face them to guard. Use your equipped skills. Reset to retry the selected starting phase.':practice==='hazard'?'Avoid three consecutive hazard pulses. Each warning appears under you: move clear or jump with '+controls.label('jump')+' before it flashes. Taking damage resets your streak.':practice==='perfect_guard'?'Face the trainer. Press '+controls.label('guard')+' just before a strike lands: the first 0.22 seconds count as a perfect guard. Release between strikes. Land three perfect guards; ordinary blocks do not count.':practice==='guard'?'Face the attacker and hold '+controls.label('guard')+' to block three strikes. Attacks from behind bypass guard. Turn with the movement keys.':practice==='combo'?'Face the training target and land three consecutive basic strikes with '+controls.label('attack')+'.':practice==='jump'?'Move right with '+controls.label('right')+' and jump the cover with '+controls.label('jump')+'. Reach the finish line.':'Move to the finish line with '+controls.label('right')+'. Use the other movement keys to explore the lane.';
    if(practice==='banking')put($('rift-practice-instructions'),'Move with '+['up','left','down','right'].map(action=>controls.label(action)).join(', ')+'. Collect all three practice tokens, then cross the CHECKPOINT line. Review unbanked tokens and choose Bank practice tokens & finish. These tokens have no gold or gear value. Only a confirmed receipt marks them secured.');
    if(['pickup','banking'].includes(practice))put($('rift-pickup-training'),'The circle stays visible throughout this demonstration and uses the same pickup reach as campaign loot.');
    if(practice==='ultimate'){
      const ultimate=(run?.build||build)?.ultimate;
      put($('rift-practice-instructions'),ultimate?'Time '+ultimate.name+' ('+controls.label('ultimate')+') for the two-second opening, starting three seconds into each six-second cycle. '+(ultimate.reference?.target==='self'?'Cast your healing or defensive ultimate during the opening.':'Face the target and stay in its lane. Your ultimate must hit during the opening; allow for projectile travel.')+' Normal mana costs and cooldowns apply. Recovery controls pause the drill to refill mana or reset cooldowns. One successful timing completes it.':'Equip an ultimate in Abyss, then return to this timing drill.');
    }
    if(['class','resource'].includes(practice)&&(run?.build||build))classPracticeInstructions();
  }
  window.addEventListener('riftbindingschange',()=>{
    if(!practice||['ranged','skills','boss'].includes(practice))return;
    refreshPracticeInstructions();
    if(!$('rift-overlay').hidden&&(!run||run.status==='fighting'))put($('rift-overlay-copy'),$('rift-practice-instructions').textContent);
    if(run&&!run.practice.completed)put($('rift-objective'),$('rift-practice-instructions').textContent);
  });
  function classPracticeInstructions(){
    if(!['class','resource'].includes(practice))return;
    const current=run?.build||build,signatures=current?.signatures||[],builder=signatures.find(s=>s.role==='builder'),finisher=signatures.find(s=>s.role==='finisher');
    if(!builder||!finisher){put($('rift-practice-instructions'),'Unlock and equip a class builder and finisher in Abyss before starting this drill. '+window.RiftAbilities.signatureHelp(current));return false;}
    const key=s=>controls.label('signature'+signatures.indexOf(s));
    if(practice==='resource'){put($('rift-practice-instructions'),'Build three charges with '+builder.name+' ('+key(builder)+'), then spend all three with '+finisher.name+' ('+key(finisher)+'). Complete two full-charge cycles in a row. Spending fewer than three charges resets your streak. Normal mana costs and cooldowns apply; recovery controls pause the drill and preserve progress.');return true;}
    put($('rift-practice-instructions'),'Use '+builder.name+' ('+key(builder)+') to build charges, then land '+finisher.name+' ('+key(finisher)+') on the training target. Face the target and stay in its lane. '+(current.skills.length?'Your equipped skills are also available: '+current.skills.map(s=>s.name).join(', ')+'. ':'')+'One charged finisher hit completes the drill.'+(run?.practice?.target_hint?' '+run.enemies[0].name+': '+run.practice.target_hint:''));return true;
  }
  function classPrimer(){
    classPracticeInstructions();
    window.RiftClassCompare.visibility(!!practice||playing&&run?.status==='fighting');
    const node=$('rift-class-primer'),current=run&&['fighting','cleared'].includes(run.status)?run.build:build;
    node.hidden=playing||!current||!!practice&&!['boss','class','skills','ranged','ultimate','resource'].includes(practice);
    if(!current)return;
    const signatures=current.signatures||[],builder=signatures.find(s=>s.role==='builder'),finisher=signatures.find(s=>s.role==='finisher');
    put(node.querySelector('strong'),window.RiftRecords.classIdentity(current)+' · Combat primer');
    const help=window.RiftAbilities.signatureHelp(current);$('rift-signature-help').hidden=!help;$('rift-signature-help-link').hidden=!help;put($('rift-signature-help'),help);
    const key=skill=>controls.label('signature'+signatures.indexOf(skill));
    const copy=builder&&finisher?'Use '+builder.name+' ('+key(builder)+') to build up to three '+(current.resource||'class')+' charges, then spend them with '+finisher.name+' ('+key(finisher)+').':builder?'Use '+builder.name+' ('+key(builder)+') to build class charges. Unlock your finisher in Abyss.':finisher?'Your finisher is '+finisher.name+' ('+key(finisher)+'). Equip a builder in Abyss to gain charges.':'Class abilities unlock through Abyss progression. Use basic attacks and your equipped skills.';
    put(node.querySelector('p'),copy+' Move out of attack warnings; guard facing incoming attacks.'+(current.class==='vanguard'?' Perfect-guard an enemy attack to gain one charge per guard raise (maximum three).':'')+(current.class==='berserker'?' Fury grants +15% damage at impact while alive at 30% HP or below.':'')+(current.class==='marksman'?' A mark lasts until your finisher, a new target mark, target defeat or escape, or a new tier.':''));
  }
  function loadout(){
    refreshPracticeInstructions();
    classPrimer();
    renderer.build(build);
    $('rift-build').textContent=build.name+' · Level '+build.level+' · '+build.class+'\n'+build.weapon;
    $('rift-sequence').textContent=build.sequence||'';
    $('rift-style').textContent=window.RiftRecords.classIdentity(build);$('rift-resource').textContent=build.resource?build.resource+' · Q builds / E spends':'Class abilities unlock through Abyss progression';
    $('rift-gear').replaceChildren();build.gear.forEach(name=>text('li',name,$('rift-gear')));
    if(!build.gear.length)text('li','No equipment yet',$('rift-gear'));
    $('rift-loadout').replaceChildren();
    for(let i=0;i<Math.min(3,build.skills.length);i++){
      const label=text('label','Skill '+(i+1),$('rift-loadout')),select=document.createElement('select');select.setAttribute('aria-label','Expedition skill '+(i+1));
      text('option','None',select).value='';build.skills.forEach(s=>{text('option',s.name,select).value=s.id;});select.value=run?(run.build.skills[i]?.id||''):build.skills[i].id;label.append(select);
      select.addEventListener('change',()=>{const others=[...root.querySelectorAll('#rift-loadout select')].filter(el=>el!==select);if(select.value&&others.some(el=>el.value===select.value)){select.value='';status(statusCopy.uniqueSkill);}});
    }
    window.RiftLoadouts.init(build,()=>busy||starting||!!run&&['fighting','cleared'].includes(run.status));
  }
  async function load(){
    ready=false;$('rift-start').disabled=true;
    const generation=++loadGeneration;
    const previousRequest=pendingRequest;
    pendingRead?.abort();
    // Keep only the latest load after the current request (including its body) settles.
    await previousRequest;
    if(generation!==loadGeneration)return;
    let artworkFailed=false;
    try{
      // Cold atlas transfers can starve the initial read's bounded request timer.
      // Retry that read once after artwork settles; never retry a mutation here.
      const initialRead=request('GET').catch(error=>{if(error?.name==='AbortError')return null;throw error;});
      const [,initialData]=await Promise.all([renderer.ready.catch(error=>{artworkFailed=true;throw error;}),initialRead]);
      if(generation!==loadGeneration)return;
      const data=initialData===null?await request('GET'):initialData;
      await Promise.all([renderer.prepareBuild(data.build),renderer.prepareRun(data.run)]).catch(error=>{artworkFailed=true;throw error;});
      if(generation!==loadGeneration)return;
      window.RiftRecords.init(data.class_names);window.RiftClassChallenges.update(data.run);window.RiftClassCompare.init(data.class_options,data.run?.build?.class||data.build?.class);window.RiftObjectives.init(data.objective_options||[]);build=data.build;rooms=data.rooms;run=data.run;levels=data.levels||[];challenge=data.challenge||null;window.RiftLoot.init(data.rarities||[]);loadout();campaign();window.RiftBestiary.render(data.bestiary||[],run);if(practice==='skills'){const select=$('rift-practice-enemy');select.replaceChildren();(data.bestiary||[]).forEach(unit=>{text('option',unit.name,select).value=unit.name;});const saved=run?.enemies?.find(enemy=>enemy.id==='practice-enemy');if(saved&&[...select.options].some(option=>option.value===saved.name))select.value=saved.name;}if(practice==='hazard')$('rift-hazard-intensity').value=run?.practice?.hazard_intensity||'standard';if(practice==='boss'){const select=$('rift-practice-boss');select.replaceChildren();(data.bestiary||[]).filter(unit=>unit.kind==='boss').forEach(unit=>{text('option',unit.name,select).value=unit.name;});if(run?.practice?.boss_start){const saved=run.practice.boss_start;if(![...select.options].some(option=>option.value===saved.name))text('option',saved.name,select).value=saved.name;select.value=saved.name;$('rift-practice-phase').value=String(saved.phase||1);$('rift-practice-slow').checked=!!run.practice.slow_telegraphs;}}ready=true;
      if(run){update(run,true);if(['fighting','cleared'].includes(run.status))message('Your expedition awaits.','Resume from the last confirmed moment. Your expedition bag is still here.','Resume expedition','SAVED EXPEDITION');}
      else if(selectedLevel===1){$('rift-start').textContent='Enter the ruins →';$('rift-start').disabled=false;}
      else{const level=levels.find(l=>l.id===selectedLevel);message(level.name.split(' · ')[1],level.tactic+'. Three tiers, one Abyss boss.','Enter mission '+level.id,level.region_name);}
      if(practice)message(drillNames[practice],$('rift-practice-instructions').textContent,run&&['fighting','cleared'].includes(run.status)?'Resume drill':'Start drill','PRACTICE');
      if(['class','resource'].includes(practice)&&!classPracticeInstructions()){$('rift-start').disabled=true;$('rift-start').textContent='Class abilities required';}
      if(practice==='ultimate'&&!(run?.build||build).ultimate){$('rift-start').disabled=true;$('rift-start').textContent='Ultimate required';}
      if(practice==='ranged'&&![...((run?.build||build).skills||[]),...((run?.build||build).signatures||[]),(run?.build||build).ultimate].some(skill=>skill?.reference?.target==='projectile')){$('rift-start').disabled=true;$('rift-start').textContent='Ranged ability required';put($('rift-practice-instructions'),'Equip a projectile ability in Abyss, then return to ranged practice. Your current build has no ranged projectile.');}
      window.RiftLoot.banking('reloaded');
      status(root.dataset.fixture?statusCopy.fixtureReady:statusCopy.characterReady);
      if(root.dataset.fixture)$('rift-overlay-note').textContent='Local playtest · Sample character · Isolated rewards';
      if(practice){$('rift-overlay-note').textContent='Your Abyss build · Practice only · No rewards';status(statusCopy.practiceReady($('rift-practice-instructions').textContent));}
    }catch(error){if(generation!==loadGeneration)return;silence();ready=false;$('rift-start').textContent=artworkFailed?'Reload artwork':'Retry loading';$('rift-start').dataset.retry='true';$('rift-start').dataset.artworkRetry=String(artworkFailed);$('rift-start').disabled=false;status(error.message);}
  }
  const moveLabels = {
    left: ['Move left', 'Moving left (holding)'],
    right: ['Move right', 'Moving right (holding)'],
    up: ['Move up', 'Moving up (holding)'],
    down: ['Move down', 'Moving down (holding)']
  };
  const touchAlignments=new Set(['center','left','right']);let touchAlignment='center';
  try{const saved=JSON.parse(localStorage.getItem('riftTouchLayout'));if(saved?.version===1&&touchAlignments.has(saved.alignment))touchAlignment=saved.alignment;}catch(_){}
  const touchModes=new Set(['pad','joystick']);let touchMode='pad';
  try{const savedMode=localStorage.getItem('riftTouchMode');if(touchModes.has(savedMode))touchMode=savedMode;}catch(_){}
  let touchOffsets={touch:{x:0,y:0},actions:{x:0,y:0}};
  try{
    const savedOffsets=JSON.parse(localStorage.getItem('riftTouchOffsets'));
    if(savedOffsets&&typeof savedOffsets==='object'){
      if(Number.isFinite(savedOffsets.touch?.x)&&Number.isFinite(savedOffsets.touch?.y))touchOffsets.touch={x:savedOffsets.touch.x,y:savedOffsets.touch.y};
      if(Number.isFinite(savedOffsets.actions?.x)&&Number.isFinite(savedOffsets.actions?.y))touchOffsets.actions={x:savedOffsets.actions.x,y:savedOffsets.actions.y};
    }
  }catch(_){}
  function paintTouchOffsets(){
    root.style.setProperty('--rift-touch-offset-x',touchOffsets.touch.x+'px');
    root.style.setProperty('--rift-touch-offset-y',touchOffsets.touch.y+'px');
    root.style.setProperty('--rift-action-offset-x',touchOffsets.actions.x+'px');
    root.style.setProperty('--rift-action-offset-y',touchOffsets.actions.y+'px');
  }
  let touchOpacity=100;
  try{const o=Number(localStorage.getItem('riftTouchOpacity'));if(o>=20&&o<=100)touchOpacity=o;}catch(_){}
  let touchScale=1.0;
  try{const s=Number(localStorage.getItem('riftTouchScale'));if(s>=0.7&&s<=1.6)touchScale=s;}catch(_){}
  const touchLayoutGroup=document.createElement('div');touchLayoutGroup.className='rift-touch-layout-setting';
  const touchModeLabel=document.createElement('label'),touchModeSelect=document.createElement('select');
  touchModeSelect.id='rift-touch-mode';touchModeLabel.htmlFor=touchModeSelect.id;touchModeLabel.textContent='Movement control style';
  for(const [val,label] of [['pad','Fixed directional pad'],['joystick','Virtual joystick']]){const opt=document.createElement('option');opt.value=val;opt.textContent=label;touchModeSelect.append(opt);}
  touchModeSelect.value=touchMode;
  const touchLayoutLabel=document.createElement('label'),touchLayoutSelect=document.createElement('select');
  touchLayoutSelect.id='rift-touch-layout';touchLayoutLabel.htmlFor=touchLayoutSelect.id;touchLayoutLabel.textContent='Movement pad alignment';
  for(const value of touchAlignments){const option=document.createElement('option');option.value=value;option.textContent=value[0].toUpperCase()+value.slice(1);touchLayoutSelect.append(option);}
  const touchOpacityLabel=document.createElement('label'),touchOpacityInput=document.createElement('input');
  touchOpacityInput.id='rift-touch-opacity';touchOpacityInput.type='range';touchOpacityInput.min='20';touchOpacityInput.max='100';touchOpacityInput.step='5';touchOpacityInput.value=String(touchOpacity);
  touchOpacityLabel.htmlFor=touchOpacityInput.id;touchOpacityLabel.append('Movement pad opacity ',touchOpacityInput);
  const touchScaleLabel=document.createElement('label'),touchScaleSelect=document.createElement('select');
  touchScaleSelect.id='rift-touch-scale';touchScaleLabel.htmlFor=touchScaleSelect.id;touchScaleLabel.textContent='Movement pad scale';
  for(const [val,text] of [['0.8','80%'],['1','100% (default)'],['1.2','120%'],['1.4','140%']]){const opt=document.createElement('option');opt.value=val;opt.textContent=text;touchScaleSelect.append(opt);}
  touchScaleSelect.value=String(touchScale);
  const touchEditBtn=document.createElement('button');
  touchEditBtn.id='rift-edit-touch-layout';touchEditBtn.type='button';touchEditBtn.textContent='Drag-to-reposition editor';
  const touchLayoutReset=document.createElement('button');
  touchLayoutReset.id='rift-reset-touch-layout';touchLayoutReset.type='button';touchLayoutReset.textContent='Reset touch layout';
  function paintTouchMode(){root.dataset.touchMode=touchMode;touchModeSelect.value=touchMode;const joystickEl=$('rift-touch-joystick');if(joystickEl)joystickEl.hidden=touchMode!=='joystick';const padEl=root.querySelector('.rift-touch');if(padEl)padEl.hidden=touchMode==='joystick';}
  function paintTouchLayout(){root.dataset.touchLayout=touchAlignment;touchLayoutSelect.value=touchAlignment;}
  function paintTouchOpacity(){root.style.setProperty('--rift-touch-opacity',String(touchOpacity/100));touchOpacityInput.value=String(touchOpacity);}
  function paintTouchScale(){root.style.setProperty('--rift-touch-scale',String(touchScale));touchScaleSelect.value=String(touchScale);}
  touchModeSelect.addEventListener('change',()=>{
    if(!touchModes.has(touchModeSelect.value))return;
    resetInput();touchMode=touchModeSelect.value;paintTouchMode();
    try{localStorage.setItem('riftTouchMode',touchMode);}catch(_){}
  });
  touchLayoutSelect.addEventListener('change',()=>{
    if(!touchAlignments.has(touchLayoutSelect.value))return;
    resetInput();touchAlignment=touchLayoutSelect.value;paintTouchLayout();
    try{localStorage.setItem('riftTouchLayout',JSON.stringify({version:1,alignment:touchAlignment}));}catch(_){}
  });
  touchOpacityInput.addEventListener('input',()=>{touchOpacity=Number(touchOpacityInput.value);paintTouchOpacity();});
  touchOpacityInput.addEventListener('change',()=>{touchOpacity=Number(touchOpacityInput.value);paintTouchOpacity();try{localStorage.setItem('riftTouchOpacity',String(touchOpacity));}catch(_){}});
  touchScaleSelect.addEventListener('change',()=>{touchScale=Number(touchScaleSelect.value);paintTouchScale();try{localStorage.setItem('riftTouchScale',String(touchScale));}catch(_){}});
  touchLayoutReset.addEventListener('click',()=>{
    touchModeSelect.value='pad';touchMode='pad';try{localStorage.setItem('riftTouchMode','pad');}catch(_){}paintTouchMode();
    touchLayoutSelect.value='center';touchLayoutSelect.dispatchEvent(new Event('change'));
    touchOpacity=100;try{localStorage.setItem('riftTouchOpacity','100');}catch(_){}paintTouchOpacity();
    touchScale=1.0;try{localStorage.setItem('riftTouchScale','1');}catch(_){}paintTouchScale();
    touchOffsets={touch:{x:0,y:0},actions:{x:0,y:0}};try{localStorage.removeItem('riftTouchOffsets');}catch(_){}paintTouchOffsets();
  });
  touchLayoutGroup.append(touchModeLabel,touchModeSelect,touchLayoutLabel,touchLayoutSelect,touchOpacityLabel,touchScaleLabel,touchScaleSelect,touchEditBtn,touchLayoutReset);
  document.querySelector('.rift-settings').append(touchLayoutGroup);
  paintTouchMode();paintTouchLayout();paintTouchOpacity();paintTouchScale();paintTouchOffsets();
  let touchEditing=false;
  let backupOffsets=null;
  function enterTouchEditor(){
    if(controls.opened)$('rift-controls-dialog').close();
    backupOffsets={touch:{...touchOffsets.touch},actions:{...touchOffsets.actions}};
    touchEditing=true;
    root.dataset.touchEditing='true';
    const editor=$('rift-touch-editor');
    if(editor){
      editor.hidden=false;
      editor.scrollIntoView({behavior:'smooth',block:'nearest'});
    }
    resetInput();
  }
  function exitTouchEditor(save){
    if(!touchEditing)return;
    if(save){
      try{localStorage.setItem('riftTouchOffsets',JSON.stringify(touchOffsets));}catch(_){}
    }else if(backupOffsets){
      touchOffsets={touch:{...backupOffsets.touch},actions:{...backupOffsets.actions}};
      paintTouchOffsets();
    }
    touchEditing=false;
    delete root.dataset.touchEditing;
    const editor=$('rift-touch-editor');
    if(editor)editor.hidden=true;
    root.querySelectorAll('.rift-dragging').forEach(el=>el.classList.remove('rift-dragging'));
    resetInput();
  }
  touchEditBtn.addEventListener('click',enterTouchEditor);
  const saveTouchBtn=$('rift-save-touch-layout');
  if(saveTouchBtn)saveTouchBtn.addEventListener('click',()=>exitTouchEditor(true));
  const cancelTouchBtn=$('rift-cancel-touch-layout');
  if(cancelTouchBtn)cancelTouchBtn.addEventListener('click',()=>exitTouchEditor(false));

  function makeDraggable(el,clusterKey){
    if(!el)return;
    let dragPointerId=null;
    let startPointer={x:0,y:0};
    let startOffset={x:0,y:0};
    el.addEventListener('pointerdown',event=>{
      if(!touchEditing||event.button>0)return;
      event.preventDefault();
      event.stopPropagation();
      dragPointerId=event.pointerId;
      try{el.setPointerCapture(event.pointerId);}catch(_){}
      el.classList.add('rift-dragging');
      startPointer={x:event.clientX,y:event.clientY};
      startOffset={...touchOffsets[clusterKey]};
    });
    el.addEventListener('pointermove',event=>{
      if(dragPointerId!==event.pointerId)return;
      event.preventDefault();
      const dx=event.clientX-startPointer.x;
      const dy=event.clientY-startPointer.y;
      touchOffsets[clusterKey].x=Math.round(Math.max(-240,Math.min(240,startOffset.x+dx)));
      touchOffsets[clusterKey].y=Math.round(Math.max(-160,Math.min(160,startOffset.y+dy)));
      paintTouchOffsets();
    });
    const endDrag=event=>{
      if(dragPointerId===event.pointerId){
        if(el.hasPointerCapture(event.pointerId)){
          try{el.releasePointerCapture(event.pointerId);}catch(_){}
        }
        dragPointerId=null;
        el.classList.remove('rift-dragging');
      }
    };
    ['pointerup','pointercancel','lostpointercapture'].forEach(name=>el.addEventListener(name,endDrag));
  }
  function hold(button,value){
    button.dataset.action=value;
    button.addEventListener('contextmenu',event=>{if(playing)event.preventDefault();});
    if(!button.hasAttribute('aria-pressed'))button.setAttribute('aria-pressed','false');
    const moveKey=button.dataset.move;
    if(moveKey&&moveLabels[moveKey]){button.setAttribute('aria-label',moveLabels[moveKey][0]);button.title=moveLabels[moveKey][0];}
    button.addEventListener('pointerdown',event=>{
      if(touchEditing||!playing||button.disabled||button.getAttribute('aria-disabled')==='true')return;
      event.preventDefault();button.setPointerCapture(event.pointerId);touchPointers.set(event.pointerId,{button,value});
      if(!touch.has(value)){touch.add(value);taps.add(value);markInput(value);window.RiftIntents.press(value);}
      button.classList.add('rift-held');button.setAttribute('aria-pressed','true');button.dataset.pressed='true';
      if(moveKey&&moveLabels[moveKey])button.setAttribute('aria-label',moveLabels[moveKey][1]);
    });
    button.addEventListener('click',event=>{
      if(event.detail===0&&playing&&!button.disabled&&button.getAttribute('aria-disabled')!=='true'){
        markInput(value);if(value==='guard'&&controls.toggleGuard)toggleGuard();
        else{taps.add(value);window.RiftIntents.press(value);}
      }
    });
    const release=event=>{
      const pointer=touchPointers.get(event.pointerId);
      if(!pointer||pointer.button!==button)return;
      touchPointers.delete(event.pointerId);
      const remaining=[...touchPointers.values()];
      const actionHeld=remaining.some(pointer=>pointer.value===value);
      if(!actionHeld){
        if(event.type==='pointerup'&&value==='guard'&&controls.toggleGuard)toggleGuard();
        if(event.type!=='pointerup'){taps.delete(value);window.RiftIntents.cancel(value);}
        touch.delete(value);
      }
      if(!remaining.some(pointer=>pointer.button===button)){
        button.classList.remove('rift-held');
        if(value!=='guard'||!controls.toggleGuard){button.setAttribute('aria-pressed','false');delete button.dataset.pressed;}
        if(moveKey&&moveLabels[moveKey])button.setAttribute('aria-label',moveLabels[moveKey][0]);
      }
      if(value==='guard')guardDisplay();
    };
    button.addEventListener('pointermove',event=>{
      if(!playing||!touchPointers.has(event.pointerId))return;
      event.preventDefault();
      if(moveKey){
        const el=document.elementFromPoint(event.clientX,event.clientY);
        const target=el?.closest?.('[data-move]');
        if(target&&target!==button&&target.dataset.move&&target.dataset.move!==value){
          release(event);
          try{target.dispatchEvent(new PointerEvent('pointerdown',event));}catch(_){}
        }
      }
    });
    ['pointerup','pointercancel','lostpointercapture'].forEach(name=>button.addEventListener(name,release));
  }
  const touchContainer=root.querySelector('.rift-touch');
  if(touchContainer){
    touchContainer.addEventListener('touchmove',e=>{e.preventDefault();},{passive:false});
    makeDraggable(touchContainer,'touch');
  }
  const joystickCluster=$('rift-touch-joystick');
  if(joystickCluster)makeDraggable(joystickCluster,'touch');
  const actionBarEl=$('rift-actionbar');
  if(actionBarEl)makeDraggable(actionBarEl,'actions');
  root.querySelectorAll('[data-hold]').forEach(button=>hold(button,button.dataset.hold));root.querySelectorAll('[data-move]').forEach(button=>hold(button,button.dataset.move));
  const joystickBase=$('rift-joystick-base'),joystickStick=$('rift-joystick-stick');
  let joystickPointerId=null;
  if(joystickBase&&joystickStick){
    const maxRadius=36,deadZone=6;
    function updateJoystick(clientX,clientY){
      const rect=joystickBase.getBoundingClientRect();
      const centerX=rect.left+rect.width/2,centerY=rect.top+rect.height/2;
      const dx=clientX-centerX,dy=clientY-centerY;
      const dist=Math.hypot(dx,dy);
      if(dist<deadZone){
        touchJoystick.x=0;touchJoystick.y=0;
        joystickStick.style.transform='translate(0px, 0px)';
        return;
      }
      const clampedDist=Math.min(dist,maxRadius);
      const angle=Math.atan2(dy,dx);
      const stickX=Math.cos(angle)*clampedDist,stickY=Math.sin(angle)*clampedDist;
      joystickStick.style.transform='translate('+stickX.toFixed(1)+'px, '+stickY.toFixed(1)+'px)';
      touchJoystick.x=Number(Math.max(-1,Math.min(1,stickX/maxRadius)).toFixed(2));
      touchJoystick.y=Number(Math.max(-1,Math.min(1,stickY/maxRadius)).toFixed(2));
      markInput(touchJoystick.x>0?'right':touchJoystick.x<0?'left':touchJoystick.y>0?'down':'up');
    }
    function releaseJoystick(event){
      if(joystickPointerId!==null&&event.pointerId===joystickPointerId){
        if(joystickBase.hasPointerCapture(event.pointerId)){
          try{joystickBase.releasePointerCapture(event.pointerId);}catch(_){}
        }
        joystickPointerId=null;touchJoystick.x=0;touchJoystick.y=0;
        joystickBase.classList.remove('rift-active');
        joystickStick.style.transform='translate(0px, 0px)';
      }
    }
    joystickBase.addEventListener('pointerdown',event=>{
      if(touchEditing||!playing||event.button>0)return;
      event.preventDefault();joystickPointerId=event.pointerId;
      try{joystickBase.setPointerCapture(event.pointerId);}catch(_){}
      joystickBase.classList.add('rift-active');
      updateJoystick(event.clientX,event.clientY);
    });
    joystickBase.addEventListener('pointermove',event=>{
      if(joystickPointerId===event.pointerId){
        event.preventDefault();updateJoystick(event.clientX,event.clientY);
      }
    });
    ['pointerup','pointercancel','lostpointercapture'].forEach(name=>joystickBase.addEventListener(name,releaseJoystick));
    joystickBase.addEventListener('touchmove',e=>{e.preventDefault();},{passive:false});
  }
  $('rift-controls-open').addEventListener('click',async()=>{startIntent++;if(playing)await pause();resetInput();if(!controls.opened)controls.open($('rift-controls-open'));});
  const openRefBtn=$('rift-open-controls-reference');
  if(openRefBtn){openRefBtn.addEventListener('click',async()=>{startIntent++;if(playing)await pause();resetInput();if(!controls.opened)controls.open(openRefBtn,true);});}
  const openLegendBtn=$('rift-open-legend-reference');
  if(openLegendBtn){
    openLegendBtn.addEventListener('click',()=>{
      const fieldGuide=$('rift-field-guide');
      if(fieldGuide){
        fieldGuide.open=true;
        const legendSection=$('rift-legend-guide');
        if(legendSection){
          legendSection.scrollIntoView({behavior:'smooth',block:'start'});
          const title=legendSection.querySelector('h3');
          if(title){title.setAttribute('tabindex','-1');title.focus();}
        }
      }
    });
  }
  const openAccessBtn=$('rift-open-accessibility-reference');
  if(openAccessBtn){
    openAccessBtn.addEventListener('click',()=>{
      const fieldGuide=$('rift-field-guide');
      if(fieldGuide){
        fieldGuide.open=true;
        const accessSection=$('rift-accessibility-guide');
        if(accessSection){
          accessSection.scrollIntoView({behavior:'smooth',block:'start'});
          const title=accessSection.querySelector('h3');
          if(title){title.setAttribute('tabindex','-1');title.focus();}
        }
      }
    });
  }
  window.addEventListener('riftbindingschange',()=>{resetInput();classPrimer();if(run)update(run,true);});
  window.addEventListener('riftintentchange',resetInput);
  // Rotation moves touch targets beneath held fingers. Require a fresh press,
  // but don't clear input for ordinary toolbar/keyboard height-only resizing.
  function resetRotatedInput(){if(touchPointers.size||window.matchMedia('(any-pointer: coarse)').matches)resetInput();}
  window.screen.orientation?.addEventListener('change',resetRotatedInput);
  window.addEventListener('orientationchange',resetRotatedInput);
  window.matchMedia('(orientation: portrait)').addEventListener('change',resetRotatedInput);

  $('rift-canvas').addEventListener('pointerdown',event=>{canvasMouse=event.pointerType==='mouse';if(playing&&canvasMouse&&controls.pointer(event.button))$('rift-canvas').setPointerCapture(event.pointerId);});
  $('rift-canvas').addEventListener('mousedown',event=>{
    const action=controls.pointer(event.button);if(!playing||controls.opened||!canvasMouse||!action||event.ctrlKey||event.metaKey||event.altKey)return;
    event.preventDefault();$('rift-canvas').focus();markInput(action);window.RiftIntents.recognize(action);
    if(action==='guard'&&controls.toggleGuard)toggleGuard();else{mouse.add(action);taps.add(action);}
  });
  window.addEventListener('mouseup',event=>{const action=controls.pointer(event.button);if(action)mouse.delete(action);});
  $('rift-canvas').addEventListener('lostpointercapture',()=>{for(const action of mouse)taps.delete(action);mouse.clear();});
  $('rift-canvas').addEventListener('contextmenu',event=>{if(playing&&controls.pointer(2))event.preventDefault();});
  window.addEventListener('keydown',event=>{
    if((event.code==='F4'||(event.code==='KeyH'&&event.altKey&&event.shiftKey))&&!event.repeat&&!event.isComposing&&!controls.opened&&!event.target.closest('input,select,textarea,[contenteditable="true"]')){event.preventDefault();const active=window.RiftDisplay?.toggleScreenshot?.();status(active?statusCopy.screenshotEnabled:statusCopy.screenshotDisabled);return;}
    if(event.code==='KeyR'&&event.altKey&&event.shiftKey&&!event.repeat&&!event.isComposing&&!controls.opened&&!event.target.closest('input,select,textarea,[contenteditable="true"]')){event.preventDefault();cycleRange();return;}
    if(event.shiftKey&&!event.ctrlKey&&!event.metaKey&&!event.altKey&&['Digit1','Digit2','Digit3'].includes(event.code)&&!event.repeat&&!event.isComposing&&!controls.opened&&!event.target.closest('input,select,textarea,[contenteditable="true"]')){
      const idx=Number(event.code.replace('Digit',''))-1,s=run?.build?.skills?.[idx];
      if(s){event.preventDefault();pinnedRangeIndex=idx;pinnedRangeSkill=s;window.RiftHUD.setRequestedRange(s);status(statusCopy.showRange(s.name));return;}
    }
    if(event.code==='KeyL'&&event.altKey&&event.shiftKey&&!event.ctrlKey&&!event.metaKey&&!event.repeat&&!event.isComposing&&!controls.opened&&!event.target.closest('input,select,textarea,[contenteditable="true"]')){event.preventDefault();openLoadoutReference();return;}
    if(controls.opened||event.ctrlKey||event.metaKey||event.altKey||event.isComposing)return;
    const action=controls.action(event.code);
    if(event.code==='Escape'&&!event.repeat){if(playing)pause();else if(run&&['fighting','cleared'].includes(run.status))begin();return;}
    if(event.target.matches('input,select,textarea'))return;
    if(['Space','Enter'].includes(event.code)&&event.target.closest('button,summary,a'))return;
    if(action==='pause'&&!event.repeat){event.preventDefault();if(playing)pause();else if(run&&['fighting','cleared'].includes(run.status))begin();return;}
    if(!playing||!action)return;event.preventDefault();keys.add(event.code);if(!event.repeat){markInput(action);keyOrder.set(event.code,++keySequence);window.RiftIntents.press(action);if(action==='guard'&&controls.toggleGuard)toggleGuard();else taps.add(event.code);}
  });
  window.addEventListener('keyup',event=>keys.delete(event.code));window.addEventListener('blur',()=>{startIntent++;resetInput();if(playing)pause();});
  document.addEventListener('visibilitychange',()=>{if(document.hidden){startIntent++;resetInput();if(playing)pause();silence();}});
  window.addEventListener('pagehide',()=>{loadGeneration++;pendingRead?.abort();});
  window.addEventListener('pageshow',event=>{
    const isHistory = event.persisted || (typeof performance !== 'undefined' && performance.getEntriesByType?.('navigation')?.[0]?.type === 'back_forward');
    if(isHistory){startIntent++;playing=false;clearTimeout(timer);resetInput();silence();load();}
  });
  document.addEventListener('focusin',event=>{
    if(playing&&event.target&&['INPUT','TEXTAREA'].includes(event.target.tagName)){
      const type=event.target.type;
      if(!['checkbox','radio','range','button','submit'].includes(type)){
        event.target.blur();
        $('rift-canvas')?.focus({preventScroll:true});
      }
    }
  },true);
  let pinnedRangeSkill=null,pinnedRangeIndex=-1;

  function cycleRange(){
    const skills=[...(run?.build?.skills||[]),...(run?.build?.signatures||[]),...(run?.build?.ultimate?[run.build.ultimate]:[])];if(!skills.length)return;
    pinnedRangeIndex=(pinnedRangeIndex+1)%(skills.length+1);
    if(pinnedRangeIndex===skills.length){pinnedRangeIndex=-1;pinnedRangeSkill=null;window.RiftHUD.setRequestedRange(null);status(statusCopy.rangeHidden);}
    else{pinnedRangeSkill=skills[pinnedRangeIndex];window.RiftHUD.setRequestedRange(pinnedRangeSkill);status(statusCopy.showRange(pinnedRangeSkill.name));}
  }
  const rangeToggle=$('rift-range-toggle');
  if(rangeToggle)rangeToggle.addEventListener('click',cycleRange);
  $('rift-start').addEventListener('click',begin);$('rift-pause').addEventListener('click',()=>playing?pause():begin());
  async function openLoadoutReference(){
    if(!ready||starting||controls.opened)return;
    if(playing)await pause();
    if(busy||controls.opened||(run&&['fighting','cleared'].includes(run.status)&&!run.paused))return;
    resetInput();window.RiftLoadouts.openReference();
  }
  const skillDescriptions=document.createElement('button');skillDescriptions.type='button';skillDescriptions.id='rift-skill-descriptions';skillDescriptions.textContent='Skill descriptions';skillDescriptions.setAttribute('aria-controls','rift-skill-glossary');skillDescriptions.addEventListener('click',openLoadoutReference);$('rift-actionbar').after(skillDescriptions);
  const loadoutPreview=document.createElement('button');loadoutPreview.type='button';loadoutPreview.id='rift-loadout-preview';loadoutPreview.textContent='Skill reference · Alt+Shift+L';loadoutPreview.setAttribute('aria-keyshortcuts','Alt+Shift+L');loadoutPreview.addEventListener('click',openLoadoutReference);$('rift-loadout-order').after(loadoutPreview);
  $('rift-retry-boss').addEventListener('click',async()=>{
 if(!ready||busy||starting||practice||run?.status!=='defeated')return;
 if(await send('retry_boss')){message('Boss room ready.','Health, mana and cooldowns restored. Banked rewards are safe. Failed attempts still count toward mission time.','Resume boss fight','RETRY');$('rift-start').focus();}
 });
  $('rift-replay').addEventListener('click',()=>{
    if(!ready||busy||starting||!run?.level||!['complete','banked'].includes(run.status)||run.room!==2)return;
    selectedLevel=run.level.id;campaignKey='';updateCampaign();begin();
  });
  $('rift-result-region').addEventListener('click',()=>{
    if(busy||starting||!run?.level||!['complete','banked','defeated'].includes(run.status))return;
    window.RiftCampaignTools.showRegion(run.level.region);
  });
  $('rift-result-skills').addEventListener('click',()=>{if(busy||starting||!run||!['complete','banked','defeated'].includes(run.status))return;const details=root.querySelector('.rift-run-statistics');details.open=true;details.querySelector('summary').focus();details.scrollIntoView({block:'start'});});
  const resultEncounterBtn=$('rift-result-encounter');
  if(resultEncounterBtn){
    resultEncounterBtn.addEventListener('click',()=>{
      if(busy||starting||!run||!['complete','banked','defeated','cleared'].includes(run.status))return;
      const enc=$('rift-last-encounter');
      if(enc){
        enc.scrollIntoView({behavior:'smooth',block:'start'});
        enc.focus();
      }
    });
  }
  const retryMissionBtn=$('rift-retry-mission');
  if(retryMissionBtn){
    retryMissionBtn.addEventListener('click',async()=>{
      if(busy||starting||!ready)return;
      if(run?.level?.id)selectedLevel=run.level.id;
      await begin();
    });
  }
  const resultCampaignBtn=$('rift-result-campaign');
  if(resultCampaignBtn){
    resultCampaignBtn.addEventListener('click',()=>{
      const campaign=$('rift-campaign');
      if(campaign){
        campaign.open=true;
        campaign.scrollIntoView({behavior:'smooth',block:'start'});
        campaign.querySelector('summary')?.focus();
      }
    });
  }
  const resultBuildBtn=$('rift-result-build');
  if(resultBuildBtn){
    resultBuildBtn.addEventListener('click',()=>{
      const buildSection=root.querySelector('.rift-class-primer')||$('rift-loadout');
      if(buildSection){
        buildSection.scrollIntoView({behavior:'smooth',block:'start'});
        root.querySelector('#rift-loadout select')?.focus();
      }
    });
  }
  const practiceBossLink=$('rift-practice-boss-link');
  if(practiceBossLink){
    practiceBossLink.addEventListener('click',()=>{
      const bossName=run?.defeated_by_boss||run?.encounter_plan?.[run?.room]?.find(enemy=>enemy.kind==='boss')?.name||'';
      window.location.href='/abyss/rift?practice=boss'+(bossName?'&boss='+encodeURIComponent(bossName):'');
    });
  }
  const copyResultBtn=$('rift-copy-result-summary');
  if(copyResultBtn){
    copyResultBtn.addEventListener('click',async()=>{
      if(!run)return;
      const text=window.RiftHUD?.buildResultSummary?window.RiftHUD.buildResultSummary(run):'';
      if(!text)return;
      const statusEl=$('rift-result-copy-status');
      try{
        await navigator.clipboard.writeText(text);
        if(statusEl){
          statusEl.textContent='Summary copied.';
          statusEl.hidden=false;
          setTimeout(()=>{statusEl.hidden=true;},3000);
        }
      }catch(_){
        if(statusEl){
          statusEl.textContent='Copy unavailable.';
          statusEl.hidden=false;
        }
      }
    });
  }
  const printResultBtn=$('rift-print-result');
  if(printResultBtn){
    printResultBtn.addEventListener('click',()=>{
      window.print();
    });
  }
  $('rift-next').addEventListener('click',async()=>{if(await checkpoint($('rift-auto').checked?'advance':'next'))status(run.status==='complete'?statusCopy.expeditionComplete:statusCopy.checkpointReached);});
  $('rift-auto').addEventListener('change',()=>{try{localStorage.setItem('rift-auto',String($('rift-auto').checked));}catch(_){}clearedAt=0;countdownAnnounced=-1;if(run)update(run,true);});
  $('rift-exit').addEventListener('click',()=>checkpoint('exit'));
  async function toggleFullscreen(){
    try{
      if(document.fullscreenElement)await document.exitFullscreen();
      else await $('rift-viewport').requestFullscreen();
    }catch(_){
      status(statusCopy.fullscreenUnavailable);
    }
  }
  $('rift-screenshot-toggle')?.addEventListener('click',()=>{
    const active=window.RiftDisplay?.toggleScreenshot?.();
    status(active?statusCopy.screenshotEnabled:statusCopy.screenshotDisabled);
  });
  $('rift-fullscreen').addEventListener('click',async()=>{
    if(document.fullscreenElement){
      try{await document.exitFullscreen();}catch(_){}
      return;
    }
    if($('rift-fullscreen-controls').checked){
      startIntent++;
      if(playing)await pause();
      resetInput();
      if(!controls.opened)controls.open();
      status(statusCopy.fullscreenReview);
      return;
    }
    await toggleFullscreen();
  });
  $('rift-controls-fullscreen').addEventListener('click',async()=>{
    if(controls.opened)$('rift-controls-dialog').close();
    await toggleFullscreen();
  });
  $('rift-fullscreen-exit').addEventListener('click',async()=>{if(!document.fullscreenElement)return;await toggleFullscreen();(root.classList.contains('rift-clean-screenshot')?$('rift-canvas'):$('rift-fullscreen')).focus();});
  document.addEventListener('fullscreenchange',()=>{
    const inFs=!!document.fullscreenElement;
    $('rift-fullscreen').setAttribute('aria-pressed',String(inFs));
    $('rift-controls-fullscreen').textContent=inFs?'Exit fullscreen':'Enter fullscreen';
  });
  function soundLabel(){
    const blocked=!audio.muted&&Boolean(audio.isBlocked?.());
    for(const id of ['rift-sound','rift-fullscreen-sound']){
      const button=$(id);
      button.textContent=audio.muted?'Sound off':blocked?'Sound blocked':'Sound on';
      button.setAttribute('aria-pressed',String(audio.muted));
      if(blocked){button.dataset.audioBlocked='true';button.title='Audio is blocked by the browser. Click to allow sound.';}
      else{delete button.dataset.audioBlocked;button.title='';}
    }
  }
  soundLabel();
  for(const id of ['rift-sound','rift-fullscreen-sound'])$(id).addEventListener('click',async()=>{await audio.unlock();audio.set('muted',!audio.muted);soundLabel();audio.play('ui',0);});
  window.addEventListener('riftaudiochange',soundLabel);
  [['effects','rift-effects-volume'],['ambience','rift-ambience-volume']].forEach(([key,id])=>{const elem=$(id);elem.value=audio[key]*100;elem.addEventListener('input',()=>audio.set(key,Number(elem.value)/100,true));elem.addEventListener('change',()=>audio.set(key,Number(elem.value)/100,false));elem.addEventListener('pointerup',()=>audio.set(key,Number(elem.value)/100,false));});
  const systemMotion=window.matchMedia('(prefers-reduced-motion: reduce)');let reducedOverride=null;
  try{const reduced=JSON.parse(localStorage.getItem('riftReducedMotion'));if(typeof reduced==='boolean')reducedOverride=reduced;}catch(_){}
  function motionPreference(){renderer.reduced=reducedOverride??systemMotion.matches;$('rift-reduced').checked=renderer.reduced;window.dispatchEvent(new Event('riftmotionchange'));}
  $('rift-reduced').addEventListener('change',()=>{reducedOverride=$('rift-reduced').checked;try{localStorage.setItem('riftReducedMotion',JSON.stringify(reducedOverride));}catch(_){}motionPreference();});
  $('rift-system-motion').addEventListener('click',()=>{reducedOverride=null;try{localStorage.removeItem('riftReducedMotion');}catch(_){}motionPreference();});
  systemMotion.addEventListener('change',()=>{if(reducedOverride===null)motionPreference();});motionPreference();
  const abbreviateToggle=$('rift-abbreviate-abilities');
  if(abbreviateToggle){
    abbreviateToggle.checked=abbreviateAbilities;
    abbreviateToggle.addEventListener('change',()=>{
      abbreviateAbilities=abbreviateToggle.checked;
      try{localStorage.setItem('riftAbbreviateAbilities',String(abbreviateAbilities));}catch(_){}
      updateAbilityLabels();
    });
  }
  function updateVisualViewportMetrics(){
    const vp=window.visualViewport;
    const h=vp?vp.height:window.innerHeight;
    root.style.setProperty('--rift-visual-height',`${h}px`);
  }
  updateVisualViewportMetrics();
  if(window.visualViewport){
    window.visualViewport.addEventListener('resize',updateVisualViewportMetrics);
  }else{
    window.addEventListener('resize',updateVisualViewportMetrics);
  }
  window.RiftGamepad.init({playing:()=>playing,skillCount:()=>run?.build.skills.length||0,recognize:action=>{markInput(action);window.RiftIntents.recognize(action);},ability:action=>{markInput(action);window.RiftIntents.press(action);},guard:()=>{if(controls.toggleGuard)toggleGuard();},togglePause:()=>playing?pause():begin(),disconnect:()=>{startIntent++;resetInput();if(playing)pause();},});
  const settings=root.querySelector('.rift-settings'),returnToBattlefield=document.createElement('button');returnToBattlefield.id='rift-settings-return';returnToBattlefield.type='button';returnToBattlefield.textContent='Return to battlefield';returnToBattlefield.disabled=true;settings.append(returnToBattlefield);
  function focusBattlefield(){if(!playing||controls.opened)return;resetInput();$('rift-canvas').focus();}
  returnToBattlefield.addEventListener('click',()=>{if(!playing||controls.opened)return;settings.open=false;focusBattlefield();});
  settings.addEventListener('toggle',()=>{if(!settings.open)focusBattlefield();});
  const skipControls=$('rift-skip-controls');
  if(skipControls){
    skipControls.addEventListener('click',event=>{
      event.preventDefault();
      const bar=$('rift-actionbar');
      if(bar){
        bar.scrollIntoView({block:'nearest'});
        const btn=bar.querySelector('button:not(:disabled):not([aria-disabled="true"])')||bar.querySelector('button:not(:disabled)');
        if(btn)btn.focus();
        else bar.focus();
      }
    });
  }
  const skipMission=$('rift-skip-mission');
  if(skipMission){
    skipMission.addEventListener('click',event=>{
      event.preventDefault();
      const campaign=$('rift-campaign');
      if(campaign){
        if(!campaign.open)campaign.open=true;
        campaign.scrollIntoView({block:'nearest'});
        const selected=$('rift-levels')?.querySelector('button[aria-pressed="true"]:not([hidden]), button[data-level]:not([hidden])')||campaign.querySelector('summary');
        if(selected)selected.focus();
        else campaign.querySelector('summary').focus();
      }
    });
  }
  if(practice){
    root.querySelector('.rift-tag').textContent='PRACTICE · '+drillNames[practice].toUpperCase();
    document.title='Practice · '+drillNames[practice]+' · Abyss Brawl';
    const practiceNote=document.createElement('p');practiceNote.id='rift-controls-practice-note';practiceNote.textContent='Practice only · No loot or campaign progress. These controls apply to your isolated drill.';$('rift-controls-title').after(practiceNote);
    $('rift-practice-guide').hidden=false;$('rift-practice-title').textContent=drillNames[practice];
    $('rift-practice-bank-controls').hidden=practice!=='banking';
    $('rift-practice-enemy-controls').hidden=practice!=='skills';
    $('rift-boss-practice-options').hidden=practice!=='boss';
    $('rift-hazard-practice-options').hidden=practice!=='hazard';
    refreshPracticeInstructions();
    for(const node of [$('rift-campaign'),$('rift-campaign-tools-extra'),root.querySelector('.rift-route')?.closest('section'),$('rift-loot')?.closest('section'),root.querySelector('.rift-run-statistics'),$('rift-walkthrough')])if(node)node.hidden=true;
    root.querySelectorAll('[data-practice-action]').forEach(button=>button.addEventListener('click',async()=>{
      if(!ready||starting||practiceToolPending||run?.status!=='fighting'||button.getAttribute('aria-disabled')==='true')return;
      practiceToolPending=true;practiceToolButtons();setSafeDisabled($('rift-practice-reset'),true);
      try{await pause();while(busy)await new Promise(resolve=>setTimeout(resolve,20));if(run?.status==='fighting'&&!run.paused&&!await send('pause'))return;if(run?.status==='fighting'&&run.paused&&await send(button.dataset.practiceAction)){
        $('rift-practice-tool-status').textContent=button.dataset.practiceAction==='practice_bank'?(run.practice.completed?'Practice receipt confirmed. Three tokens secured; no real rewards were granted.':'Practice receipt is not complete. Review the saved checkpoint state.'):['practice_spawn','practice_clear'].includes(button.dataset.practiceAction)?'Saved practice arena: '+(run.enemies.length?run.enemies.map(enemy=>enemy.name).join(', '):'empty')+'. Practice is paused.':button.dataset.practiceAction==='practice_freeze'?(run.practice.freeze_movement?'Enemy movement frozen. Attacks and projectiles continue.':'Enemy movement resumed. Reset to start a new run without freeze assistance.'):button.textContent+' applied. Drill progress is unchanged.';
        if(!run.practice.completed)message(drillNames[practice],$('rift-practice-instructions').textContent,'Resume drill','PRACTICE PAUSED');
      }}finally{practiceToolPending=false;practiceToolButtons();setSafeDisabled($('rift-practice-reset'),!ready||busy||!run);}
    }));
    $('rift-practice-reset').addEventListener('click',async()=>{if(!ready||starting||practiceToolPending||!run||$('rift-practice-reset').getAttribute('aria-disabled')==='true')return;practiceToolPending=true;practiceToolButtons();setSafeDisabled($('rift-practice-reset'),true);try{await pause();while(busy)await new Promise(resolve=>setTimeout(resolve,20));if(await send('practice_reset')){message(drillNames[practice],$('rift-practice-instructions').textContent,'Start drill','PRACTICE');$('rift-canvas').focus();}}finally{practiceToolPending=false;practiceToolButtons();setSafeDisabled($('rift-practice-reset'),!ready||!run);}});
  }
  load();
})();
