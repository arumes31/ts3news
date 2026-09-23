/* Original procedural arcade sound design. No downloads or third-party samples. */
(function () {
  'use strict';
  function setting(key, fallback) { try { const value = JSON.parse(localStorage.getItem('riftAudio:' + key)); return value === null ? fallback : value; } catch (_) { return fallback; } }
  function levelSetting(key,fallback){const value=setting(key,fallback);return typeof value==='number'&&Number.isFinite(value)?Math.max(0,Math.min(1,value)):fallback;}
  const audio = { context: null, muted: setting('muted', false)===true, effects: levelSetting('effects', .65), ambience: levelSetting('ambience', .35), played: 0, voices: 0 };
  audio.music=levelSetting('music',audio.ambience);audio.voice=levelSetting('voice',audio.effects);audio.interface=levelSetting('interface',audio.effects);audio.mono=setting('mono',false)===true;
  audio.steadyAmbience=setting('steadyAmbience',false)===true;
  audio.interfaceMuted=setting('interfaceMuted',false)===true;
  audio.nightMode=setting('nightMode',false)===true;
  audio.dynamicRange=setting('dynamicRange',audio.nightMode?'night':'standard');
  if(audio.dynamicRange==='night')audio.nightMode=true;
  audio.streamerMusic=setting('streamerMusic',false)===true;
  audio.musicPreset=setting('musicPreset',audio.streamerMusic?'streamer':'standard');
  if(audio.musicPreset==='streamer')audio.streamerMusic=true;
  const channelLevel=key=>(key==='interface'&&audio.interfaceMuted)||(key==='music'&&(audio.streamerMusic||audio.musicPreset==='streamer'))?0:audio[key];
  audio.channelLevel = channelLevel;
  let audioBlocked = false;
  if (typeof navigator !== 'undefined' && typeof navigator.getAutoplayPolicy === 'function') {
    try {
      if (navigator.getAutoplayPolicy('audiocontext') === 'disallowed') audioBlocked = true;
    } catch (_) {}
  }
  let master, sfx, ambient, music, voice, interfaceBus, limiter, noise, ambientNodes = [], active = false, room = -1, nextBird = 0, activation = 0, previewIntent = 0, previewTimer = 0;
  Object.defineProperty(audio, 'active', { get() { return active; }, configurable: true });
  Object.defineProperty(audio, 'limiter', { get() { return limiter; }, configurable: true });
  Object.defineProperty(audio, 'musicBus', { get() { return music; }, configurable: true });
  Object.defineProperty(audio, 'blocked', {
    get() { return audioBlocked; },
    set(v) {
      const next = Boolean(v);
      if (audioBlocked !== next) {
        audioBlocked = next;
        window.dispatchEvent(new Event('riftaudiochange'));
      }
    },
    configurable: true
  });
  audio.isBlocked = function () {
    if (audio.context && audio.context.state === 'running') return false;
    return audioBlocked;
  };
  audio.isActive = function () { return active; };
  const panners=new Map(),sources=new Map();let previewRequested=false;
  const buses=()=>({effects:sfx,ambience:ambient,music,voice,interface:interfaceBus});
  const clamp = value => Math.max(0, Math.min(1, Number(value) || 0));
  function save(key, value) { try { localStorage.setItem('riftAudio:' + key, JSON.stringify(value)); } catch (_) {} }
  // Freeze migrated values once so later parent-channel edits stay independent.
  for(const key of ['music','voice','interface'])save(key,audio[key]);
  audio.isDragging = false;
  audio.activeDragKey = null;
  audio.smoothRampTime = 0.05;
  audio.isDraggingSlider = function (key) {
    return audio.isDragging && (!key || audio.activeDragKey === key);
  };
  let saveTimer = null;
  const pendingSaves = new Set();
  function flushPendingSaves() {
    if (saveTimer) { clearTimeout(saveTimer); saveTimer = null; }
    for (const k of pendingSaves) save(k, audio[k]);
    pendingSaves.clear();
  }
  audio.flushPendingSaves = flushPendingSaves;
  function busGain(bus, value, rampTime = 0.03) {
    if (!bus || !audio.context) return;
    const t = audio.context.currentTime;
    try {
      if (value === 0 && !audio.isDragging) {
        bus.gain.setValueAtTime(0, t);
      } else {
        bus.gain.setTargetAtTime(value, t, rampTime);
      }
    } catch (_) {
      try { bus.gain.value = value; } catch (__) {}
    }
  }
  function applyDynamicRange(rampTime = 0.03) {
    if (!audio.context || !limiter) return;
    const isNight = audio.nightMode || audio.dynamicRange === 'night';
    const t = audio.context.currentTime;
    const threshold = isNight ? -28 : -16;
    const ratio = isNight ? 14 : 8;
    const knee = isNight ? 10 : 30;
    const attack = isNight ? 0.002 : 0.003;
    const release = isNight ? 0.20 : 0.25;
    try {
      limiter.threshold.value = threshold;
      limiter.ratio.value = ratio;
      limiter.knee.value = knee;
      limiter.attack.value = attack;
      limiter.release.value = release;
      if (master) {
        const targetMaster = audio.muted ? 0 : isNight ? 0.45 : 0.6;
        if (rampTime > 0) {
          master.gain.setTargetAtTime(targetMaster, t, rampTime);
        } else {
          master.gain.value = targetMaster;
        }
      }
    } catch (_) {}
  }
  function initContext() {
    const AC = window.AudioContext || window.webkitAudioContext;
    if (!AC) return false;
    const c = audio.context = new AC();
    master = c.createGain(); sfx = c.createGain(); ambient = c.createGain(); music = c.createGain(); voice = c.createGain(); interfaceBus = c.createGain();
    limiter = c.createDynamicsCompressor();
    applyDynamicRange(0);
    Object.values(buses()).forEach(bus => bus.connect(master)); master.connect(limiter); limiter.connect(c.destination);
    master.gain.value = audio.muted ? 0 : (audio.nightMode || audio.dynamicRange === 'night' ? 0.45 : 0.6);
    for (const [key, bus] of Object.entries(buses())) bus.gain.value = channelLevel(key);
    noise = c.createBuffer(1, c.sampleRate * 2, c.sampleRate);
    const data = noise.getChannelData(0);
    for (let i = 0; i < data.length; i++) data[i] = Math.random() * 2 - 1;
    c.onstatechange = () => {
      if (c.state === 'running') {
        audio.blocked = false;
      } else if (c.state === 'suspended' && audioBlocked) {
        window.dispatchEvent(new Event('riftaudiochange'));
      }
    };
    return true;
  }
  audio.unlock = async function () {
    try {
      if (audio.context?.state === 'closed') {
        clearAmbienceTimers?.(); stopAmbience(); stopVoices(); audio.releaseAbandonedAmbience?.(); audio.context = null; audio.voices = 0; panners.clear();
      }
      if (!audio.context) {
        if (!initContext()) {
          audio.blocked = true;
          return false;
        }
      }
      if (audio.context.state !== 'running') {
        try {
          await audio.context.resume();
        } catch (_) {
          try { await audio.context.close(); } catch (_) {}
          audio.context = null; audio.voices = 0; panners.clear();
          if (!initContext()) {
            audio.blocked = true;
            return false;
          }
          if (audio.context.state !== 'running') {
            try { await audio.context.resume(); } catch (_) {}
          }
        }
      }
      const ready = audio.context.state === 'running';
      audio.blocked = !ready;
      return ready;
    } catch (_) {
      audio.blocked = true;
      return false;
    }
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
  const recentImpactTimes = new Map();
  audio.isImpactCue = function (kind) {
    return kind === 'hit' || kind.startsWith('hit_') || kind === 'slash' || kind === 'knockdown' || kind === 'slam' || kind === 'block' || kind === 'perfect_guard' || kind.endsWith('_hurt');
  };
  audio.shouldThrottleImpact = function (kind, now = performance.now()) {
    if (!audio.isImpactCue(kind)) return false;
    const history = (recentImpactTimes.get(kind) || []).filter(t => now - t < 50);
    return history.filter(t => now - t < 38).length >= 2;
  };
  audio.clearImpactHistory = function () {
    recentImpactTimes.clear();
  };
  audio.isEnemyCue = function (kind) {
    return /(?:_attack|_death|_roar|_escape|_hurt)$/.test(kind) || kind === 'slam' || kind === 'arrow';
  };
  audio.distanceAttenuation = function (dist) {
    if (typeof dist !== 'number' || !Number.isFinite(dist) || dist <= 0) return 1.0;
    const minDistance = 120;
    const maxDistance = 850;
    const minGain = 0.3;
    if (dist <= minDistance) return 1.0;
    if (dist >= maxDistance) return minGain;
    const t = (dist - minDistance) / (maxDistance - minDistance);
    return 1.0 - t * (1.0 - minGain);
  };
  let lastUINavTime = 0;
  audio.canPlayUINavigation = function (isHeld = false, now = performance.now()) {
    const minInterval = isHeld ? 110 : 35;
    return (now - lastUINavTime) >= minInterval;
  };
  audio.recordUINavigation = function (now = performance.now()) {
    lastUINavTime = now;
  };
  audio.playUINav = function (pan = 0, isHeld = false, now = performance.now()) {
    if (!audio.context || audio.context.state !== 'running' || audio.muted || !active) return false;
    if (!audio.canPlayUINavigation(isHeld, now)) return false;
    lastUINavTime = now;
    const att = isHeld ? 0.7 : 1.0;
    return playCue('ui', pan || 0, att);
  };
  audio.resetUINavLimits = function () {
    lastUINavTime = 0;
  };
  audio.play = function (kind, pan, extra, extra2, distance) {
    if (!audio.context || audio.context.state !== 'running' || audio.muted || !active) return false;
    if (kind === 'step' && extra) { audio.step(extra, pan); return; }
    if (kind === 'land') { audio.land(extra, pan, extra2); return; }
    if (kind === 'finisher_cast') { audio.finisherCast(extra, pan); return; }
    if (kind === 'hit' && extra) { audio.hit(extra, pan); return; }
    if (kind === 'hurt' && extra) { audio.hurt(extra, pan, distance); return; }
    if (kind === 'death' && extra) { audio.death(extra, pan, distance); return; }
    if (kind === 'ui') {
      const isHeld = Boolean(extra);
      const now = performance.now();
      if (!audio.canPlayUINavigation(isHeld, now)) return false;
      lastUINavTime = now;
      const att = isHeld ? 0.7 : 1.0;
      return playCue('ui', pan || 0, att);
    }
    const att = audio.isEnemyCue(kind) ? audio.distanceAttenuation(distance) : 1.0;
    return playCue(kind, pan, att);
  };
  audio.step = function (material, pan) {
    if (!audio.context || audio.context.state !== 'running' || audio.muted || !active) return;
    const mat = String(material || 'stone').toLowerCase();
    const cue = ['metal','wood','water','mud','ice','grass','dirt','stone'].includes(mat) ? 'step_' + mat : 'step';
    playCue(cue, pan || 0);
  };
  audio.hit = function (family, pan) {
    if (!audio.context || audio.context.state !== 'running' || audio.muted || !active) return;
    const fam = String(family || 'blade').toLowerCase();
    const cue = ['blade','blunt','pierce','arcane','fist','ranged'].includes(fam) ? 'hit_' + fam : 'hit';
    playCue(cue, pan || 0);
  };
  audio.hurt = function (kind, pan, distance) {
    if (!audio.context || audio.context.state !== 'running' || audio.muted || !active) return;
    const k = String(kind || 'hurt').toLowerCase();
    const cue = ['goblin','knight','archer','treasure','boss','wolf','spore'].includes(k) ? k + '_hurt' : 'hurt';
    const att = audio.isEnemyCue(cue) ? audio.distanceAttenuation(distance) : 1.0;
    playCue(cue, pan || 0, att);
  };
  audio.death = function (kind, pan, distance) {
    if (!audio.context || audio.context.state !== 'running' || audio.muted || !active) return;
    const k = String(kind || 'goblin').toLowerCase();
    const cue = ['goblin','knight','archer','treasure','boss','wolf','spore'].includes(k) ? k + '_death' : 'goblin_death';
    const att = audio.isEnemyCue(cue) ? audio.distanceAttenuation(distance) : 1.0;
    playCue(cue, pan || 0, att);
  };
  audio.finisherCast = function (charges, pan) {
    if (!audio.context || audio.context.state !== 'running' || audio.muted || !active) return;
    const c = Math.max(0, Math.round(Number(charges) || 0));
    const vol = c >= 3 ? 1.3 : c > 0 ? 1.0 : 0.6;
    hiss(.25, .10 * vol, 4800, pan || 0, 0, sfx);
    tone(140, 320, .32, .14 * vol, 'sine', 0, pan || 0, sfx);
    [440, 660, 880, 1320].slice(0, Math.max(2, c + 1)).forEach((f, i) => {
      tone(f, f * 1.05, .35, .06 * vol, 'triangle', i * .04, pan || 0, sfx);
    });
    if (c >= 3) {
      tone(90, 45, .4, .18 * vol, 'triangle', .02, pan || 0, sfx);
    }
  };
  audio.land = function (intensity = 0.5, pan = 0, material) {
    if (!audio.context || audio.context.state !== 'running' || audio.muted || !active) return;
    let numeric = 0.5;
    if (typeof intensity === 'string') {
      const lowered = intensity.toLowerCase();
      numeric = lowered === 'heavy' ? 0.9 : lowered === 'light' ? 0.3 : 0.6;
    } else if (typeof intensity === 'number' && Number.isFinite(intensity)) {
      numeric = intensity;
    }
    const cue = numeric < 0.45 ? 'land_light' : numeric >= 0.75 ? 'land_heavy' : 'land';
    playCue(cue, pan || 0);

    const mat = String(material || '').toLowerCase();
    if (mat && mat !== 'stone') {
      const vol = numeric < 0.45 ? 0.6 : numeric >= 0.75 ? 1.4 : 1.0;
      if (mat === 'metal') {
        hiss(.08, .05 * vol, 2900, pan || 0, 0, sfx);
        tone(640, 400, .08, .06 * vol, 'triangle', 0, pan || 0, sfx);
      } else if (mat === 'water') {
        hiss(.13, .08 * vol, 1500, pan || 0, 0, sfx);
        tone(260, 95, .11, .06 * vol, 'sine', 0, pan || 0, sfx);
      } else if (mat === 'wood') {
        tone(200, 80, .09, .08 * vol, 'triangle', 0, pan || 0, sfx);
      } else if (mat === 'ice') {
        hiss(.10, .07 * vol, 4500, pan || 0, 0, sfx);
        tone(800, 580, .06, .04 * vol, 'sine', 0, pan || 0, sfx);
      } else if (mat === 'mud') {
        hiss(.11, .07 * vol, 950, pan || 0, 0, sfx);
        tone(160, 70, .09, .05 * vol, 'triangle', 0, pan || 0, sfx);
      }
    }
  };
  function playCue(kind, pan, attenuation = 1.0) {
    const now = performance.now();
    if (audio.isImpactCue(kind)) {
      if (audio.shouldThrottleImpact(kind, now)) {
        return false;
      }
      const history = (recentImpactTimes.get(kind) || []).filter(t => now - t < 50);
      history.push(now);
      recentImpactTimes.set(kind, history);
    }
    audio.played++;
    const target=kind==='ui'||kind==='bank'||kind==='empty_mana'||kind==='cooldown_rejection'?interfaceBus:/(?:_attack|_death|_roar|_escape|_hurt)$/.test(kind)?voice:sfx;
    let gainMult = typeof attenuation === 'number' && Number.isFinite(attenuation) ? Math.max(0.05, Math.min(1.0, attenuation)) : 1.0;
    if (audio.isImpactCue(kind)) {
      const history = recentImpactTimes.get(kind) || [];
      if (history.length > 1) {
        gainMult *= 0.75;
      }
    }
    const detune = audio.isImpactCue(kind) && (recentImpactTimes.get(kind)?.length || 0) > 1
      ? 0.96 + Math.random() * 0.08
      : 1.0;
    const t = (f, end, d, v, wave, delay) => tone(f * detune, end * detune, d, v * gainMult, wave, delay, pan, target);
    const h = (duration, volume, cutoff, position, delay) => hiss(duration, volume * gainMult, cutoff ? cutoff * Math.max(0.6, gainMult) : cutoff, position, delay, target);
    switch (kind) {
      case 'step': case 'step_stone': h(.065, .07, 650, pan); t(120, 60, .05, .04, 'triangle'); break;
      case 'step_metal': h(.07, .06, 2800, pan); t(620, 480, .06, .05, 'triangle'); break;
      case 'step_wood': h(.06, .07, 450, pan); t(160, 90, .07, .07, 'triangle'); break;
      case 'step_water': h(.11, .08, 1400, pan); t(260, 110, .08, .04, 'sine'); break;
      case 'step_mud': h(.1, .075, 950, pan); t(180, 80, .07, .045, 'triangle'); break;
      case 'step_ice': h(.08, .075, 4500, pan); t(880, 720, .05, .03, 'sine'); break;
      case 'step_grass': case 'step_dirt': h(.07, .06, 380, pan); t(90, 50, .06, .035, 'triangle'); break;
      case 'jump': t(160, 480, .15, .09, 'triangle'); h(.09, .04, 1200, pan); break;
      case 'land_light': h(.06, .05, 800, pan); t(140, 80, .05, .05, 'triangle'); break;
      case 'land': case 'land_medium': h(.10, .09, 650, pan); t(110, 50, .09, .08, 'triangle'); break;
      case 'land_heavy': h(.16, .14, 520, pan); t(90, 32, .18, .15, 'triangle'); t(70, 25, .22, .12, 'sine', .01); break;
      case 'slash': h(.14, .15, 3800, pan); t(350, 100, .12, .05, 'sawtooth'); break;
      case 'third_strike': h(.18, .25, 2600, pan); t(220, 50, .24, .25, 'triangle'); t(90, 25, .28, .22, 'sine', .01); t(520, 180, .10, .12, 'sawtooth'); break;
      case 'finisher_cast': {
        h(.25, .10, 4800, pan);
        t(140, 320, .32, .14, 'sine');
        [440, 660, 880].forEach((f, i) => t(f, f * 1.05, .35, .06, 'triangle', i * .04));
        break;
      }
      case 'heavy_recovery': {
        h(.12, .07, 1400, pan);
        t(130, 55, .16, .08, 'triangle');
        break;
      }
      case 'shield_absorb': {
        h(.16, .12, 5400, pan);
        t(380, 180, .18, .12, 'sine');
        [1200, 1600, 2100].forEach((f, i) => t(f, f * 0.95, .15, .05, 'triangle', i * .02));
        break;
      }
      case 'mark_target': {
        h(.08, .09, 4200, pan);
        t(880, 1320, .14, .08, 'triangle');
        t(1760, 1760, .12, .05, 'sine', .02);
        break;
      }
      case 'thaw': {
        h(.08, .12, 6000, pan);
        t(1600, 520, .14, .09, 'triangle');
        [880, 1175, 1480].forEach((f, i) => t(f, f * 1.05, .18, .05, 'sine', i * .03));
        break;
      }
      case 'boss_stagger': {
        h(.25, .18, 900, pan);
        t(140, 42, .32, .18, 'triangle');
        t(70, 32, .38, .15, 'sawtooth', .02);
        [340, 480].forEach((f, i) => t(f, f * 0.9, .22, .06, 'triangle', i * .04));
        break;
      }
      case 'boss_phase': {
        h(.55, .22, 1200, pan);
        t(75, 30, .75, .22, 'sawtooth');
        t(110, 48, .65, .16, 'triangle', .03);
        [220, 330, 440].forEach((f, i) => t(f, f * 1.15, .45, .08, 'sawtooth', i * .05));
        break;
      }
      case 'victory': {
        h(.35, .15, 2400, pan);
        t(110, 220, .45, .16, 'sawtooth');
        [440, 554, 659, 880].forEach((f, i) => t(f, f * 1.02, .5, .08, 'triangle', i * .06));
        [1320, 1760].forEach((f, i) => t(f, f, .4, .04, 'sine', .24 + i * .05));
        break;
      }
      case 'hit': case 'hit_blade': h(.08, .18, 3200, pan); t(420, 180, .09, .14, 'sawtooth'); t(130, 60, .12, .15, 'triangle'); break;
      case 'hit_blunt': h(.14, .24, 750, pan); t(150, 40, .20, .24, 'triangle'); t(80, 30, .22, .18, 'sine', .01); break;
      case 'hit_pierce': h(.05, .20, 5200, pan); t(980, 420, .06, .12, 'triangle'); t(180, 85, .08, .12, 'triangle'); break;
      case 'hit_arcane': h(.18, .16, 6000, pan); t(523, 392, .18, .12, 'sine'); t(784, 523, .16, .10, 'triangle', .02); break;
      case 'hit_fist': h(.09, .16, 900, pan); t(140, 50, .14, .18, 'triangle'); break;
      case 'hit_ranged': h(.07, .18, 4800, pan); t(820, 340, .07, .10, 'triangle'); t(160, 70, .09, .12, 'triangle'); break;
      case 'hurt': t(170, 65, .2, .1, 'sawtooth'); h(.12, .12, 950, pan); break;
      case 'goblin_hurt': t(360, 150, .14, .08, 'sawtooth'); h(.07, .06, 1800, pan); break;
      case 'knight_hurt': h(.16, .14, 1100, pan); t(130, 65, .16, .12, 'triangle'); t(90, 45, .18, .08, 'sawtooth'); break;
      case 'archer_hurt': t(480, 220, .12, .06, 'square'); h(.1, .07, 3200, pan); break;
      case 'treasure_hurt': [784, 1175].forEach((f, i) => t(f, f * .8, .12, .06, 'triangle', i * .04)); h(.08, .07, 5000, pan); break;
      case 'boss_hurt': t(110, 45, .35, .16, 'sawtooth'); t(75, 30, .4, .14, 'sawtooth', .02); h(.3, .12, 600, pan); break;
      case 'wolf_hurt': t(380, 210, .12, .07, 'sine'); h(.06, .05, 1500, pan); break;
      case 'spore_hurt': h(.14, .08, 900, pan); t(220, 110, .12, .07, 'triangle'); break;
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
      case 'ultimate_anticipation': t(130, 480, .55, .15, 'sawtooth'); t(65, 240, .6, .18, 'sine'); h(.4, .12, 3500, pan); break;
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
      case 'wolf_death': t(420, 180, .5, .1, 'sine'); t(280, 110, .45, .08, 'triangle', .05); h(.2, .05, 1400, pan); break;
      case 'spore_death': h(.35, .15, 800, pan); t(180, 45, .25, .12, 'sine'); t(240, 60, .15, .08, 'triangle', .04); break;
      case 'guardian_linked': t(330,660,.45,.05,'triangle'); break;
      case 'guardian_unlinked': t(660,220,.4,.05,'sine'); break;
      case 'guardians_defeated': [440,660,880].forEach((f,i)=>t(f,f,.5,.045,'sine',i*.13)); break;
      case 'collapse_start': t(180,45,1,.065,'triangle'); break;
      case 'collapse_hit': t(90,35,.35,.07,'triangle'); break;
      case 'collapse_escaped': [392,587,784].forEach((f,i)=>t(f,f,.6,.045,'sine',i*.12)); break;
      case 'ritual_warning': t(220,440,.7,.05,'triangle'); break;
      case 'ritual_interrupt': t(660,220,.25,.05,'sine'); break;
      case 'ritual_pulse': t(110,55,.5,.07,'triangle'); break;
      case 'ritual_complete': [330,440,660].forEach((f,i)=>t(f,f,.5,.045,'sine',i*.12)); break;
      case 'spirit_move': t(440,660,.45,.04,'sine'); break;
      case 'spirit_threat': t(330,220,.35,.055,'sine'); break;
      case 'spirit_arrived': [523,784,1047].forEach((f,i)=>t(f,f,.65,.045,'sine',i*.14)); break;
      case 'beacon_charge': t(440,660,.2,.04,'sine'); break;
      case 'beacon_captured': [660,880,1100].forEach((f,i)=>t(f,f,.3,.05,'sine',i*.07)); break;
      case 'beacons_complete': [523,659,784,1047].forEach((f,i)=>t(f,f,.4,.055,'triangle',i*.1)); break;
      case 'hunt_complete': [440,554,659,880].forEach((f,i)=>t(f,f,.4,.055,'triangle',i*.1)); break;
      case 'generator_hurt': h(.12,.06,3400,pan); t(420,180,.15,.045,'triangle'); break;
      case 'generator_break': h(.4,.12,2400,pan); t(440,70,.5,.07,'sawtooth'); break;
      case 'generator_shutdown': t(660,110,.65,.055,'sine'); break;
      case 'relic_pickup': [330,440,660].forEach((f,i)=>t(f,f,.3,.05,'sine',i*.09)); break;
      case 'relic_delivered': [523,659,784,1047].forEach((f,i)=>t(f,f,.5,.06,'triangle',i*.1)); break;
      case 'totem_hurt': h(.12,.07,2200,pan); t(160,95,.15,.05,'triangle'); break;
      case 'totem_break': h(.5,.12,3200,pan); t(320,60,.5,.08,'triangle'); break;
      case 'wave_incoming': [0,.25].forEach(delay=>t(220,330,.2,.065,'triangle',delay)); break;
      case 'wave_start': t(110,55,.35,.08,'triangle'); t(440,660,.22,.055,'sine',.1); break;
      case 'waves_complete': [392,523,659,784].forEach((f,i)=>t(f,f,.3,.055,'triangle',i*.08)); break;
      case 'circle_charge': t(392,587,.35,.05,'sine'); break;
      case 'circle_contested': t(196,147,.2,.06,'triangle'); break;
      case 'circle_complete': [523,659,784,1047].forEach((f,i)=>t(f,f,.4,.055,'triangle',i*.09)); break;
      case 'sigil_pickup': [784,1175,1568].forEach((f,i)=>t(f,f*1.01,.28,.055,'sine',i*.07)); break;
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
    return true;
  };
  function stopVoices(){recentImpactTimes.clear();for(const [source,cleanup] of [...sources]){try{source.stop();}catch(_){}cleanup();}}
  const trackedAmbienceNodes = new Set();
  const pendingAmbienceTimers = new Set();
  function trackAmbienceNode(node) {
    if (node) trackedAmbienceNodes.add(node);
    return node;
  }
  function releaseAmbienceNode(node) {
    if (!node) return;
    try { node.stop?.(); } catch (_) {}
    try { node.disconnect?.(); } catch (_) {}
    trackedAmbienceNodes.delete(node);
  }
  function clearAmbienceTimers() {
    for (const timer of pendingAmbienceTimers) {
      clearTimeout(timer);
    }
    pendingAmbienceTimers.clear();
  }
  function releaseAmbienceGroup(group) {
    if (!group) return;
    if (group.timer) {
      clearTimeout(group.timer);
      pendingAmbienceTimers.delete(group.timer);
      group.timer = null;
    }
    group.sources?.forEach(releaseAmbienceNode);
    group.filters?.forEach(releaseAmbienceNode);
    group.gains?.forEach(releaseAmbienceNode);
    group.sources = [];
    group.filters = [];
    group.gains = [];
  }
  audio.getTrackedAmbienceCount = function () {
    return trackedAmbienceNodes.size;
  };
  audio.releaseAbandonedAmbience = function () {
    const liveNodes = new Set();
    if (activeAmbience) {
      activeAmbience.sources?.forEach(n => liveNodes.add(n));
      activeAmbience.filters?.forEach(n => liveNodes.add(n));
      activeAmbience.gains?.forEach(n => liveNodes.add(n));
    }
    for (const group of outgoingAmbience) {
      group.sources?.forEach(n => liveNodes.add(n));
      group.filters?.forEach(n => liveNodes.add(n));
      group.gains?.forEach(n => liveNodes.add(n));
    }
    if (bossMusicNodes) {
      bossMusicNodes.sources?.forEach(n => liveNodes.add(n));
      bossMusicNodes.filters?.forEach(n => liveNodes.add(n));
      bossMusicNodes.gains?.forEach(n => liveNodes.add(n));
    }
    let releasedCount = 0;
    for (const node of [...trackedAmbienceNodes]) {
      if (!liveNodes.has(node)) {
        releaseAmbienceNode(node);
        releasedCount++;
      }
    }
    return releasedCount;
  };
  let activeAmbience = null; const outgoingAmbience = new Set();
  let bossMusicNodes = null;
  audio.crossfading = false;
  audio.bossMusicActive = false;
  audio.bossCrossfading = false;
  function stopAmbience() {
    audio.stopBossMusic?.(0);
    room = -1;
    audio.currentRegion = -1;
    clearAmbienceTimers();
    if (activeAmbience) {
      releaseAmbienceGroup(activeAmbience);
      activeAmbience = null;
    }
    for (const group of outgoingAmbience) {
      releaseAmbienceGroup(group);
    }
    outgoingAmbience.clear();
    ambientNodes = [];
    audio.crossfading = false;
    audio.releaseAbandonedAmbience();
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
    const sources = [], filters = [], gains = [];
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
    bass.start();
    trackAmbienceNode(bass); trackAmbienceNode(bassFilter); trackAmbienceNode(bassGain);
    sources.push(bass); filters.push(bassFilter); gains.push(bassGain);
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
      osc.start();
      trackAmbienceNode(osc); trackAmbienceNode(level);
      sources.push(osc); gains.push(level);
    });
    bossMusicNodes = { sources, filters, gains, timer: null };
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
      const timer = setTimeout(() => {
        pendingAmbienceTimers.delete(timer);
        releaseAmbienceGroup(nodes);
        audio.bossFadingOut = false;
      }, (fade + 0.1) * 1000);
      nodes.timer = timer;
      pendingAmbienceTimers.add(timer);
    } else {
      audio.bossFadingOut = false;
      releaseAmbienceGroup(nodes);
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
      if (outgoingAmbience.size >= 2) {
        for (const stale of outgoingAmbience) {
          if (outgoingAmbience.size < 2) break;
          releaseAmbienceGroup(stale);
          outgoingAmbience.delete(stale);
        }
      }
      outgoingAmbience.add(old);
      const stopTime = c.currentTime + fade;
      old.gains.forEach(g => {
        try { g.gain.setValueAtTime(g.gain.value, c.currentTime); g.gain.linearRampToValueAtTime(0.0001, stopTime); } catch (_) {}
      });
      old.sources.forEach(s => { try { s.stop(stopTime + 0.05); } catch (_) {} });
      if (fade > 0) {
        audio.crossfading = true;
        const timer = setTimeout(() => {
          pendingAmbienceTimers.delete(timer);
          releaseAmbienceGroup(old);
          outgoingAmbience.delete(old);
          if (outgoingAmbience.size === 0) audio.crossfading = false;
        }, (fade + 0.1) * 1000);
        old.timer = timer;
        pendingAmbienceTimers.add(timer);
      } else {
        releaseAmbienceGroup(old);
        outgoingAmbience.delete(old);
      }
    }
    audio.releaseAbandonedAmbience();
    const currentSources = [], currentFilters = [], currentGains = [];
    const wind = c.createBufferSource(), filter = c.createBiquadFilter(), gain = c.createGain();
    wind.buffer = noise; wind.loop = true; filter.type = 'lowpass'; filter.frequency.value = [460,780,1100,640,350,260,500,390,180,220][region]||460;
    const isPaused = !audio.isRegionActive(region);
    const windBaseGain = .14;
    if (!isPaused && fade > 0 && hasPrevious) {
      gain.gain.setValueAtTime(0.0001, c.currentTime);
      gain.gain.linearRampToValueAtTime(windBaseGain, c.currentTime + fade);
    } else {
      gain.gain.value = isPaused ? 0.0001 : windBaseGain;
    }
    wind.connect(filter); filter.connect(gain); gain.connect(ambient);
    wind.onended = () => { releaseAmbienceNode(filter); releaseAmbienceNode(gain); };
    wind.start();
    trackAmbienceNode(wind); trackAmbienceNode(filter); trackAmbienceNode(gain);
    currentSources.push(wind); currentFilters.push(filter); currentGains.push(gain);
    const root=[130.81,73.42,146.83,82.41,98,65.41,87.31,110,61.74,55][region]||130.81;
    const baseGains = [windBaseGain];
    [root,root*1.5,root*2].map(f=>f*(tier===2?.75:tier===1?.9:1)).forEach(f => {
      const osc = c.createOscillator(), level = c.createGain(); osc.type = 'sine'; osc.frequency.value = f;
      const oscBaseGain = .017;
      baseGains.push(oscBaseGain);
      if (!isPaused && fade > 0 && hasPrevious) {
        level.gain.setValueAtTime(0.0001, c.currentTime);
        level.gain.linearRampToValueAtTime(oscBaseGain, c.currentTime + fade);
      } else {
        level.gain.value = isPaused ? 0.0001 : oscBaseGain;
      }
      osc.connect(level); level.connect(music);
      osc.onended = () => releaseAmbienceNode(level);
      osc.start();
      trackAmbienceNode(osc); trackAmbienceNode(level);
      currentSources.push(osc); currentGains.push(level);
    });
    activeAmbience = { region, sources: currentSources, filters: currentFilters, gains: currentGains, baseGains, paused: isPaused, timer: null };
    ambientNodes = currentSources;
    nextBird = c.currentTime + 2;
  };
  const inactiveRegions = new Set();
  audio.isRegionActive = function (region) {
    if (typeof region !== 'number' || region < 0) return true;
    return !inactiveRegions.has(region);
  };
  audio.setRegionActive = function (region, isActive, customFade) {
    if (typeof region !== 'number' || region < 0) return;
    const wasActive = !inactiveRegions.has(region);
    if (isActive) {
      inactiveRegions.delete(region);
    } else {
      inactiveRegions.add(region);
    }
    if (wasActive === Boolean(isActive)) return;

    if (activeAmbience && activeAmbience.region === region) {
      const c = audio.context;
      const fade = typeof customFade === 'number' && customFade >= 0 ? customFade : 0.4;
      if (!isActive) {
        activeAmbience.paused = true;
        if (c && c.state === 'running') {
          activeAmbience.gains.forEach(g => {
            try {
              g.gain.setValueAtTime(g.gain.value, c.currentTime);
              g.gain.linearRampToValueAtTime(0.0001, c.currentTime + fade);
            } catch (_) {}
          });
        }
      } else {
        activeAmbience.paused = false;
        if (c && c.state === 'running') {
          activeAmbience.gains.forEach((g, i) => {
            const target = activeAmbience.baseGains?.[i] || 0.017;
            try {
              g.gain.setValueAtTime(Math.max(0.0001, g.gain.value), c.currentTime);
              g.gain.linearRampToValueAtTime(target, c.currentTime + fade);
            } catch (_) {}
          });
          nextBird = c.currentTime + 2;
        }
      }
    }
  };
  audio.pauseRegionAmbience = function (region, fade) {
    audio.setRegionActive(region, false, fade);
  };
  audio.resumeRegionAmbience = function (region, fade) {
    audio.setRegionActive(region, true, fade);
  };
  audio.isAmbiencePaused = function () {
    return Boolean(activeAmbience?.paused);
  };
  audio.tick = function () {
    if (!active || audio.steadyAmbience || !audio.context || audio.context.state !== 'running') return;
    if (activeAmbience?.paused || !audio.isRegionActive(audio.currentRegion)) return;
    const c = audio.context;
    if (c.currentTime > nextBird) {
      if (room < 2) { tone(1800,2400,.13,.024,'sine',0,-.6,ambient); tone(2100,1600,.16,.02,'sine',.2,.5,ambient); }
      else tone(62,46,.8,.045,'sine',0,.5,ambient);
      nextBird = c.currentTime + 3 + Math.random()*4;
    }
  };
  audio.silence = function () {
    active = false;
    stopAmbience();
    stopVoices();
    audio.stopBossMusic?.(0);
    audio.releaseAbandonedAmbience();
    if (audio.context && audio.context.state === 'running') {
      try { audio.context.suspend().catch(() => {}); } catch (_) {}
    }
  };
  audio.isSilent = function () {
    return !active || !audio.context || audio.context.state !== 'running' || audio.muted || (activeAmbience === null && audio.voices === 0 && !audio.bossMusicActive);
  };
  audio.recover = async function (targetArea) {
    try {
      const ready = await audio.unlock();
      if (!ready) {
        audio.silence();
        return false;
      }
      if (typeof targetArea === 'number' && targetArea >= 0) {
        return await audio.setActive(true, targetArea);
      }
      return true;
    } catch (_) {
      audio.silence();
      return false;
    }
  };
  audio.setActive = async function (value, index) {
    const intent = ++activation;
    previewIntent++;
    previewRequested = false;
    clearTimeout(previewTimer);
    if (value) {
      const ready = await audio.unlock();
      if (!ready || intent !== activation) {
        audio.silence();
        audio.blocked = !ready;
        return false;
      }
      active = true;
      audio.blocked = false;
      audio.area(index || 0);
      return true;
    } else {
      audio.silence();
      return true;
    }
  };
  audio.set = function (key, value, isDragging = false) {
    if(!['muted','mono','interfaceMuted','steadyAmbience','nightMode','dynamicRange','streamerMusic','musicPreset',...Object.keys(buses())].includes(key))return;
    audio.isDragging = Boolean(isDragging);
    audio.activeDragKey = isDragging ? key : null;
    const ramp = isDragging ? audio.smoothRampTime : 0.03;
    if (key === 'nightMode') {
      audio.nightMode = !!value;
      audio.dynamicRange = audio.nightMode ? 'night' : 'standard';
      save('nightMode', audio.nightMode);
      save('dynamicRange', audio.dynamicRange);
      applyDynamicRange();
    } else if (key === 'dynamicRange') {
      audio.dynamicRange = value === 'night' ? 'night' : 'standard';
      audio.nightMode = audio.dynamicRange === 'night';
      save('dynamicRange', audio.dynamicRange);
      save('nightMode', audio.nightMode);
      applyDynamicRange();
    } else if (key === 'streamerMusic') {
      audio.streamerMusic = !!value;
      audio.musicPreset = audio.streamerMusic ? 'streamer' : 'standard';
      save('streamerMusic', audio.streamerMusic);
      save('musicPreset', audio.musicPreset);
    } else if (key === 'musicPreset') {
      audio.musicPreset = value === 'streamer' ? 'streamer' : 'standard';
      audio.streamerMusic = audio.musicPreset === 'streamer';
      save('musicPreset', audio.musicPreset);
      save('streamerMusic', audio.streamerMusic);
    } else {
      audio[key]=key==='muted'||key==='mono'||key==='interfaceMuted'||key==='steadyAmbience'?!!value:clamp(value);
      if (isDragging) {
        pendingSaves.add(key);
        if (!saveTimer) saveTimer = setTimeout(flushPendingSaves, 200);
      } else {
        flushPendingSaves();
        save(key, audio[key]);
      }
    }
    window.dispatchEvent(new Event('riftaudiochange'));
    if(audio.context){
      if (isDragging && buses()[key]) {
        busGain(buses()[key], channelLevel(key), ramp);
      } else {
        busGain(master,audio.muted?0:(audio.nightMode||audio.dynamicRange==='night'?0.45:0.6), ramp);
        for(const [name,bus] of Object.entries(buses()))busGain(bus,channelLevel(name), ramp);
        for(const [panner,position] of panners)panner.pan.setTargetAtTime(audio.mono?0:position,audio.context.currentTime,.03);
      }
    }
  };
  audio.applyDynamicRangePreset = function (preset) {
    audio.set('dynamicRange', preset);
    return audio.dynamicRange;
  };
  audio.applyMusicPreset = function (preset) {
    audio.set('musicPreset', preset);
    return audio.musicPreset;
  };
  audio.resetMix=()=>{for(const [key,value] of Object.entries({effects:.65,ambience:.35,music:.35,voice:.65,interface:.65,mono:false,interfaceMuted:false,steadyAmbience:false,nightMode:false,dynamicRange:'standard',streamerMusic:false,musicPreset:'standard'}))audio.set(key,value);};
  const creatureCues=new Set(['goblin_attack','knight_attack','treasure_attack','goblin_hurt','knight_hurt','archer_hurt','treasure_hurt','boss_hurt','wolf_hurt','spore_hurt','goblin_death','knight_death','treasure_death','treasure_escape','archer_death','boss_roar','boss_death','wolf_death','spore_death','slam','arrow','fire','ice','void','poison','radiant','rune']);
  audio.previewCue=kind=>creatureCues.has(kind)?audio.preview('voice',kind):Promise.resolve(false);
  audio.cancelPreview=()=>{previewIntent++;previewRequested=false;clearTimeout(previewTimer);if(!active){stopVoices();if(audio.context?.state==='running')audio.context.suspend().catch(()=>{});}};
  audio.preview=async (channel,cueKind)=>{
    if(!Object.hasOwn(buses(),channel))return false;
    const intent=++previewIntent;previewRequested=true;clearTimeout(previewTimer);if(!active){stopAmbience();stopVoices();}
    const ready=await audio.unlock();if(intent!==previewIntent||document.hidden){if(document.hidden)previewRequested=false;if(!active&&!previewRequested&&audio.context?.state==='running')await audio.context.suspend().catch(()=>{});return false;}
    if(!ready||audio.muted||channel==='interface'&&audio.interfaceMuted||channel==='music'&&(audio.streamerMusic||audio.musicPreset==='streamer')){if(!ready)audio.blocked=true;previewRequested=false;if(!active&&audio.context?.state==='running')await audio.context.suspend().catch(()=>{});return false;}
    const bus=buses()[channel];
    if(cueKind)playCue(cueKind,0);
    else if(channel==='ambience')hiss(.45,.1,900,0,0,bus);
    else if(channel==='voice')tone(170,65,.4,.09,'sawtooth',0,0,bus);
    else if(channel==='music')[220,330,440].forEach(f=>tone(f,f,.5,.035,'sine',0,0,bus));
    else tone(channel==='interface'?540:350,channel==='interface'?720:100,.2,.09,'triangle',0,0,bus);
    previewTimer=setTimeout(()=>{if(intent!==previewIntent)return;previewRequested=false;if(!active&&audio.context?.state==='running')audio.context.suspend().catch(()=>{});},cueKind?1750:650);
    return true;
  };
  window.addEventListener('pagehide', () => {
    active = false;
    activation++;
    previewIntent++;
    previewRequested = false;
    clearTimeout(previewTimer);
    clearAmbienceTimers();
    stopAmbience();
    stopVoices();
    audio.releaseAbandonedAmbience();
    if (audio.context) audio.context.close().catch(() => {});
  });
  window.addEventListener('pageshow', () => {
    active = false;
    activation++;
    previewIntent++;
    previewRequested = false;
    clearTimeout(previewTimer);
    clearAmbienceTimers();
    stopAmbience();
    stopVoices();
    audio.releaseAbandonedAmbience();
    if (audio.context?.state === 'closed') {
      audio.context = null;
      audio.voices = 0;
      panners.clear();
    }
  });
  window.addEventListener('popstate', () => {
    if (audio.context?.state === 'closed') {
      audio.context = null;
      audio.voices = 0;
      panners.clear();
    }
  });
  window.RiftAudio = audio;
})();
