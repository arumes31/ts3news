/* Original procedural arcade sound design. No downloads or third-party samples. */
(function () {
  'use strict';
  function setting(key, fallback) { try { const value = JSON.parse(localStorage.getItem('riftAudio:' + key)); return value === null ? fallback : value; } catch (_) { return fallback; } }
  const audio = { context: null, muted: !!setting('muted', false), effects: Number(setting('effects', .65)), ambience: Number(setting('ambience', .35)), played: 0, voices: 0 };
  let master, sfx, ambient, noise, ambientNodes = [], active = false, room = -1, nextBird = 0;
  const clamp = value => Math.max(0, Math.min(1, Number(value) || 0));
  function save(key, value) { try { localStorage.setItem('riftAudio:' + key, JSON.stringify(value)); } catch (_) {} }
  function busGain(bus, value) { if (bus) bus.gain.setTargetAtTime(value, audio.context.currentTime, .03); }
  audio.unlock = async function () {
    try {
      if (!audio.context) {
        const AC = window.AudioContext || window.webkitAudioContext;
        if (!AC) return false;
        const c = audio.context = new AC();
        master = c.createGain(); sfx = c.createGain(); ambient = c.createGain();
        const limiter = c.createDynamicsCompressor(); limiter.threshold.value = -16; limiter.ratio.value = 8;
        sfx.connect(master); ambient.connect(master); master.connect(limiter); limiter.connect(c.destination);
        master.gain.value = audio.muted ? 0 : .6; sfx.gain.value = clamp(audio.effects); ambient.gain.value = clamp(audio.ambience);
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
    stereo.pan.value = Math.max(-.8, Math.min(.8, pan || 0));
    source.connect(gain); gain.connect(stereo); stereo.connect(bus || sfx); audio.voices++;
    source.onended = () => { source.disconnect(); gain.disconnect(); stereo.disconnect(); audio.voices--; };
    source.start(start); source.stop(start + duration + .02);
  }
  function hiss(duration, volume, cutoff, pan, delay) {
    const c = audio.context;
    if (!c || c.state !== 'running' || audio.voices >= 40) return;
    const start = c.currentTime + (delay || 0), source = c.createBufferSource(), gain = c.createGain(), filter = c.createBiquadFilter(), stereo = c.createStereoPanner();
    source.buffer = noise; filter.type = 'lowpass'; filter.frequency.value = cutoff;
    gain.gain.setValueAtTime(volume, start); gain.gain.exponentialRampToValueAtTime(.0001, start + duration);
    stereo.pan.value = Math.max(-.8, Math.min(.8, pan || 0)); source.connect(filter); filter.connect(gain); gain.connect(stereo); stereo.connect(sfx); audio.voices++;
    source.onended = () => { source.disconnect(); filter.disconnect(); gain.disconnect(); stereo.disconnect(); audio.voices--; };
    source.start(start); source.stop(start + duration + .02);
  }
  audio.play = function (kind, pan) {
    if (!audio.context || audio.context.state !== 'running' || audio.muted || !active) return;
    audio.played++;
    const t = (f, end, d, v, wave, delay) => tone(f, end, d, v, wave, delay, pan);
    switch (kind) {
      case 'step': hiss(.065, .07, 650, pan); break;
      case 'jump': t(160, 480, .15, .09, 'triangle'); hiss(.09, .04, 1200, pan); break;
      case 'slash': hiss(.14, .15, 3800, pan); t(350, 100, .12, .05, 'sawtooth'); break;
      case 'hit': hiss(.1, .2, 1700, pan); t(110, 48, .15, .18, 'triangle'); break;
      case 'hurt': t(170, 65, .2, .1, 'sawtooth'); hiss(.12, .12, 950, pan); break;
      case 'block': t(940, 760, .2, .08, 'square'); t(1510, 1200, .12, .04, 'sine'); hiss(.04, .09, 7000, pan); break;
      case 'fire': hiss(.5, .18, 1500, pan); t(200, 45, .45, .1, 'sawtooth'); break;
      case 'ice': [880,1320,1760].forEach((f,i) => t(f, f*.8, .25, .045, 'sine', i*.045)); hiss(.1,.055,7000,pan); break;
      case 'void': t(240, 38, .6, .13, 'sawtooth'); t(243, 42, .55, .07, 'sine'); break;
      case 'void_cost': t(80,40,.18,.08,'triangle'); break;
      case 'rune': [294,440,587].forEach((f,i)=>t(f,f*1.5,.3,.055,'triangle',i*.06)); hiss(.18,.06,4000,pan); break;
      case 'poison': [180,270,140].forEach((f,i)=>t(f,f*.5,.14,.08,'sine',i*.1)); hiss(.4,.08,750,pan); break;
      case 'radiant': [523,784,1047].forEach((f,i)=>t(f,f,.55,.05,'sine',i*.06)); break;
      case 'pack': t(320,620,.35,.075,'sine'); t(620,300,.6,.06,'sine',.3); hiss(.18,.05,850,pan); break;
      case 'quake': hiss(.85,.22,600,pan); t(90,25,.7,.2,'triangle'); break;
      case 'ultimate': [196,294,392,588].forEach((f,i)=>t(f,f*2,.75,.07,'sawtooth',i*.07)); hiss(.8,.2,2200,pan); break;
      case 'knockdown': hiss(.2,.16,500,pan); t(85,32,.2,.12,'triangle'); break;
      case 'shield': case 'heal': [330,440,660].forEach((f,i) => t(f,f*1.05,.45,.06,'sine',i*.07)); break;
      case 'arrow': hiss(.12,.1,6000,pan); t(700,300,.08,.025,'triangle'); break;
      case 'goblin_attack': t(390,190,.19,.06,'sawtooth'); hiss(.08,.08,1600,pan); break;
      case 'knight_attack': hiss(.25,.2,900,pan); t(140,55,.22,.16,'triangle'); break;
      case 'goblin_death': t(440,80,.4,.075,'sawtooth'); break;
      case 'treasure_attack': t(560,240,.13,.07,'triangle'); break;
      case 'treasure_death': [660,990,1320].forEach((f,i)=>t(f,f*1.2,.23,.06,'triangle',i*.1)); break;
      case 'archer_death': hiss(.4,.16,4000,pan); t(600,180,.28,.04,'square'); break;
      case 'knight_death': hiss(.7,.24,1000,pan); t(100,30,.6,.18,'triangle'); break;
      case 'boss_roar': t(85,45,.9,.18,'sawtooth'); t(88,41,.8,.08,'sawtooth'); hiss(.7,.15,700,pan); break;
      case 'slam': hiss(.65,.3,1100,pan); t(100,28,.5,.3,'sine'); t(140,40,.4,.08,'triangle',.08); break;
      case 'boss_death': hiss(1.5,.25,950,pan); [110,82,55].forEach((f,i) => t(f,28,.7,.1,'sawtooth',i*.2)); break;
      case 'pickup': [660,990,1320].forEach((f,i) => t(f,f,.15,.045,'triangle',i*.04)); break;
      case 'clear': case 'bank': [392,494,587,784].forEach((f,i) => t(f,f,.4,.07,'triangle',i*.12)); break;
      case 'defeat': [294,247,196,147].forEach((f,i) => t(f,f*.95,.65,.07,'triangle',i*.16)); break;
      case 'area': t(110,165,.8,.05,'sine'); t(220,247,.9,.04,'sine',.1); break;
      case 'ui': t(540,720,.07,.035,'triangle'); break;
      default: break;
    }
  };
  function stopAmbience() { ambientNodes.forEach(node => { try { node.stop(); } catch (_) {} node.disconnect(); }); ambientNodes = []; }
  audio.area = function (index) {
    if (room === index && ambientNodes.length) return;
    room = index; stopAmbience();
    const c = audio.context; if (!c) return;
    const wind = c.createBufferSource(), filter = c.createBiquadFilter(), gain = c.createGain();
    const region=Math.floor(index/3), tier=index%3;
    wind.buffer = noise; wind.loop = true; filter.type = 'lowpass'; filter.frequency.value = [460,780,1100,640,350,260,500,390,180,220][region]||460; gain.gain.value = .14;
    wind.connect(filter); filter.connect(gain); gain.connect(ambient); wind.onended = () => { filter.disconnect(); gain.disconnect(); }; wind.start(); ambientNodes.push(wind);
    const root=[130.81,73.42,146.83,82.41,98,65.41,87.31,110,61.74,55][region]||130.81;
    [root,root*1.5,root*2].map(f=>f*(tier===2?.75:tier===1?.9:1)).forEach(f => {
      const osc = c.createOscillator(), level = c.createGain(); osc.type = 'sine'; osc.frequency.value = f; level.gain.value = .017;
      osc.connect(level); level.connect(ambient); osc.onended = () => level.disconnect(); osc.start(); ambientNodes.push(osc);
    });
    nextBird = c.currentTime + 2;
  };
  audio.tick = function () {
    if (!active || !audio.context || audio.context.state !== 'running') return;
    const c = audio.context;
    if (c.currentTime > nextBird) {
      if (room < 2) { tone(1800,2400,.13,.024,'sine',0,-.6,ambient); tone(2100,1600,.16,.02,'sine',.2,.5,ambient); }
      else tone(62,46,.8,.045,'sine',0,.5,ambient);
      nextBird = c.currentTime + 3 + Math.random()*4;
    }
  };
  audio.setActive = async function (value, index) {
    active = value;
    if (value) { if (await audio.unlock()) audio.area(index || 0); }
    else if (audio.context && audio.context.state === 'running') { try { await audio.context.suspend(); } catch (_) {} }
  };
  audio.set = function (key, value) {
    audio[key] = key === 'muted' ? !!value : clamp(value); save(key, audio[key]);
    if (audio.context) { busGain(master, audio.muted ? 0 : .6); busGain(sfx, audio.effects); busGain(ambient, audio.ambience); }
  };
  window.addEventListener('pagehide', () => { active = false; stopAmbience(); if (audio.context) audio.context.close().catch(() => {}); });
  window.RiftAudio = audio;
})();
