/* Original procedural arcade sound design. No downloads or third-party samples. */
(function () {
  'use strict';
  function setting(key, fallback) { try { const value = JSON.parse(localStorage.getItem('riftAudio:' + key)); return value === null ? fallback : value; } catch (_) { return fallback; } }
  function levelSetting(key,fallback){const value=setting(key,fallback);return typeof value==='number'&&Number.isFinite(value)?Math.max(0,Math.min(1,value)):fallback;}
  const audio = { context: null, muted: setting('muted', false)===true, effects: levelSetting('effects', .65), ambience: levelSetting('ambience', .35), played: 0, voices: 0 };
  audio.music=levelSetting('music',audio.ambience);audio.voice=levelSetting('voice',audio.effects);audio.interface=levelSetting('interface',audio.effects);audio.mono=setting('mono',false)===true;
  audio.steadyAmbience=setting('steadyAmbience',false)===true;
  audio.interfaceMuted=setting('interfaceMuted',false)===true;
  const channelLevel=key=>key==='interface'&&audio.interfaceMuted?0:audio[key];
  let master, sfx, ambient, music, voice, interfaceBus, noise, ambientNodes = [], active = false, room = -1, nextBird = 0, activation = 0, previewIntent = 0, previewTimer = 0;
  const panners=new Map(),sources=new Map();let previewRequested=false;
  const buses=()=>({effects:sfx,ambience:ambient,music,voice,interface:interfaceBus});
  const clamp = value => Math.max(0, Math.min(1, Number(value) || 0));
  function save(key, value) { try { localStorage.setItem('riftAudio:' + key, JSON.stringify(value)); } catch (_) {} }
  // Freeze migrated values once so later parent-channel edits stay independent.
  for(const key of ['music','voice','interface'])save(key,audio[key]);
  function busGain(bus, value) { if (bus) bus.gain.setTargetAtTime(value, audio.context.currentTime, .03); }
  audio.unlock = async function () {
    try {
      if(audio.context?.state==='closed'){stopAmbience();stopVoices();audio.context=null;audio.voices=0;panners.clear();}
      if (!audio.context) {
        const AC = window.AudioContext || window.webkitAudioContext;
        if (!AC) return false;
        const c = audio.context = new AC();
        master = c.createGain(); sfx = c.createGain(); ambient = c.createGain(); music=c.createGain(); voice=c.createGain(); interfaceBus=c.createGain();
        const limiter = c.createDynamicsCompressor(); limiter.threshold.value = -16; limiter.ratio.value = 8;
        Object.values(buses()).forEach(bus=>bus.connect(master)); master.connect(limiter); limiter.connect(c.destination);
        master.gain.value = audio.muted ? 0 : .6; for(const [key,bus] of Object.entries(buses()))bus.gain.value=channelLevel(key);
        noise = c.createBuffer(1, c.sampleRate * 2, c.sampleRate);
        const data = noise.getChannelData(0);
        for (let i = 0; i < data.length; i++) data[i] = Math.random() * 2 - 1;
      }
      if (audio.context.state !== 'running') await audio.context.resume();
      return audio.context.state === 'running';
    } catch (_) { return false; }
  };
  function tone(frequency, endFrequency, duration, volume, type, delay, pan, bus) {
    const c = audio.context;
    if (!c || c.state !== 'running' || audio.voices >= 40) return;
    const start = c.currentTime + (delay || 0), gain = c.createGain(), source = c.createOscillator(), stereo = c.createStereoPanner();
    source.type = type || 'triangle'; source.frequency.setValueAtTime(Math.max(20, frequency), start);
    source.frequency.exponentialRampToValueAtTime(Math.max(20, endFrequency), start + duration);
    gain.gain.setValueAtTime(0, start); gain.gain.linearRampToValueAtTime(volume, start + .008); gain.gain.exponentialRampToValueAtTime(.0001, start + duration);
    const position=Math.max(-.8,Math.min(.8,pan||0));panners.set(stereo,position);stereo.pan.value=audio.mono?0:position;
    source.connect(gain); gain.connect(stereo); stereo.connect(bus || sfx); audio.voices++;
    source.onended = () => { if(!sources.delete(source))return;source.disconnect(); gain.disconnect(); stereo.disconnect();panners.delete(stereo); if(audio.context===c)audio.voices=Math.max(0,audio.voices-1); };
    sources.set(source,source.onended);source.start(start); source.stop(start + duration + .02);
  }
  function hiss(duration, volume, cutoff, pan, delay, bus) {
    const c = audio.context;
    if (!c || c.state !== 'running' || audio.voices >= 40) return;
    const start = c.currentTime + (delay || 0), source = c.createBufferSource(), gain = c.createGain(), filter = c.createBiquadFilter(), stereo = c.createStereoPanner();
    source.buffer = noise; filter.type = 'lowpass'; filter.frequency.value = cutoff;
    gain.gain.setValueAtTime(volume, start); gain.gain.exponentialRampToValueAtTime(.0001, start + duration);
    const position=Math.max(-.8,Math.min(.8,pan||0));panners.set(stereo,position);stereo.pan.value=audio.mono?0:position; source.connect(filter); filter.connect(gain); gain.connect(stereo); stereo.connect(bus||sfx); audio.voices++;
    source.onended = () => { if(!sources.delete(source))return;source.disconnect(); filter.disconnect(); gain.disconnect(); stereo.disconnect();panners.delete(stereo); if(audio.context===c)audio.voices=Math.max(0,audio.voices-1); };
    sources.set(source,source.onended);source.start(start); source.stop(start + duration + .02);
  }
  let lastEmptyMana = 0, lastCooldownRejection = 0;
  audio.playEmptyMana = function (pan = 0) {
    const now = performance.now();
    if (now - lastEmptyMana < 500) return false;
    lastEmptyMana = now;
    audio.play('empty_mana', pan);
    return true;
  };
  audio.playCooldownRejection = function (pan = 0) {
    const now = performance.now();
    if (now - lastCooldownRejection < 500) return false;
    lastCooldownRejection = now;
    audio.play('cooldown_rejection', pan);
    return true;
  };
  audio.play = function (kind, pan, extra) {
    if (!audio.context || audio.context.state !== 'running' || audio.muted || !active) return;
    if (kind === 'step' && extra) { audio.step(extra, pan); return; }
    playCue(kind,pan);
  };
  audio.step = function (material, pan) {
    if (!audio.context || audio.context.state !== 'running' || audio.muted || !active) return;
    const mat = String(material || 'stone').toLowerCase();
    const cue = ['metal','wood','water','mud','ice','grass','dirt','stone'].includes(mat) ? 'step_' + mat : 'step';
    playCue(cue, pan || 0);
  };
  function playCue(kind,pan){
    audio.played++;
    const target=kind==='ui'||kind==='bank'||kind==='empty_mana'||kind==='cooldown_rejection'?interfaceBus:/(?:_attack|_death|_roar|_escape)$/.test(kind)?voice:sfx;
    const t = (f, end, d, v, wave, delay) => tone(f, end, d, v, wave, delay, pan,target);
    const h = (duration,volume,cutoff,position,delay)=>hiss(duration,volume,cutoff,position,delay,target);
    switch (kind) {
      case 'step': case 'step_stone': h(.065, .07, 650, pan); t(120, 60, .05, .04, 'triangle'); break;
      case 'step_metal': h(.07, .06, 2800, pan); t(620, 480, .06, .05, 'triangle'); break;
      case 'step_wood': h(.06, .07, 450, pan); t(160, 90, .07, .07, 'triangle'); break;
      case 'step_water': h(.11, .08, 1400, pan); t(260, 110, .08, .04, 'sine'); break;
      case 'step_mud': h(.1, .075, 950, pan); t(180, 80, .07, .045, 'triangle'); break;
      case 'step_ice': h(.08, .075, 4500, pan); t(880, 720, .05, .03, 'sine'); break;
      case 'step_grass': case 'step_dirt': h(.07, .06, 380, pan); t(90, 50, .06, .035, 'triangle'); break;
      case 'jump': t(160, 480, .15, .09, 'triangle'); h(.09, .04, 1200, pan); break;
      case 'slash': h(.14, .15, 3800, pan); t(350, 100, .12, .05, 'sawtooth'); break;
      case 'hit': h(.1, .2, 1700, pan); t(110, 48, .15, .18, 'triangle'); break;
      case 'hurt': t(170, 65, .2, .1, 'sawtooth'); h(.12, .12, 950, pan); break;
      case 'block': t(940, 760, .2, .08, 'square'); t(1510, 1200, .12, .04, 'sine'); h(.04, .09, 7000, pan); break;
      case 'fire': h(.5, .18, 1500, pan); t(200, 45, .45, .1, 'sawtooth'); break;
      case 'ice': [880,1320,1760].forEach((f,i) => t(f, f*.8, .25, .045, 'sine', i*.045)); h(.1,.055,7000,pan); break;
      case 'void': t(240, 38, .6, .13, 'sawtooth'); t(243, 42, .55, .07, 'sine'); break;
      case 'void_cost': t(80,40,.18,.08,'triangle'); break;
      case 'rune': [294,440,587].forEach((f,i)=>t(f,f*1.5,.3,.055,'triangle',i*.06)); h(.18,.06,4000,pan); break;
      case 'poison': [180,270,140].forEach((f,i)=>t(f,f*.5,.14,.08,'sine',i*.1)); h(.4,.08,750,pan); break;
      case 'radiant': [523,784,1047].forEach((f,i)=>t(f,f,.55,.05,'sine',i*.06)); break;
      case 'pack': t(320,620,.35,.075,'sine'); t(620,300,.6,.06,'sine',.3); h(.18,.05,850,pan); break;
      case 'quake': h(.85,.22,600,pan); t(90,25,.7,.2,'triangle'); break;
      case 'ultimate': [196,294,392,588].forEach((f,i)=>t(f,f*2,.75,.07,'sawtooth',i*.07)); h(.8,.2,2200,pan); break;
      case 'knockdown': h(.2,.16,500,pan); t(85,32,.2,.12,'triangle'); break;
      case 'shield': case 'heal': [330,440,660].forEach((f,i) => t(f,f*1.05,.45,.06,'sine',i*.07)); break;
      case 'arrow': h(.12,.1,6000,pan); t(700,300,.08,.025,'triangle'); break;
      case 'goblin_attack': t(390,190,.19,.06,'sawtooth'); h(.08,.08,1600,pan); break;
      case 'knight_attack': h(.25,.2,900,pan); t(140,55,.22,.16,'triangle'); break;
      case 'goblin_death': t(440,80,.4,.075,'sawtooth'); break;
      case 'treasure_attack': t(560,240,.13,.07,'triangle'); break;
      case 'treasure_death': [660,990,1320].forEach((f,i)=>t(f,f*1.2,.23,.06,'triangle',i*.1)); break;
      case 'treasure_escape': [440,660,880,1320,1760].forEach((f,i)=>t(f,f*1.25,.18,.055,'triangle',i*.05)); h(.28,.08,6500,pan); t(240,60,.35,.07,'sine'); break;
      case 'archer_death': h(.4,.16,4000,pan); t(600,180,.28,.04,'square'); break;
      case 'knight_death': h(.7,.24,1000,pan); t(100,30,.6,.18,'triangle'); break;
      case 'boss_roar': t(85,45,.9,.18,'sawtooth'); t(88,41,.8,.08,'sawtooth'); h(.7,.15,700,pan); break;
      case 'slam': h(.65,.3,1100,pan); t(100,28,.5,.3,'sine'); t(140,40,.4,.08,'triangle',.08); break;
      case 'boss_death': h(1.5,.25,950,pan); [110,82,55].forEach((f,i) => t(f,28,.7,.1,'sawtooth',i*.2)); break;
      case 'pickup': [660,990,1320].forEach((f,i) => t(f,f,.15,.045,'triangle',i*.04)); break;
      case 'rare_item': case 'rare_discovery': [523,659,784,1047,1318,1568].forEach((f,i)=>t(f,f*1.02,.38,.06,'sine',i*.055)); [1047,1318,1568,2093].forEach((f,i)=>t(f,f*.98,.45,.04,'triangle',.15+i*.04)); h(.35,.075,7500,pan); break;
      case 'clear': case 'bank': [392,494,587,784].forEach((f,i) => t(f,f,.4,.07,'triangle',i*.12)); break;
      case 'defeat': [294,247,196,147].forEach((f,i) => t(f,f*.95,.65,.07,'triangle',i*.16)); break;
      case 'area': t(110,165,.8,.05,'sine'); t(220,247,.9,.04,'sine',.1); break;
      case 'ui': t(540,720,.07,.035,'triangle'); break;
      case 'low_health': [0,.22].forEach(delay=>t(180,120,.16,.075,'triangle',delay)); break;
      case 'ultimate_ready': [523,784,1047].forEach((f,i)=>t(f,f,.22,.05,'sine',i*.09)); break;
      case 'finisher_ready': t(330,494,.16,.05,'triangle');t(494,660,.18,.05,'triangle',.14);break;
      case 'empty_mana': t(150,75,.15,.08,'triangle'); h(.08,.045,450,pan); break;
      case 'cooldown_rejection': t(480,240,.08,.055,'sine'); t(240,120,.09,.04,'triangle',.03); break;
      case 'dodge': t(280,560,.12,.06,'triangle'); h(.06,.035,3500,pan); break;
      case 'perfect_guard': t(1350,920,.1,.1,'triangle'); [1175,1760,2350].forEach((f,i)=>t(f,f*.96,.28,.06,'sine',i*.02)); h(.05,.08,8500,pan); break;
      case 'hazard_warning': [0,.14].forEach((d,i)=>t(520+i*160,680+i*160,.1,.07,'sawtooth',d)); h(.16,.05,3200,pan); break;
      case 'hazard_deactivation': t(580,220,.22,.06,'sine'); t(380,160,.18,.04,'triangle',.04); h(.18,.035,1200,pan); break;
      default: break;
    }
  };
  function stopVoices(){for(const [source,cleanup] of [...sources]){try{source.stop();}catch(_){}cleanup();}}
  let activeAmbience = null; const outgoingAmbience = new Set();
  let bossMusicNodes = null;
  audio.crossfading = false;
  audio.bossMusicActive = false;
  audio.bossCrossfading = false;
  audio.currentRegion = -1;
  function stopAmbience() {
    audio.stopBossMusic?.(0);
    if(activeAmbience){activeAmbience.sources.forEach(s=>{try{s.stop();}catch(_){}s.disconnect();});activeAmbience.gains.forEach(g=>{try{g.disconnect();}catch(_){}});}
    for(const group of outgoingAmbience){group.sources.forEach(s=>{try{s.stop();}catch(_){}s.disconnect();});group.gains.forEach(g=>{try{g.disconnect();}catch(_){}});}
    outgoingAmbience.clear();activeAmbience=null;ambientNodes=[];audio.crossfading=false;
  }
  audio.startBossMusic = function (customFade) {
    const c = audio.context;
    if (!c || !active || audio.bossMusicActive || c.state !== 'running') return false;
    const fade = typeof customFade === 'number' && customFade >= 0 ? customFade : 1.5;
    audio.bossMusicActive = true;
    audio.bossCrossfading = true;
    if (activeAmbience?.gains?.length) {
      activeAmbience.gains.slice(1).forEach(g => {
        try { g.gain.setValueAtTime(g.gain.value, c.currentTime); g.gain.linearRampToValueAtTime(0.005, c.currentTime + fade); } catch (_) {}
      });
    }
    const region = audio.currentRegion >= 0 ? audio.currentRegion : 0;
    const root = [130.81,73.42,146.83,82.41,98,65.41,87.31,110,61.74,55][region] || 130.81;
    const sources = [], gains = [];
    const bass = c.createOscillator(), bassFilter = c.createBiquadFilter(), bassGain = c.createGain();
    bass.type = 'sawtooth';
    bass.frequency.value = Math.max(30, root * 0.5);
    bassFilter.type = 'lowpass';
    bassFilter.frequency.value = 260;
    if (fade > 0) {
      bassGain.gain.setValueAtTime(0.0001, c.currentTime);
      bassGain.gain.linearRampToValueAtTime(0.024, c.currentTime + fade);
    } else {
      bassGain.gain.value = 0.024;
    }
    bass.connect(bassFilter); bassFilter.connect(bassGain); bassGain.connect(music);
    bass.start(); sources.push(bass); gains.push(bassGain);
    [root * 0.75, root * 1.2, root * 1.414, root * 1.8].forEach((f, i) => {
      const osc = c.createOscillator(), level = c.createGain();
      osc.type = i === 1 ? 'triangle' : 'sine';
      osc.frequency.value = f;
      const targetGain = [0.018, 0.016, 0.014, 0.012][i];
      if (fade > 0) {
        level.gain.setValueAtTime(0.0001, c.currentTime);
        level.gain.linearRampToValueAtTime(targetGain, c.currentTime + fade);
      } else {
        level.gain.value = targetGain;
      }
      osc.connect(level); level.connect(music);
      osc.start(); sources.push(osc); gains.push(level);
    });
    bossMusicNodes = { sources, gains };
    if (fade > 0) {
      setTimeout(() => {
        if (audio.bossMusicActive) audio.bossCrossfading = false;
      }, (fade + 0.05) * 1000);
    } else {
      audio.bossCrossfading = false;
    }
    return true;
  };
  audio.stopBossMusic = function (customFade) {
    if (!bossMusicNodes) {
      audio.bossMusicActive = false;
      audio.bossCrossfading = false;
      audio.bossFadingOut = false;
      return;
    }
    const nodes = bossMusicNodes;
    bossMusicNodes = null;
    audio.bossMusicActive = false;
    audio.bossCrossfading = false;
    const c = audio.context;
    const fade = typeof customFade === 'number' && customFade >= 0 ? customFade : 0;
    if (fade > 0 && c && c.state === 'running') {
      const stopTime = c.currentTime + fade;
      nodes.gains.forEach(g => {
        try { g.gain.setValueAtTime(g.gain.value, c.currentTime); g.gain.linearRampToValueAtTime(0.0001, stopTime); } catch (_) {}
      });
      nodes.sources.forEach(s => { try { s.stop(stopTime + 0.05); } catch (_) {} });
      setTimeout(() => {
        nodes.sources.forEach(s => { try { s.disconnect(); } catch (_) {} });
        nodes.gains.forEach(g => { try { g.disconnect(); } catch (_) {} });
        audio.bossFadingOut = false;
      }, (fade + 0.1) * 1000);
    } else {
      audio.bossFadingOut = false;
      nodes.sources.forEach(s => { try { s.stop(); } catch (_) {} s.disconnect(); });
      nodes.gains.forEach(g => { try { g.disconnect(); } catch (_) {} });
    }
  };
  audio.bossFadingOut = false;
  audio.fadeBossMusic = function (customFade) {
    if (!audio.bossMusicActive && !bossMusicNodes) return false;
    const fade = typeof customFade === 'number' && customFade >= 0 ? customFade : 1.8;
    audio.bossFadingOut = true;
    const c = audio.context;
    if (c && c.state === 'running' && activeAmbience?.gains?.length) {
      activeAmbience.gains.slice(1).forEach(g => {
        try {
          g.gain.setValueAtTime(g.gain.value, c.currentTime);
          g.gain.linearRampToValueAtTime(0.017, c.currentTime + fade);
        } catch (_) {}
      });
    }
    audio.stopBossMusic(fade);
    return true;
  };
  audio.area = function (index, customFade) {
    if (room === index && activeAmbience?.sources?.length) return;
    const prevRoom = room;
    room = index;
    const c = audio.context; if (!c||!active) return;
    const prevRegion = prevRoom >= 0 ? Math.floor(prevRoom / 3) : -1;
    const region=Math.floor(index/3), tier=index%3;
    audio.currentRegion = region;
    const hasPrevious = Boolean(activeAmbience?.sources?.length);
    const isRegionTransition = hasPrevious && prevRegion >= 0 && prevRegion !== region;
    const fade = typeof customFade === 'number' && customFade >= 0 ? customFade : isRegionTransition ? 1.5 : hasPrevious ? 0.6 : 0;
    if (hasPrevious) {
      const old = activeAmbience;
      outgoingAmbience.add(old);
      const stopTime = c.currentTime + fade;
      old.gains.forEach(g => {
        try { g.gain.setValueAtTime(g.gain.value, c.currentTime); g.gain.linearRampToValueAtTime(0.0001, stopTime); } catch (_) {}
      });
      old.sources.forEach(s => { try { s.stop(stopTime + 0.05); } catch (_) {} });
      if (fade > 0) {
        audio.crossfading = true;
        setTimeout(() => {
          old.sources.forEach(s => { try { s.disconnect(); } catch (_) {} });
          old.gains.forEach(g => { try { g.disconnect(); } catch (_) {} });
          outgoingAmbience.delete(old);
          if (outgoingAmbience.size === 0) audio.crossfading = false;
        }, (fade + 0.1) * 1000);
      } else {
        old.sources.forEach(s => { try { s.stop(); } catch (_) {} s.disconnect(); });
        old.gains.forEach(g => { try { g.disconnect(); } catch (_) {} });
        outgoingAmbience.delete(old);
      }
    }
    const currentSources = [], currentGains = [];
    const wind = c.createBufferSource(), filter = c.createBiquadFilter(), gain = c.createGain();
    wind.buffer = noise; wind.loop = true; filter.type = 'lowpass'; filter.frequency.value = [460,780,1100,640,350,260,500,390,180,220][region]||460;
    if (fade > 0 && hasPrevious) {
      gain.gain.setValueAtTime(0.0001, c.currentTime);
      gain.gain.linearRampToValueAtTime(.14, c.currentTime + fade);
    } else {
      gain.gain.value = .14;
    }
    wind.connect(filter); filter.connect(gain); gain.connect(ambient);
    wind.onended = () => { filter.disconnect(); gain.disconnect(); };
    wind.start();
    currentSources.push(wind); currentGains.push(gain);
    const root=[130.81,73.42,146.83,82.41,98,65.41,87.31,110,61.74,55][region]||130.81;
    [root,root*1.5,root*2].map(f=>f*(tier===2?.75:tier===1?.9:1)).forEach(f => {
      const osc = c.createOscillator(), level = c.createGain(); osc.type = 'sine'; osc.frequency.value = f;
      if (fade > 0 && hasPrevious) {
        level.gain.setValueAtTime(0.0001, c.currentTime);
        level.gain.linearRampToValueAtTime(.017, c.currentTime + fade);
      } else {
        level.gain.value = .017;
      }
      osc.connect(level); level.connect(music); osc.onended = () => level.disconnect(); osc.start();
      currentSources.push(osc); currentGains.push(level);
    });
    activeAmbience = { sources: currentSources, gains: currentGains };
    ambientNodes = currentSources;
    nextBird = c.currentTime + 2;
  };
  audio.tick = function () {
    if (!active || audio.steadyAmbience || !audio.context || audio.context.state !== 'running') return;
    const c = audio.context;
    if (c.currentTime > nextBird) {
      if (room < 2) { tone(1800,2400,.13,.024,'sine',0,-.6,ambient); tone(2100,1600,.16,.02,'sine',.2,.5,ambient); }
      else tone(62,46,.8,.045,'sine',0,.5,ambient);
      nextBird = c.currentTime + 3 + Math.random()*4;
    }
  };
  audio.setActive = async function (value, index) {
    const intent=++activation;previewIntent++;previewRequested=false;clearTimeout(previewTimer);active=value;
    if(value){const ready=await audio.unlock();if(ready&&intent===activation&&active)audio.area(index||0);else if(!active&&!previewRequested&&audio.context?.state==='running')await audio.context.suspend().catch(()=>{});}
    else{stopAmbience();stopVoices();if(audio.context&&audio.context.state==='running'){try{await audio.context.suspend();}catch(_){}}}
  };
  audio.set = function (key, value) {
    if(!['muted','mono','interfaceMuted','steadyAmbience',...Object.keys(buses())].includes(key))return;
    audio[key]=key==='muted'||key==='mono'||key==='interfaceMuted'||key==='steadyAmbience'?!!value:clamp(value);save(key,audio[key]);
    window.dispatchEvent(new Event('riftaudiochange'));
    if(audio.context){busGain(master,audio.muted?0:.6);for(const [name,bus] of Object.entries(buses()))busGain(bus,channelLevel(name));for(const [panner,position] of panners)panner.pan.setTargetAtTime(audio.mono?0:position,audio.context.currentTime,.03);}
  };
  audio.resetMix=()=>{for(const [key,value] of Object.entries({effects:.65,ambience:.35,music:.35,voice:.65,interface:.65,mono:false,interfaceMuted:false,steadyAmbience:false}))audio.set(key,value);};
  const creatureCues=new Set(['goblin_attack','knight_attack','treasure_attack','goblin_death','knight_death','treasure_death','treasure_escape','archer_death','boss_roar','boss_death','slam','arrow','fire','ice','void','poison','radiant','rune']);
  audio.previewCue=kind=>creatureCues.has(kind)?audio.preview('voice',kind):Promise.resolve(false);
  audio.cancelPreview=()=>{previewIntent++;previewRequested=false;clearTimeout(previewTimer);if(!active){stopVoices();if(audio.context?.state==='running')audio.context.suspend().catch(()=>{});}};
  audio.preview=async (channel,cueKind)=>{
    if(!Object.hasOwn(buses(),channel))return false;
    const intent=++previewIntent;previewRequested=true;clearTimeout(previewTimer);if(!active){stopAmbience();stopVoices();}
    const ready=await audio.unlock();if(intent!==previewIntent||document.hidden){if(document.hidden)previewRequested=false;if(!active&&!previewRequested&&audio.context?.state==='running')await audio.context.suspend().catch(()=>{});return false;}
    if(!ready||audio.muted||channel==='interface'&&audio.interfaceMuted){previewRequested=false;if(!active&&audio.context?.state==='running')await audio.context.suspend().catch(()=>{});return false;}
    const bus=buses()[channel];
    if(cueKind)playCue(cueKind,0);
    else if(channel==='ambience')hiss(.45,.1,900,0,0,bus);
    else if(channel==='voice')tone(170,65,.4,.09,'sawtooth',0,0,bus);
    else if(channel==='music')[220,330,440].forEach(f=>tone(f,f,.5,.035,'sine',0,0,bus));
    else tone(channel==='interface'?540:350,channel==='interface'?720:100,.2,.09,'triangle',0,0,bus);
    previewTimer=setTimeout(()=>{if(intent!==previewIntent)return;previewRequested=false;if(!active&&audio.context?.state==='running')audio.context.suspend().catch(()=>{});},cueKind?1750:650);
    return true;
  };
  window.addEventListener('pagehide', () => { active = false;activation++;previewIntent++;previewRequested=false;clearTimeout(previewTimer);stopAmbience();stopVoices(); if (audio.context) audio.context.close().catch(() => {}); });
  window.RiftAudio = audio;
})();
