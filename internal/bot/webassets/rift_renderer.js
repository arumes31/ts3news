(function () {
  'use strict';
  const root = document.getElementById('rift-app'), canvas = document.getElementById('rift-canvas'), ctx = canvas.getContext('2d');
  const images = {}, effectRows = { slash:0, third_strike:0, finisher_cast:4, ultimate_anticipation:4, hit:0, hit_blade:0, hit_blunt:0, hit_pierce:0, hit_arcane:0, hit_fist:0, hit_ranged:0, fire:1, slam:1, quake:1, ice:2, shield:3, heal:3, block:3, perfect_guard:3, radiant:3, rune:3, void:4, poison:4, ultimate:4, pack:2, pickup:5, clear:5, treasure_escape:5, rare_item:5, rare_discovery:5 };
  const bestiary=window.RiftBestiary,catalogImages={},display=window.RiftDisplay;
  const styles = ['vanguard','berserker','marksman','beastmaster','elementalist','chronomancer','oracle','geomancer','bloodblade','voidwalker','runesmith','alchemist'];
  const foundations = {warrior:'vanguard',ranger:'marksman',arcanist:'elementalist',warden:'oracle',reaver:'bloodblade',artificer:'runesmith'};
  const deaths = new Map();
  let animationTime = 0, decorationTime = 0, motion = 1;
  let previewStyle = 'vanguard';
  let previewLevel = null, transitionAt = -1000, impactAt=-Infinity;
  // Authored atlas panels have slightly different row heights. Crop inside
  // each panel to keep neighboring regions out of the battlefield.
  const regionRows = [0,.179,.363,.559,.755,1];
  let snapshot = null, previous = null, received = 0, camera = 0, seen = 0, runID = '', effects = [], last = 0, footstep = 0;
  const renderer = { reduced: window.matchMedia('(prefers-reduced-motion: reduce)').matches, ready: null, frameCount: 0, rangeSkill: null };
  renderer.setRangeSkill = function(skill){ renderer.rangeSkill = skill; };
  renderer.getRangeSkill = function(){ return renderer.rangeSkill; };
  renderer.build = build => { previewStyle = build.class; };
  renderer.preview = level => { previewLevel = level; };
  function combatTextProperties(e) {
    let color='#fff0bb',label=String(Math.round(e.value||0));
    if(e.kind==='hurt'){color='#ffb2a0';}
    else if(e.kind==='block'||e.kind==='perfect_guard'){color=e.kind==='perfect_guard'?'#ffd700':'#7fd7ff';label=(e.kind==='perfect_guard'?'⭐ Perfect Guard':'🛡️ Guarded')+(Math.round(e.value||0)>0?' -'+Math.round(e.value):'');}
    else if(e.kind==='pickup'){color='#ffe082';label='+'+Math.round(e.value)+' gold';}
    else if(e.kind==='resource'){color='#7ef5d0';label='+'+Math.round(e.value)+' '+(snapshot?.build?.resource||'Charge');}
    else if(e.kind==='heal'){color='#a8f0b0';label='+'+Math.round(e.value)+' HP';}
    else if(e.kind==='barrier'){color='#c6a8f8';label='+'+Math.round(e.value)+' Barrier';}
    else if(e.kind==='treasure_escape'){color='#ffd700';label='💨 Escaped!';}
    else if(e.kind==='rare_item'||e.kind==='rare_discovery'){const r=Math.round(e.value||0);color=r>=4?'#ff9800':r>=3?'#9c27b0':'#2196f3';label=r>=4?'★ Legendary Discovery!':r>=3?'◆ Epic Discovery!':'✨ Rare Discovery!';}
    return {color,label};
  }
  renderer.combatText = combatTextProperties;
  function drawStaticPickup(targetCtx, x, y) {
    targetCtx.save();
    targetCtx.strokeStyle = '#ffe082';
    targetCtx.fillStyle = '#ffe08233';
    targetCtx.lineWidth = 2;
    targetCtx.beginPath();
    targetCtx.arc(x, y, 16, 0, Math.PI * 2);
    targetCtx.fill();
    targetCtx.stroke();
    targetCtx.font = 'bold 11px monospace';
    targetCtx.fillStyle = '#ffe082';
    targetCtx.textAlign = 'center';
    targetCtx.fillText('✓', x, y + 4);
    targetCtx.restore();
  }
  renderer.drawStaticPickup = drawStaticPickup;
  const criticalAtlasKeys = ['area','boss','regions','props','heroesA','heroesB','mobs','items','effects'];
  const atlasProgress = { loaded: 0, total: criticalAtlasKeys.length, ready: false };
  function updateAtlasProgress(loaded, total, status) {
    const el = document.getElementById('rift-atlas-progress');
    if (el) {
      el.dataset.loaded = String(loaded);
      el.dataset.total = String(total);
      if (status === 'error') {
        el.textContent = 'Critical atlases stalled (' + loaded + '/' + total + ')';
      } else if (loaded >= total) {
        el.textContent = 'Critical atlases loaded (' + loaded + '/' + total + ')';
      } else {
        el.textContent = 'Loading critical atlases: ' + loaded + '/' + total;
      }
    }
    try {
      window.dispatchEvent(new CustomEvent('riftatlasprogress', { detail: { loaded, total, status: status || (loaded >= total ? 'ready' : 'loading') } }));
    } catch (_) {}
  }
  updateAtlasProgress(0, criticalAtlasKeys.length);
  renderer.atlasProgress = atlasProgress;
  renderer.getCriticalAtlasKeys = () => criticalAtlasKeys.slice();
  const hazardPatternProfiles = {
    fire: { kind: 'fire', pattern: 'diagonal-stripes', label: 'Diagonal stripes' },
    ice: { kind: 'ice', pattern: 'diamond-grid', label: 'Diamond cross-hatch' },
    poison: { kind: 'poison', pattern: 'polka-dots', label: 'Bubble stippling' },
    thorns: { kind: 'thorns', pattern: 'chevrons', label: 'Chevron teeth' },
    rune: { kind: 'rune', pattern: 'concentric-diamonds', label: 'Concentric diamonds' },
    radiant: { kind: 'radiant', pattern: 'vertical-beams', label: 'Vertical beam stripes' },
    void: { kind: 'void', pattern: 'dashed-scanlines', label: 'Horizontal dashed scanlines' },
  };
  function getHazardPatternInfo(kind) {
    return hazardPatternProfiles[kind] || { kind, pattern: 'diagonal-stripes', label: 'Diagonal stripes' };
  }
  renderer.getHazardPatternInfo = getHazardPatternInfo;
  renderer.getHazardKindsWithPatterns = () => Object.keys(hazardPatternProfiles);
  renderer.drawHazardPattern = drawHazardPattern;
  const projectileShapeProfiles = {
    hostile: { hostility: 'hostile', shape: 'barbed-wedge', label: 'Barbed wedge with rear spurs' },
    friendly: { hostility: 'friendly', shape: 'diamond-crest', label: 'Diamond crest with swept wings' },
  };
  function getProjectileShapeInfo(isEnemy) {
    return isEnemy ? projectileShapeProfiles.hostile : projectileShapeProfiles.friendly;
  }
  renderer.getProjectileShapeInfo = getProjectileShapeInfo;

  function drawProjectileShape(ctx, x, y, isEnemy, vx, vy, kind) {
    const angle = Math.atan2(vy || 0, vx || (isEnemy ? -1 : 1));
    ctx.save();
    ctx.translate(x, y);
    ctx.rotate(angle);
    if (isEnemy) {
      // Sharp barbed wedge with jagged backward spurs
      ctx.beginPath();
      ctx.moveTo(14, 0);
      ctx.lineTo(-6, -7);
      ctx.lineTo(-3, -3);
      ctx.lineTo(-12, -5);
      ctx.lineTo(-8, 0);
      ctx.lineTo(-12, 5);
      ctx.lineTo(-3, 3);
      ctx.lineTo(-6, 7);
      ctx.closePath();
      ctx.fillStyle = '#ff6f4f';
      ctx.fill();
      ctx.strokeStyle = '#ffe4d6';
      ctx.lineWidth = 1.5;
      ctx.stroke();
      ctx.fillStyle = '#fff29c';
      ctx.beginPath();
      ctx.arc(0, 0, 2.5, 0, Math.PI * 2);
      ctx.fill();
    } else {
      // Smooth aerodynamic diamond crest with swept wings
      ctx.beginPath();
      ctx.moveTo(15, 0);
      ctx.lineTo(3, -6);
      ctx.lineTo(-10, -8);
      ctx.lineTo(-5, 0);
      ctx.lineTo(-10, 8);
      ctx.lineTo(3, 6);
      ctx.closePath();
      ctx.fillStyle = '#5eead4';
      ctx.fill();
      ctx.strokeStyle = '#e6fffa';
      ctx.lineWidth = 1.5;
      ctx.stroke();
      ctx.fillStyle = '#ffffff';
      ctx.beginPath();
      ctx.moveTo(5, 0);
      ctx.lineTo(0, -3);
      ctx.lineTo(-5, 0);
      ctx.lineTo(0, 3);
      ctx.closePath();
      ctx.fill();
    }
    ctx.restore();
  }
  renderer.drawProjectileShape = drawProjectileShape;

  function drawHazardPattern(ctx, kind, x, y, w, h, active, color) {
    ctx.save();
    ctx.globalAlpha = active ? 0.75 : 0.4;
    switch (kind) {
      case 'fire': {
        ctx.lineWidth = active ? 2.5 : 1.5;
        ctx.strokeStyle = active ? '#fff0b0' : color;
        ctx.beginPath();
        const step = 12;
        const start = Math.floor((x - h) / step) * step;
        const end = x + w + h;
        for (let px = start; px < end; px += step) {
          ctx.moveTo(px, y + h);
          ctx.lineTo(px + h, y);
        }
        ctx.stroke();
        break;
      }
      case 'ice': {
        ctx.lineWidth = active ? 2 : 1;
        ctx.strokeStyle = active ? '#e6f8ff' : color;
        ctx.beginPath();
        const step = 14;
        const start = Math.floor((x - h) / step) * step;
        const end = x + w + h;
        for (let px = start; px < end; px += step) {
          ctx.moveTo(px, y + h);
          ctx.lineTo(px + h, y);
          ctx.moveTo(px, y);
          ctx.lineTo(px + h, y + h);
        }
        ctx.stroke();
        break;
      }
      case 'poison': {
        ctx.fillStyle = active ? '#f0ffd0' : color;
        const r = active ? 3 : 2;
        const stepX = 14, stepY = 10;
        for (let py = y + 5; py < y + h; py += stepY) {
          const shift = Math.floor((py - y) / stepY) % 2 === 0 ? 0 : stepX / 2;
          for (let px = x + 5 + shift; px < x + w; px += stepX) {
            ctx.beginPath();
            ctx.arc(px, py, r, 0, Math.PI * 2);
            ctx.fill();
          }
        }
        break;
      }
      case 'thorns': {
        ctx.lineWidth = active ? 2.5 : 1.5;
        ctx.strokeStyle = active ? '#f4ffd4' : color;
        ctx.beginPath();
        const toothW = 10, toothH = 6;
        for (let py = y + 6; py < y + h; py += 12) {
          for (let px = x; px < x + w + toothW; px += toothW) {
            ctx.moveTo(px, py + toothH);
            ctx.lineTo(px + toothW / 2, py);
            ctx.lineTo(px + toothW, py + toothH);
          }
        }
        ctx.stroke();
        break;
      }
      case 'rune': {
        ctx.lineWidth = active ? 2 : 1;
        ctx.strokeStyle = active ? '#f6ebff' : color;
        const sz = 8;
        const stepX = 18, stepY = 14;
        for (let py = y + 8; py < y + h; py += stepY) {
          for (let px = x + 10; px < x + w; px += stepX) {
            ctx.strokeRect(px - sz / 2, py - sz / 2, sz, sz);
            ctx.beginPath();
            ctx.moveTo(px, py - sz);
            ctx.lineTo(px + sz, py);
            ctx.lineTo(px, py + sz);
            ctx.lineTo(px - sz, py);
            ctx.closePath();
            ctx.stroke();
          }
        }
        break;
      }
      case 'radiant': {
        ctx.lineWidth = active ? 3 : 1.5;
        ctx.strokeStyle = active ? '#ffffff' : color;
        ctx.beginPath();
        const step = 10;
        for (let px = x + 5; px < x + w; px += step) {
          ctx.moveTo(px, y);
          ctx.lineTo(px, y + h);
        }
        ctx.stroke();
        break;
      }
      case 'void': {
        ctx.lineWidth = active ? 2.5 : 1.5;
        ctx.strokeStyle = active ? '#f2e8ff' : color;
        ctx.setLineDash([8, 6]);
        ctx.beginPath();
        const stepY = 8;
        for (let py = y + 5; py < y + h; py += stepY) {
          ctx.moveTo(x, py);
          ctx.lineTo(x + w, py);
        }
        ctx.stroke();
        ctx.setLineDash([]);
        break;
      }
      default: {
        ctx.lineWidth = active ? 2 : 1;
        ctx.strokeStyle = color;
        ctx.beginPath();
        const step = 12;
        for (let px = x; px < x + w + h; px += step) {
          ctx.moveTo(px, y + h);
          ctx.lineTo(px + h, y);
        }
        ctx.stroke();
        break;
      }
    }
    ctx.restore();
  }
  const baseImages = Promise.all(criticalAtlasKeys.map(key => new Promise((resolve, reject) => {
    const img = new Image();
    img.onload = () => {
      images[key] = img;
      atlasProgress.loaded++;
      if (atlasProgress.loaded >= atlasProgress.total) {
        atlasProgress.ready = true;
      }
      updateAtlasProgress(atlasProgress.loaded, atlasProgress.total);
      resolve();
    };
    img.onerror = () => {
      updateAtlasProgress(atlasProgress.loaded, atlasProgress.total, 'error');
      reject(new Error('Could not load ' + key + ' artwork. Reload to try again.'));
    };
    img.src = key === 'props' ? document.getElementById('rift-props-asset').href : root.dataset[key];
  })));
  renderer.ready=Promise.all([baseImages,...bestiary.assets.map(path=>new Promise((resolve,reject)=>{const img=new Image();img.onload=()=>{catalogImages[path]=img;resolve();};img.onerror=()=>reject(new Error('Could not load Abyss creature art. Reload to try again.'));img.src=bestiary.assetURL(path);} ))]);
  renderer.snapshot = function (run, replay) {
    if (!run) return;
  };
  renderer.feed = function (run, replay) { return renderer.snapshot(run, replay); };
  renderer.renderActor = function (unit, now) { return actor(unit, now || performance.now()); };
  renderer.snapshot = function (run, replay) {
    const changed = runID !== run.id || snapshot && run.counter < snapshot.counter;
    if (changed) { impactAt=-Infinity; runID = run.id; seen = replay ? run.counter : 0; effects = []; previous = null; deaths.clear(); }
    else previous = snapshot;
    if (previous && (previous.room !== run.room || previous.level?.id !== run.level?.id)) { previous = null; effects = []; deaths.clear(); camera=0; transitionAt=animationTime; }
    snapshot = run; received = performance.now();
    [run.player,...run.enemies].forEach(unit=>{if(unit.hp<=0&&!deaths.has(unit.id))deaths.set(unit.id,replay?animationTime-1000:animationTime);});
    (run.events || []).forEach(event => {
      if (event.id <= seen) return;
      seen = event.id;
      if(!replay&&(event.kind==='slam'||event.kind==='third_strike'||event.kind==='ultimate_anticipation'||event.kind==='boss_phase'||(event.kind==='finisher_cast'&&event.value>0)||event.kind==='hurt'&&event.value>0))impactAt=performance.now();
      if (event.kind !== 'area') effects.push({ ...event, started: animationTime });
      const floorMat = run.floor || run.level?.rooms?.[run.room]?.floor || 'stone';
      const extraArg = event.kind === 'hit' ? (run.build?.weapon || run.build?.class || 'blade') : event.value;
      const dx = run.player ? (event.x - run.player.x) : 0;
      const dy = run.player ? ((event.y || run.player.y) - run.player.y) : 0;
      const dist = Math.hypot(dx, dy);
      window.RiftAudio.play(event.kind, dx / 700, extraArg, floorMat, dist);
    });
    if (effects.length > 40) effects = effects.slice(-40);
    window.RiftAudio.area((run.level?.region||0)*3+run.room);
  };
  function sprite(row, col, x, y, size, flip, alpha, atlas = 'heroesA') {
    const img = images[atlas]; if (!img) return;
    ctx.save(); ctx.globalAlpha = alpha === undefined ? 1 : alpha; ctx.translate(Math.round(x),Math.round(y)); ctx.scale(flip < 0 ? -1 : 1,1);
    ctx.drawImage(img,col*img.width/16,row*img.height/6,img.width/16,img.height/6,-size/2,-size*.91,size,size); ctx.restore();
  }
  function fx(row, frame, x, y, size, alpha) {
    const img = images.effects; if (!img) return;
    ctx.save(); ctx.globalAlpha = alpha*(row===5?1:display.effectIntensity); ctx.drawImage(img,frame*img.width/6,row*img.height/6,img.width/6,img.height/6,Math.round(x-size/2),Math.round(y-size/2),size,size); ctx.restore();
  }
  function actor(unit, now) {
    const shared=unit.art_key?bestiary.profile(unit):null;
    const index = Math.max(0,styles.indexOf(foundations[unit.kind] || unit.kind));
    const atlas = unit.id === 'player' ? (index < 6 ? 'heroesA' : 'heroesB') : 'mobs';
    const row = unit.id === 'player' ? index%6 : ({goblin:0,archer:1,knight:2,boss:3,wolf:4,spore:5}[unit.kind] ?? 0);
    const size = unit.kind === 'boss' ? 168 : shared&&['rat','bat','slime','spider','goblin'].includes(shared.rig) ? 80 : unit.kind === 'wolf' ? 63 : 101;
    let x = unit.x, y = unit.y;
    if (previous && snapshot && unit.hp > 0) {
      const old = unit.id === 'player' ? previous.player : previous.enemies.find(e => e.id === unit.id);
      const t = Math.min(1,(now-received)/110);
      if (old) { x = old.x+(x-old.x)*t; y = old.y+(y-old.y)*t; }
    }
    if (unit.hp <= 0) { ctx.fillStyle='#03110b70'; ctx.beginPath(); ctx.ellipse(x-camera,y+2,size*.28,7,0,0,Math.PI*2); ctx.fill(); const fallen=animationTime-(deaths.get(unit.id)??0);if(shared)catalogActor(unit,'defeat',x-camera,y,size,fallen<700?.85:.4);else sprite(row,fallen<320?13:14,x-camera,y,size,unit.facing,fallen<700?.85:.4,atlas);return; }
    let col = renderer.reduced ? 0 : Math.floor(decorationTime/650)%2;
    if (unit.pose === 'run') col = 2+Math.floor(animationTime/105)%4;
    if (unit.pose === 'attack') col = unit.pose_time > .25 ? 8 : unit.pose_time > .12 ? 9 : 10;
    if (unit.pose === 'cast' || unit.pose === 'ultimate_anticipation') col = 11;
    if (unit.pose === 'windup') col = 8;
    if (unit.guard) col = 7;
    if (unit.pose === 'guard_walk') col = Math.floor(animationTime/160)%2 === 0 ? 7 : 3;
    if (unit.pose === 'land' || unit.pose === 'recovery') col = 10;
    if (unit.jump > 0) col = 6;
    if (unit.pose === 'hit' && !unit.guard) col = 12;
    if (unit.knockdown > 0) col = 13;
    if (unit.id === 'player' && snapshot.status === 'cleared' && unit.pose !== 'run') col = 15;
    const jump = unit.jump > 0 ? Math.sin((.65-unit.jump)/.65*Math.PI)*52 : 0;
    const landSquash = unit.pose === 'land' && !renderer.reduced ? 2 : 0;
    const recoverySquash = unit.pose === 'recovery' && !renderer.reduced ? 2 : 0;
    const guardStride = unit.pose === 'guard_walk' && !renderer.reduced ? Math.sin(animationTime/80)*2*motion : 0;
    const ultimateHover = unit.pose === 'ultimate_anticipation' && !renderer.reduced ? Math.sin(animationTime/70)*4 + 7 : 0;
    const recoil = (!renderer.reduced && unit.recoil_x) ? unit.recoil_x : 0;
    if (unit.id === 'player') renderer.lastPlayerRecoil = recoil;
    const drawX = x - camera + recoil;
    if (unit.id !== 'player') {
      const jumpNorm = Math.min(1, Math.max(0, jump / 52));
      const shadowScale = renderer.reduced ? 1.0 : Math.max(0.48, 1.0 - jumpNorm * 0.42);
      const shadowAlpha = renderer.reduced ? 0.38 : Math.max(0.18, 0.44 * (1.0 - jumpNorm * 0.45));
      const shadowRx = size * 0.30 * shadowScale;
      const shadowRy = Math.max(2.5, 6.5 * shadowScale);
      ctx.save();
      ctx.fillStyle = 'rgba(3, 17, 11, ' + shadowAlpha + ')';
      ctx.beginPath();
      ctx.ellipse(drawX, y + 2, shadowRx, shadowRy, 0, 0, Math.PI * 2);
      ctx.fill();
      ctx.restore();
      renderer.lastEnemyJumpShadow = { unitId: unit.id, x: Math.round(drawX), y: Math.round(y + 2), jump: Number(jump.toFixed(2)), scale: Number(shadowScale.toFixed(3)), alpha: Number(shadowAlpha.toFixed(3)), reduced: !!renderer.reduced };
    } else {
      ctx.fillStyle = '#03110b70';
      ctx.beginPath();
      ctx.ellipse(drawX, y + 2, size * 0.28, 7, 0, 0, Math.PI * 2);
      ctx.fill();
    }
    if(shared)catalogActor(unit,unit.pose,drawX,y-jump+landSquash+recoverySquash+guardStride-ultimateHover,size,1);else sprite(row,col,drawX,y-jump+landSquash+recoverySquash+guardStride-ultimateHover,size,unit.facing,1,atlas);
    if (unit.guard || unit.id === 'player' && snapshot.barrier > 0) fx(3,1,drawX,y-size*.4,80,.55);
    if (unit.guard && unit.pose === 'hit') fx(3,2,drawX,y-size*.4,105,.85);
    if (unit.pose === 'stagger') {
      renderer.lastBossStagger = { unitId: unit.id, pose: unit.pose, poseTime: unit.pose_time, x: Math.round(drawX), y: Math.round(y) };
      ctx.save();
      const stars = 3;
      for (let s = 0; s < stars; s++) {
        const starAng = renderer.reduced ? (s * Math.PI * 2 / stars) : ((animationTime / 140) + s * Math.PI * 2 / stars);
        const starX = drawX + Math.cos(starAng) * 22;
        const starY = y - jump - size - 14 + Math.sin(starAng) * 6;
        ctx.fillStyle = '#ffd54f';
        ctx.strokeStyle = '#071813';
        ctx.lineWidth = 1;
        ctx.beginPath();
        ctx.arc(starX, starY, 3, 0, Math.PI * 2);
        ctx.fill();
        ctx.stroke();
      }
      ctx.restore();
    }
    if (unit.id === 'player' && (snapshot.skill_timers?.slowed || 0) > 0) {
      renderer.lastSlowFrost = { x: Math.round(drawX), y: Math.round(y), remaining: snapshot.skill_timers.slowed };
      const frostPulse = renderer.reduced ? 0 : Math.sin(animationTime / 160) * 2;
      ctx.save();
      ctx.strokeStyle = 'rgba(150, 235, 255, 0.7)';
      ctx.lineWidth = 1.5;
      ctx.beginPath();
      ctx.ellipse(drawX, y + 1, 26 + frostPulse, 6, 0, 0, Math.PI * 2);
      ctx.stroke();
      ctx.restore();
    }
    if (unit.id !== 'player' && unit.kind !== 'wolf') {
      if (unit.kind === 'boss') {
        renderer.lastBossPhaseDraw = { unitId: unit.id, phase: unit.phase || 1 };
      }
      if(!display.cleanScreenshot&&display.healthBars){
        ctx.fillStyle='#0a1715dc'; ctx.fillRect(x-camera-24,y-size*.9-8,48,5);
        ctx.fillStyle=unit.kind==='boss'?'#e9a35c':'#bc7055'; ctx.fillRect(x-camera-23,y-size*.9-7,46*unit.hp/unit.max_hp,3);
        ctx.fillStyle='rgba(255,255,255,0.45)';
        ctx.fillRect(Math.round(x-camera-23+46*0.25),Math.round(y-size*.9-7),1,3);
        ctx.fillRect(Math.round(x-camera-23+46*0.50),Math.round(y-size*.9-7),1,3);
        if(unit.kind==='boss'&&unit.phase>=2){
          ctx.strokeStyle = unit.phase>=3 ? '#ff6575' : '#ffb84d';
          ctx.lineWidth = 1.5;
          ctx.strokeRect(x-camera-25,y-size*.9-9,50,7);
        } else if(unit.hp/unit.max_hp<=0.25){
          ctx.strokeStyle='#ffd79e'; ctx.lineWidth=1;
          ctx.strokeRect(x-camera-24.5,y-size*.9-8.5,49,6);
        }
      }
      if(!display.cleanScreenshot&&unit.art_key&&(display.enemyNames==='all'||display.enemyNames==='boss'&&unit.kind==='boss')){const displayName=(unit.kind==='boss'&&unit.name&&unit.name.length>26)?unit.name.slice(0,24)+'…':unit.name;const textY=y-size*.9-18;ctx.font=(10*display.textScale)+'px monospace';ctx.textAlign='center';ctx.fillStyle='#e9efce';ctx.strokeStyle='#0a1715';ctx.lineWidth=3;ctx.strokeText(displayName,x-camera,textY);ctx.fillText(displayName,x-camera,textY);}
      if(!display.cleanScreenshot&&unit.kind==='boss'&&unit.windup>0){const attack=unit.attack_name||(unit.art_key&&(unit.attacks+1)%2===0?'Aimed Volley':'Ground Slam');const attackY=y-size*.9-30;ctx.font='bold '+(10*display.textScale)+'px monospace';ctx.textAlign='center';ctx.fillStyle='#ffe599';ctx.strokeStyle='#0a1715';ctx.lineWidth=3;ctx.strokeText('⚡ '+attack+' ('+unit.windup.toFixed(1)+'s)',x-camera,attackY);ctx.fillText('⚡ '+attack+' ('+unit.windup.toFixed(1)+'s)',x-camera,attackY);}
      if(snapshot.marked===unit.id){
        const targetX = drawX;
        const targetY = y - jump - size * 0.45;
        const pulse = renderer.reduced ? 0 : Math.sin(animationTime / 180);
        renderer.lastMarkedTargetPulse = { unitId: unit.id, x: Math.round(targetX), y: Math.round(targetY), pulse: Number(pulse.toFixed(3)), reduced: !!renderer.reduced };
        ctx.save();
        ctx.strokeStyle = 'rgba(126, 245, 208, ' + (renderer.reduced ? 0.7 : 0.6 + 0.3 * pulse) + ')';
        ctx.lineWidth = 1.5;
        ctx.beginPath();
        ctx.ellipse(targetX, y + 1, size * 0.34 * (renderer.reduced ? 1 : (1 + 0.06 * pulse)), 6, 0, 0, Math.PI * 2);
        ctx.stroke();

        const rx = size * 0.42 * (renderer.reduced ? 1 : (1 + 0.08 * pulse));
        const ry = size * 0.52 * (renderer.reduced ? 1 : (1 + 0.08 * pulse));
        ctx.strokeStyle = 'rgba(126, 245, 208, ' + (renderer.reduced ? 0.75 : 0.65 + 0.25 * pulse) + ')';
        ctx.lineWidth = 2;
        ctx.beginPath();
        ctx.ellipse(targetX, targetY, rx, ry, 0, 0, Math.PI * 2);
        ctx.stroke();

        if(!renderer.reduced){
          const bx = rx * 0.82, by = ry * 0.82;
          const bLen = 6;
          ctx.strokeStyle = '#a6ffe5';
          ctx.lineWidth = 1.5;
          ctx.beginPath();
          ctx.moveTo(targetX - bx, targetY - by + bLen); ctx.lineTo(targetX - bx, targetY - by); ctx.lineTo(targetX - bx + bLen, targetY - by);
          ctx.moveTo(targetX + bx - bLen, targetY - by); ctx.lineTo(targetX + bx, targetY - by); ctx.lineTo(targetX + bx, targetY - by + bLen);
          ctx.moveTo(targetX - bx, targetY + by - bLen); ctx.lineTo(targetX - bx, targetY + by); ctx.lineTo(targetX - bx + bLen, targetY + by);
          ctx.moveTo(targetX + bx - bLen, targetY + by); ctx.lineTo(targetX + bx, targetY + by); ctx.lineTo(targetX + bx, targetY + by - bLen);
          ctx.stroke();
        }

        const caretBob = renderer.reduced ? 0 : Math.sin(animationTime / 140) * 3;
        const caretY = y - jump - size - 12 + caretBob;
        ctx.fillStyle = '#7ef5d0';
        ctx.beginPath();
        ctx.moveTo(targetX, caretY);
        ctx.lineTo(targetX - 5, caretY - 7);
        ctx.lineTo(targetX + 5, caretY - 7);
        ctx.closePath();
        ctx.fill();

        ctx.strokeStyle = '#b4ffeb';
        ctx.lineWidth = 1.2;
        ctx.beginPath();
        ctx.moveTo(targetX, caretY - 14);
        ctx.lineTo(targetX + 3.5, caretY - 10.5);
        ctx.lineTo(targetX, caretY - 7);
        ctx.lineTo(targetX - 3.5, caretY - 10.5);
        ctx.closePath();
        ctx.stroke();

        ctx.restore();
      }
    }
  }
  function catalogActor(unit,pose,x,y,size,alpha){
    const profile=bestiary.profile(unit);
    // Keep the expanded Brawl animations for matching existing species. All
    // other anatomy comes directly from Abyss's shared actor-frame provider.
    const localRow={goblin:0,wolf:4,knight:2}[profile.rig];
    let mapped=pose==='hit'?'hurt':pose==='windup'?'cast':pose==='knockdown'?'defeat':pose==='land'||pose==='guard_walk'||pose==='recovery'?'idle':pose==='ultimate_anticipation'?'cast':pose==='stagger'?'hurt':pose;
    if(localRow!==undefined){
      let col=renderer.reduced?0:Math.floor(decorationTime/650)%2;
      if(pose==='run')col=2+Math.floor(animationTime/105)%4;
      if(pose==='attack')col=unit.pose_time>.25?8:unit.pose_time>.12?9:10;
      if(pose==='windup')col=8;if(pose==='cast'||pose==='ultimate_anticipation')col=11;if(pose==='hit'||pose==='stagger')col=12;if(pose==='knockdown')col=13;if(pose==='defeat')col=14;
      if(pose==='guard_walk')col=Math.floor(animationTime/160)%2===0?7:3;
      if(pose==='land'||pose==='recovery')col=10;
      sprite(localRow,col,x,y,size,unit.facing,alpha,'mobs');
    }else{
      const frame=bestiary.frame(unit,mapped,Math.floor((mapped==='idle'?decorationTime:animationTime)/(pose==='run'?110:200))),img=catalogImages[frame.asset],source=frame.source;
      if(!img||!source)return;
      const stride=((pose==='run'?Math.sin(animationTime/65)*3:pose==='guard_walk'?Math.sin(animationTime/80)*2:0))*(!renderer.reduced?motion:0);
      ctx.save();ctx.globalAlpha=alpha;ctx.translate(Math.round(x),Math.round(y+stride));ctx.scale(unit.facing<0?-1:1,1);
      if(pose==='knockdown')ctx.rotate(-.55);
      if(pose==='stagger'){
        const staggerTremor = !renderer.reduced ? Math.sin(animationTime / 40) * 2.5 : 0;
        ctx.translate(staggerTremor, 0);
        ctx.rotate(-0.18);
      }
      ctx.drawImage(img,source.x*img.width,source.y*img.height,source.width*img.width,source.height*img.height,-size/2,-size*.91,size,size);ctx.restore();
    }
    if(alpha===1&&profile.element!=='physical'){ctx.globalAlpha=.6;ctx.fillStyle=profile.palette[0];ctx.beginPath();ctx.ellipse(x,y+1,size*.28,4,0,0,Math.PI*2);ctx.fill();ctx.globalAlpha=1;}
  }
  function render(now) {
    requestAnimationFrame(render);
    if (!images.area || document.hidden || !ctx || now-last<1000/display.fps-1) return;
    renderer.frameCount++;
    ctx.imageSmoothingEnabled = false;
    const dt = Math.min(.05,(now-last)/1000); last = now;
    motion=renderer.reduced?0:display.motionIntensity;
    if (!snapshot || !snapshot.paused) { animationTime += dt*1000; decorationTime += dt*1000*motion; }
    const wallNow = now; now = animationTime;
    ctx.setTransform(1,0,0,1,0,0);ctx.fillStyle='#091914';ctx.fillRect(0,0,960,540);
    const impactAge=wallNow-impactAt,shake=snapshot&&!snapshot.paused&&motion>0?display.shakeIntensity*motion*4*Math.max(0,1-impactAge/200):0;
    if(shake>0)ctx.translate(Math.sin(impactAge*.19)*shake,Math.cos(impactAge*.23)*shake*.6);
    const targetCamera = snapshot ? Math.max(0,Math.min(640,snapshot.player.x-350)) : 220;
    camera = display.cameraSmooth?camera+(targetCamera-camera)*Math.min(1,dt*8):targetCamera;
    // Slow background parallax retains the full walkable foreground.
    const region = snapshot?.level && ['fighting','cleared'].includes(snapshot.status) ? snapshot.level.region : previewLevel?.region;
    const background = region !== undefined ? images.regions : snapshot?.room===2 ? images.boss : images.area;
    if(region !== undefined){
      const row=Math.floor(region/2), top=regionRows[row], bottom=regionRows[row+1];
      ctx.drawImage(background,region%2*background.width/2+2,top*background.height+2,background.width/2-4,(bottom-top)*background.height-4,-camera*.35,0,1184,540);
    }else ctx.drawImage(background,0,0,background.width,background.height,-camera*.35,0,1184,540);
    if (snapshot && snapshot.room === 1) { ctx.fillStyle='#61532316';ctx.fillRect(0,0,960,540); }
    if (!renderer.reduced && display.particles) {
      for(let i=0;i<Math.round(22*display.particleIntensity);i++) { const x=(i*157+decorationTime*.004*(i%3+1))%1000; const y=80+(i*41)%300+Math.sin(decorationTime*.0005+i)*14*motion; ctx.globalAlpha=.3+Math.sin(decorationTime*.001+i)*.2*motion; ctx.fillStyle=i%3?'#a9ce8c':'#ffd98a';ctx.fillRect(x,y,2,2); }
      ctx.globalAlpha=1;
    }
    if (!snapshot) { const index=Math.max(0,styles.indexOf(foundations[previewStyle]||previewStyle));sprite(index%6,renderer.reduced?0:Math.floor(decorationTime/650)%2,630,400,113,-1,1,index<6?'heroesA':'heroesB');return; }
    const run=snapshot;
    const arena=run.practice?.arena||run.level?.rooms[run.room];
    if(!display.cleanScreenshot&&run.practice&&['movement','jump'].includes(run.practice.mode)){
      ctx.save();ctx.strokeStyle='#e3f9ac';ctx.lineWidth=4;ctx.setLineDash([10,7]);ctx.beginPath();ctx.moveTo(run.practice.goal_x-camera,250);ctx.lineTo(run.practice.goal_x-camera,535);ctx.stroke();ctx.setLineDash([]);ctx.fillStyle='#e3f9ac';ctx.font='bold 14px monospace';ctx.textAlign='center';ctx.fillText('FINISH',run.practice.goal_x-camera,240);ctx.restore();
    }
    (arena?.hazards||[]).forEach(h=>{
      const phase=(run.clock+h.offset)%h.period, warning=phase<1.2, active=phase>=1.2&&phase<1.2+h.duration&&run.status==='fighting';
      const x=h.x-camera,color={fire:'#ff9a52',ice:'#9be5ff',rune:'#d1acff',poison:'#c7ee76',thorns:'#b5d780',radiant:'#d7dfff',void:'#b194ff'}[h.kind]||'#ffbf70';
      ctx.save();ctx.fillStyle=color;ctx.strokeStyle=color;ctx.lineWidth=active?3:1;ctx.globalAlpha=active?.55:warning?.18:.06;ctx.fillRect(x,h.y,h.w,h.h);ctx.globalAlpha=active?1:warning?.7:.2;
      ctx.setLineDash(warning?[5,4]:[]);ctx.strokeRect(x,h.y,h.w,h.h);ctx.setLineDash([]);
      if(display.hazardPatterns!==false&&(warning||active)){ctx.save();ctx.beginPath();ctx.rect(x,h.y,h.w,h.h);ctx.clip();drawHazardPattern(ctx,h.kind,x,h.y,h.w,h.h,active,color);ctx.restore();}
      if(display.hazardContrast&&(warning||active)){ctx.globalAlpha=1;ctx.strokeStyle='#fff8d8';ctx.lineWidth=3;ctx.setLineDash(active?[]:[8,4]);ctx.strokeRect(x-2,h.y-2,h.w+4,h.h+4);ctx.setLineDash([]);}
      if(!display.cleanScreenshot&&display.hazardLabels&&(warning||active)){ctx.font='bold 9px monospace';ctx.textAlign='center';ctx.fillText(active?'JUMP':h.kind.toUpperCase(),x+h.w/2,h.y-5);}
      if(active&&!renderer.reduced)fx(effectRows[h.kind]??3,Math.floor(now/90)%6,x+h.w/2,h.y+h.h/2,h.w,.7);
      ctx.restore();
    });
    run.enemies.forEach(e => {
      if(!display.cleanScreenshot&&e.hp>0 && e.windup>0 && e.kind==='boss') {
        const attackName=e.attack_name||(e.art_key&&(e.attacks+1)%2===0?'Aimed Volley':'Ground Slam');
        ctx.fillStyle='#c8783b55';ctx.strokeStyle='#ffce7d';ctx.lineWidth=2;ctx.beginPath();ctx.ellipse(e.target_x-camera,e.target_y,125,62,0,0,Math.PI*2);ctx.fill();ctx.stroke();
        ctx.fillStyle='#ffe2b0';ctx.font='bold '+(12*display.textScale)+'px monospace';ctx.textAlign='center';ctx.fillText(attackName.toUpperCase()+' · JUMP OR MOVE',e.target_x-camera,e.target_y+4);
      }
    });
    const activeRangeSkill = renderer.rangeSkill || (display.skillRange && run.build?.skills?.[0] ? run.build.skills[0] : null);
    if (!display.cleanScreenshot && activeRangeSkill && run.player && ['fighting','cleared'].includes(run.status)) {
      const p = run.player, px = p.x - camera, py = p.y, ref = activeRangeSkill.reference;
      if (ref) {
        ctx.save();
        if (ref.target === 'area') {
          ctx.fillStyle = '#2dd4bf24';
          ctx.strokeStyle = '#2dd4bf';
          ctx.lineWidth = 2;
          ctx.setLineDash([6, 4]);
          ctx.beginPath();
          ctx.ellipse(px, py, ref.horizontal, ref.depth, 0, 0, Math.PI * 2);
          ctx.fill();
          ctx.stroke();
          ctx.setLineDash([]);
          ctx.font = 'bold ' + (11 * display.textScale) + 'px monospace';
          ctx.textAlign = 'center';
          ctx.fillStyle = '#99f6e4';
          ctx.strokeStyle = '#071813';
          ctx.lineWidth = 3;
          const label = activeRangeSkill.name.toUpperCase() + ' · AREA (' + ref.horizontal + 'h × ' + ref.depth + 'd)';
          ctx.strokeText(label, px, py - ref.depth - 6);
          ctx.fillText(label, px, py - ref.depth - 6);
        } else if (ref.target === 'projectile') {
          const reach = 530;
          const left = p.facing < 0 ? Math.max(0, px - reach) : px;
          const width = p.facing < 0 ? px - left : Math.min(960 - px, reach);
          const top = py - ref.depth, height = ref.depth * 2;
          ctx.fillStyle = '#38bdf822';
          ctx.strokeStyle = '#38bdf8';
          ctx.lineWidth = 2;
          ctx.setLineDash([6, 4]);
          ctx.fillRect(left, top, width, height);
          ctx.strokeRect(left, top, width, height);
          ctx.setLineDash([]);
          ctx.fillStyle = '#38bdf8aa';
          ctx.beginPath();
          const arrowX = p.facing < 0 ? left + 18 : left + width - 18;
          ctx.moveTo(arrowX, py - 8);
          ctx.lineTo(arrowX + (p.facing < 0 ? -12 : 12), py);
          ctx.lineTo(arrowX, py + 8);
          ctx.fill();
          ctx.font = 'bold ' + (11 * display.textScale) + 'px monospace';
          ctx.textAlign = 'center';
          ctx.fillStyle = '#bae6fd';
          ctx.strokeStyle = '#071813';
          ctx.lineWidth = 3;
          const label = activeRangeSkill.name.toUpperCase() + ' · PROJECTILE LANE (±' + ref.depth + 'd)';
          ctx.strokeText(label, left + width / 2, top - 6);
          ctx.fillText(label, left + width / 2, top - 6);
        } else if (ref.target === 'self') {
          ctx.fillStyle = '#c084fc28';
          ctx.strokeStyle = '#c084fc';
          ctx.lineWidth = 2;
          ctx.setLineDash([4, 4]);
          ctx.beginPath();
          ctx.ellipse(px, py - 20, 48, 48, 0, 0, Math.PI * 2);
          ctx.fill();
          ctx.stroke();
          ctx.setLineDash([]);
          ctx.font = 'bold ' + (11 * display.textScale) + 'px monospace';
          ctx.textAlign = 'center';
          ctx.fillStyle = '#e9d5ff';
          ctx.strokeStyle = '#071813';
          ctx.lineWidth = 3;
          const label = activeRangeSkill.name.toUpperCase() + ' · SELF (' + (ref.barrier ? 'BARRIER' : 'HEAL') + ')';
          ctx.strokeText(label, px, py - 72);
          ctx.fillText(label, px, py - 72);
        }
        ctx.restore();
      }
    }
    const activeAreaEffect = window.RiftHUD?.detectPlayerAreaEffects ? window.RiftHUD.detectPlayerAreaEffects(run) : null;
    if (!display.cleanScreenshot && activeAreaEffect && run.player && ['fighting','cleared'].includes(run.status)) {
      const p = run.player, px = p.x - camera, py = p.y - (p.jump || 0) * 120;
      ctx.save();
      const color = activeAreaEffect.state === 'active' ? '#ff7a45' : activeAreaEffect.state === 'warning' ? '#ffd066' : activeAreaEffect.state === 'evading' ? '#99f6e4' : '#d8b4fe';
      ctx.strokeStyle = color;
      ctx.lineWidth = activeAreaEffect.state === 'active' ? 3 : 2;
      if (activeAreaEffect.state === 'warning') ctx.setLineDash([4, 4]);
      ctx.beginPath();
      ctx.ellipse(px, py, 36, 16, 0, 0, Math.PI * 2);
      ctx.stroke();
      ctx.setLineDash([]);
      ctx.font = 'bold ' + (10 * display.textScale) + 'px monospace';
      ctx.fillStyle = color;
      ctx.strokeStyle = '#071813';
      ctx.lineWidth = 3;
      ctx.textAlign = 'center';
      const label = activeAreaEffect.name.toUpperCase() + (activeAreaEffect.state === 'evading' ? ' (EVADING)' : '');
      ctx.strokeText(label, px, py + 20);
      ctx.fillText(label, px, py + 20);
      ctx.restore();
    }
    if(display.enemyIndicators && !display.cleanScreenshot && run.status==='fighting'){
      const offscreen=run.enemies.filter(e=>e.hp>0&&(e.x-camera<0||e.x-camera>960));
      renderer.lastOffscreen=offscreen;
      offscreen.forEach(e=>{
        const left=e.x-camera<0,isBoss=e.kind==='boss';
        const ix=left?22:938,iy=Math.max(50,Math.min(490,e.y));
        const color=isBoss?'#ffbe60':'#ff6b6b',edgeColor=isBoss?'#ffe89e':'#ffa3a3';
        ctx.save();
        ctx.fillStyle='#071813e0';ctx.strokeStyle=edgeColor;ctx.lineWidth=2;
        ctx.beginPath();
        if(left){
          ctx.moveTo(ix+14,iy-12);ctx.lineTo(ix-10,iy);ctx.lineTo(ix+14,iy+12);ctx.closePath();
        }else{
          ctx.moveTo(ix-14,iy-12);ctx.lineTo(ix+10,iy);ctx.lineTo(ix-14,iy+12);ctx.closePath();
        }
        ctx.fill();ctx.stroke();
        ctx.fillStyle=color;
        ctx.beginPath();
        if(left){
          ctx.moveTo(ix+11,iy-8);ctx.lineTo(ix-5,iy);ctx.lineTo(ix+11,iy+8);ctx.closePath();
        }else{
          ctx.moveTo(ix-11,iy-8);ctx.lineTo(ix+5,iy);ctx.lineTo(ix-11,iy+8);ctx.closePath();
        }
        ctx.fill();
        if(isBoss){
          ctx.font='bold 9px monospace';ctx.textAlign=left?'left':'right';ctx.fillStyle='#ffe599';ctx.strokeStyle='#071813';ctx.lineWidth=3;
          ctx.strokeText('BOSS',left?ix+18:ix-18,iy+3);ctx.fillText('BOSS',left?ix+18:ix-18,iy+3);
        }else if(e.windup>0){
          ctx.font='bold 11px monospace';ctx.textAlign='center';ctx.fillStyle='#fff0aa';ctx.strokeStyle='#071813';ctx.lineWidth=3;
          ctx.strokeText('!',left?ix+4:ix-4,iy+4);ctx.fillText('!',left?ix+4:ix-4,iy+4);
        }
        ctx.restore();
      });
    }else{
      renderer.lastOffscreen=[];
    }
    (run.drops||[]).forEach(drop => {
      if(drop.collected||drop.banked)return;
      const y=drop.y-8+(renderer.reduced||!display.lootMotion?0:Math.sin(decorationTime/200)*3*motion),x=drop.x-camera;
      const legendary=window.RiftLoot.legendary(drop);
      if(legendary){ctx.strokeStyle='#ffc66d';ctx.lineWidth=3;ctx.beginPath();ctx.moveTo(x,y-25);ctx.lineTo(x+25,y);ctx.lineTo(x,y+25);ctx.lineTo(x-25,y);ctx.closePath();ctx.stroke();}
      if(display.lootSparkle)fx(5,0,x,y,34,.7);
      const icon=drop.gear?window.RiftLoot.icon(drop.gear.Slot):8,img=images.items,size=drop.gear?36:25;
      ctx.drawImage(img,icon%4*img.width/4,Math.floor(icon/4)*img.height/4,img.width/4,img.height/4,x-size/2,y-size/2,size,size);
    });
    for(const label of (!display.cleanScreenshot&&display.optionalCombatText)?window.RiftLoot.floorLabels(run.drops||[],camera):[]){
      if(label.moved){ctx.strokeStyle='#80927b';ctx.lineWidth=1;ctx.beginPath();ctx.moveTo(label.drop.x-camera,label.drop.y-26);ctx.lineTo(label.x,label.y+17);ctx.stroke();}
      ctx.font='10px monospace';ctx.textAlign='center';ctx.fillStyle='#081914';ctx.fillRect(label.x-57,label.y,114,17);ctx.fillStyle=label.legendary?'#ffc66d':'#d7ecbb';ctx.fillText(label.text,label.x,label.y+12);
    }
    const units=[...run.enemies,run.player];
    if(run.build?.class==='beastmaster'&&run.player.hp>0){for(let i=0;i<Math.min(3,run.build?.pets||0);i++)units.push({id:'pet'+i,kind:'wolf',x:run.player.x-run.player.facing*(55+i*36),y:run.player.y+22+i*8,hp:1,max_hp:1,facing:run.player.facing,pose:run.player.pose==='cast'?'cast':['run','guard_walk'].includes(run.player.pose)?'run':'idle',jump:0});}
    (arena?.obstacles||[]).forEach(o=>units.push({y:o.y+o.h,cover:o}));
    units.sort((a,b)=>a.y-b.y).forEach(unit=>{
      if(!unit.cover){actor(unit,wallNow);return;}
      const o=unit.cover,img=images.props,index=[0,1,2,3,4,5,6,3,3,7][run.level?.region||0],sw=img.width/4,sh=img.height/2;
      ctx.fillStyle='#03110a70';ctx.beginPath();ctx.ellipse(o.x+o.w/2-camera,o.y+o.h-3,o.w*.58,9,0,0,Math.PI*2);ctx.fill();
      // Each sprite's base lies at 90% of its atlas cell. Align it with
      // the collision footprint so jumping and circling cover read clearly.
      const width=o.w+14,height=o.h+38;
      ctx.drawImage(img,index%4*sw,Math.floor(index/4)*sh,sw,sh,o.x-7-camera,o.y+o.h-height*.9,width,height);
    });
    run.projectiles.forEach(p=>{
      if(display.projectileShapes!==false){
        if(p.kind==='arrow'){
          drawProjectileShape(ctx,p.x-camera,p.y-30,Boolean(p.enemy),p.vx,p.vy,p.kind);
        }else if(p.kind==='pack'){
          sprite(4,2+Math.floor(now/70)%4,p.x-camera,p.y,70,p.vx,.85,'mobs');
          drawProjectileShape(ctx,p.x-camera,p.y-28,Boolean(p.enemy),p.vx,p.vy,p.kind);
        }else{
          fx(effectRows[p.kind]??1,Math.floor(now/80)%3,p.x-camera,p.y-28,58,.95);
          drawProjectileShape(ctx,p.x-camera,p.y-28,Boolean(p.enemy),p.vx,p.vy,p.kind);
        }
      }else{
        if(p.kind==='arrow'){ctx.fillStyle='#d8b3e9';ctx.fillRect(p.x-camera-12,p.y-30,25,3);}
        else if(p.kind==='pack')sprite(4,2+Math.floor(now/70)%4,p.x-camera,p.y,70,p.vx,.85,'mobs');
        else fx(effectRows[p.kind]??1,Math.floor(now/80)%3,p.x-camera,p.y-28,58,.95);
      }
    });
    effects=effects.filter(e=>now-e.started<750);
    effects.forEach(e=>{
      const age=(now-e.started)/750;
      if(e.kind==='third_strike'){
        renderer.lastThirdStrikeAccent = { x: e.x, y: e.y, age, started: e.started };
        if(!renderer.reduced){
          const screenX = e.x - camera, screenY = e.y;
          ctx.save();
          const ringRadius = 12 + age * 65;
          const fade = Math.max(0, 1 - age * 2.2);
          if (fade > 0) {
            ctx.strokeStyle = 'rgba(255, 238, 140, ' + fade + ')';
            ctx.lineWidth = Math.max(1, 3.5 * fade);
            ctx.beginPath();
            ctx.ellipse(screenX, screenY, ringRadius, ringRadius * 0.55, 0, 0, Math.PI * 2);
            ctx.stroke();

            const spireLen = (1 - age * 1.8) * 32;
            if (spireLen > 0) {
              ctx.strokeStyle = 'rgba(255, 255, 220, ' + (fade * 0.9) + ')';
              ctx.lineWidth = Math.max(1, 2 * fade);
              ctx.beginPath();
              ctx.moveTo(screenX - spireLen, screenY - spireLen * 0.4);
              ctx.lineTo(screenX + spireLen, screenY + spireLen * 0.4);
              ctx.moveTo(screenX - spireLen * 0.7, screenY + spireLen * 0.5);
              ctx.lineTo(screenX + spireLen * 0.7, screenY - spireLen * 0.5);
              ctx.stroke();
            }

            const flareSize = Math.max(1, 14 * (1 - age * 2.5));
            if (flareSize > 0) {
              ctx.fillStyle = 'rgba(255, 245, 180, ' + (fade * 0.8) + ')';
              ctx.beginPath();
              ctx.arc(screenX, screenY, flareSize, 0, Math.PI * 2);
              ctx.fill();
            }
          }
          ctx.restore();
          fx(0, Math.min(5, Math.floor(age * 8)), screenX, screenY, 145, Math.max(0, 1 - age * 1.5));
        }
      }
      if(e.kind==='finisher_cast'){
        renderer.lastFinisherCastAccent = { x: e.x, y: e.y, charges: e.value, age, started: e.started };
        if(!renderer.reduced){
          const screenX = e.x - camera, screenY = e.y + 35;
          const charges = Math.max(0, Math.round(e.value || 0));
          const fade = Math.max(0, 1 - age * 2.0);
          if(fade > 0){
            ctx.save();
            const bloomRadius = 24 + charges * 8 + age * 20;
            ctx.fillStyle = charges >= 3 ? 'rgba(255, 215, 0, ' + (fade * 0.35) + ')' : 'rgba(126, 245, 208, ' + (fade * 0.3) + ')';
            ctx.beginPath();
            ctx.ellipse(screenX, screenY, bloomRadius, bloomRadius * 0.45, 0, 0, Math.PI * 2);
            ctx.fill();

            ctx.strokeStyle = charges >= 3 ? 'rgba(255, 235, 120, ' + (fade * 0.8) + ')' : 'rgba(150, 255, 230, ' + (fade * 0.75) + ')';
            ctx.lineWidth = Math.max(1, (2 + charges * 0.5) * fade);
            ctx.beginPath();
            ctx.ellipse(screenX, screenY, bloomRadius, bloomRadius * 0.45, 0, 0, Math.PI * 2);
            ctx.stroke();

            const rings = 2 + charges;
            for(let r = 0; r < rings; r++){
              const ringProgress = (age * 3.5 + r / rings) % 1;
              const ringY = screenY - ringProgress * 60;
              const ringW = Math.max(8, (18 + charges * 4) * (1 - ringProgress * 0.4));
              const ringFade = fade * Math.sin(ringProgress * Math.PI);
              if(ringFade > 0){
                ctx.strokeStyle = charges >= 3 ? 'rgba(255, 240, 160, ' + (ringFade * 0.85) + ')' : 'rgba(180, 255, 240, ' + (ringFade * 0.8) + ')';
                ctx.lineWidth = Math.max(1, 2.5 * ringFade);
                ctx.beginPath();
                ctx.ellipse(screenX, ringY, ringW, ringW * 0.35, 0, 0, Math.PI * 2);
                ctx.stroke();
              }
            }

            const motes = 3 + charges * 3;
            for(let m = 0; m < motes; m++){
              const motePhase = (age * 2.8 + m / motes) % 1;
              const mx = screenX + Math.sin(m * 2.4 + age * 6) * (16 + charges * 4);
              const my = screenY - motePhase * 68;
              const mFade = fade * Math.sin(motePhase * Math.PI);
              if(mFade > 0){
                ctx.fillStyle = charges >= 3 ? 'rgba(255, 245, 180, ' + mFade + ')' : 'rgba(200, 255, 245, ' + mFade + ')';
                ctx.fillRect(mx - 1.5, my - 1.5, 3, 3);
              }
            }
            ctx.restore();
          }
        }
      }
      if(e.kind==='ultimate_anticipation'){
        renderer.lastUltimateAnticipation = { x: e.x, y: e.y, age, started: e.started };
        if(!renderer.reduced){
          const screenX = e.x - camera, screenY = e.y + 35;
          const fade = Math.max(0, 1 - age * 1.8);
          if(fade > 0){
            ctx.save();
            const arms = 4;
            const vortexR = Math.max(14, (1 - age * 0.7) * 72);
            for(let a = 0; a < arms; a++){
              const baseAngle = a * (Math.PI * 2 / arms) + age * 8;
              ctx.beginPath();
              for(let step = 0; step < 16; step++){
                const t = step / 15;
                const r = t * vortexR;
                const angle = baseAngle + t * 2.2;
                const px = screenX + Math.cos(angle) * r;
                const py = (screenY - 25) + Math.sin(angle) * (r * 0.45);
                if(step === 0) ctx.moveTo(px, py); else ctx.lineTo(px, py);
              }
              ctx.strokeStyle = 'rgba(180, 140, 255, ' + (fade * 0.75) + ')';
              ctx.lineWidth = Math.max(1, 2.5 * fade);
              ctx.stroke();
            }

            const coreRadius = Math.max(6, 16 * Math.sin(age * Math.PI));
            ctx.fillStyle = 'rgba(255, 235, 180, ' + (fade * 0.7) + ')';
            ctx.beginPath();
            ctx.arc(screenX, screenY - 25, coreRadius, 0, Math.PI * 2);
            ctx.fill();

            ctx.strokeStyle = 'rgba(210, 170, 255, ' + (fade * 0.7) + ')';
            ctx.lineWidth = Math.max(1, 2 * fade);
            ctx.beginPath();
            ctx.ellipse(screenX, screenY, 40 * (1 - age * 0.2), 16 * (1 - age * 0.2), 0, 0, Math.PI * 2);
            ctx.stroke();
            ctx.restore();
          }
        }
      }
      if(e.kind==='heavy_recovery'){
        renderer.lastHeavyRecovery = { x: e.x, y: e.y, age, started: e.started, value: e.value };
        if(!renderer.reduced){
          const screenX = e.x - camera, screenY = e.y + 35;
          const fade = Math.max(0, 1 - age * 3.5);
          if(fade > 0){
            ctx.save();
            const ringRadius = 16 + age * 65;
            ctx.strokeStyle = 'rgba(180, 225, 255, ' + (fade * 0.7) + ')';
            ctx.lineWidth = Math.max(1, 2 * fade);
            ctx.beginPath();
            ctx.ellipse(screenX, screenY, ringRadius, ringRadius * 0.38, 0, 0, Math.PI * 2);
            ctx.stroke();

            const puffCount = 4;
            for(let p = 0; p < puffCount; p++){
              const dir = (p % 2 === 0 ? 1 : -1);
              const progress = (age * 3.0 + p * 0.08) % 1;
              const puffX = screenX + dir * (12 + progress * 28 + (p > 1 ? 8 : 0));
              const puffY = screenY - 2 - Math.sin(progress * Math.PI) * 6;
              const puffR = 2.5 + progress * 3;
              ctx.fillStyle = 'rgba(215, 235, 255, ' + (fade * (1 - progress) * 0.55) + ')';
              ctx.beginPath();
              ctx.arc(puffX, puffY, puffR, 0, Math.PI * 2);
              ctx.fill();
            }
            ctx.restore();
          }
        }
      }
      if(e.kind==='shield_absorb'){
        renderer.lastShieldShimmer = { x: e.x, y: e.y, age, started: e.started, value: e.value };
        const screenX = e.x - camera, screenY = e.y + 30;
        const fade = Math.max(0, 1 - age * 2.8);
        if(fade > 0){
          ctx.save();
          const radius = 38 + (renderer.reduced ? 0 : Math.sin(age * 25) * 3);
          ctx.strokeStyle = 'rgba(196, 144, 255, ' + (fade * 0.85) + ')';
          ctx.lineWidth = Math.max(1, 3 * fade);
          ctx.beginPath();
          ctx.ellipse(screenX, screenY - 5, radius, radius * 1.15, 0, 0, Math.PI * 2);
          ctx.stroke();

          ctx.strokeStyle = 'rgba(125, 249, 255, ' + (fade * 0.9) + ')';
          ctx.lineWidth = Math.max(1, 1.5 * fade);
          const innerR = radius * 0.85;
          ctx.beginPath();
          ctx.ellipse(screenX, screenY - 5, innerR, innerR * 1.1, 0, 0, Math.PI * 2);
          ctx.stroke();

          if(!renderer.reduced){
            const facets = 6;
            for(let f = 0; f < facets; f++){
              const ang = f * (Math.PI / 3) + age * 2;
              const fx1 = screenX + Math.cos(ang) * (radius * 0.6);
              const fy1 = (screenY - 5) + Math.sin(ang) * (radius * 0.7);
              const fx2 = screenX + Math.cos(ang) * radius;
              const fy2 = (screenY - 5) + Math.sin(ang) * (radius * 1.15);
              ctx.strokeStyle = 'rgba(255, 255, 255, ' + (fade * 0.65) + ')';
              ctx.lineWidth = 1;
              ctx.beginPath();
              ctx.moveTo(fx1, fy1);
              ctx.lineTo(fx2, fy2);
              ctx.stroke();
            }
            for(let s = 0; s < 5; s++){
              const sparkAngle = s * 1.25 + age * 5;
              const sparkDist = radius + age * 22;
              const sx = screenX + Math.cos(sparkAngle) * sparkDist;
              const sy = (screenY - 5) + Math.sin(sparkAngle) * (sparkDist * 0.8);
              ctx.fillStyle = 'rgba(230, 210, 255, ' + (fade * 0.75) + ')';
              ctx.beginPath();
              ctx.arc(sx, sy, 1.8 * fade, 0, Math.PI * 2);
              ctx.fill();
            }
          }
          ctx.restore();
        }
      }
      if(e.kind==='mark_target'){
        renderer.lastMarkTargetEvent = { x: e.x, y: e.y, age, started: e.started };
        if(!renderer.reduced){
          const screenX = e.x - camera, screenY = e.y;
          const fade = Math.max(0, 1 - age * 2.5);
          if(fade > 0){
            ctx.save();
            const lockR = 42 * (1 - age * 0.45);
            ctx.strokeStyle = 'rgba(126, 245, 208, ' + (fade * 0.85) + ')';
            ctx.lineWidth = Math.max(1, 2 * fade);
            ctx.beginPath();
            ctx.arc(screenX, screenY, lockR, 0, Math.PI * 2);
            ctx.stroke();

            const tickLen = 7;
            ctx.strokeStyle = 'rgba(166, 255, 229, ' + (fade * 0.9) + ')';
            ctx.lineWidth = Math.max(1, 1.5 * fade);
            ctx.beginPath();
            ctx.moveTo(screenX - lockR - tickLen, screenY); ctx.lineTo(screenX - lockR + tickLen, screenY);
            ctx.moveTo(screenX + lockR - tickLen, screenY); ctx.lineTo(screenX + lockR + tickLen, screenY);
            ctx.moveTo(screenX, screenY - lockR - tickLen); ctx.lineTo(screenX, screenY - lockR + tickLen);
            ctx.moveTo(screenX, screenY + lockR - tickLen); ctx.lineTo(screenX, screenY + lockR + tickLen);
            ctx.stroke();
            ctx.restore();
          }
        }
      }
      if(e.kind==='thaw'){
        renderer.lastThawEffect = { x: e.x, y: e.y, age, started: e.started };
        const screenX = e.x - camera, screenY = e.y;
        const fade = Math.max(0, 1 - age * 2.6);
        if(fade > 0){
          ctx.save();
          const ringR = 20 + age * 55;
          ctx.strokeStyle = 'rgba(195, 245, 255, ' + (fade * 0.8) + ')';
          ctx.lineWidth = Math.max(1, 2.5 * fade);
          ctx.beginPath();
          ctx.ellipse(screenX, screenY + 5, ringR, ringR * 0.42, 0, 0, Math.PI * 2);
          ctx.stroke();

          if(!renderer.reduced){
            const shards = 8;
            for(let s = 0; s < shards; s++){
              const ang = s * (Math.PI * 2 / shards) + 0.35;
              const dist = 12 + age * 68;
              const sx = screenX + Math.cos(ang) * dist;
              const sy = screenY + Math.sin(ang) * (dist * 0.6) + (age * age * 40);
              const shardSize = Math.max(1, (1 - age * 1.6) * 3.5);
              ctx.fillStyle = s % 2 === 0 ? 'rgba(230, 250, 255, ' + (fade * 0.9) + ')' : 'rgba(145, 235, 255, ' + (fade * 0.85) + ')';
              ctx.beginPath();
              ctx.moveTo(sx, sy - shardSize * 1.5);
              ctx.lineTo(sx + shardSize, sy);
              ctx.lineTo(sx, sy + shardSize * 1.5);
              ctx.lineTo(sx - shardSize, sy);
              ctx.closePath();
              ctx.fill();
            }

            const motes = 4;
            for(let m = 0; m < motes; m++){
              const progress = (age * 3.2 + m * 0.25) % 1;
              const mx = screenX + Math.sin(m * 2.1 + age * 5) * 16;
              const my = screenY - progress * 40;
              const mFade = fade * Math.sin(progress * Math.PI);
              if(mFade > 0){
                ctx.fillStyle = 'rgba(220, 245, 255, ' + (mFade * 0.6) + ')';
                ctx.beginPath();
                ctx.arc(mx, my, 2 * (1 - progress * 0.5), 0, Math.PI * 2);
                ctx.fill();
              }
            }
          }
          ctx.restore();
        }
      }
      if(e.kind==='boss_phase'){
        const phase = Math.round(e.value) || 2;
        renderer.lastBossPhaseEvent = { x: e.x, y: e.y, phase, age, started: e.started, reduced: !!renderer.reduced };
        const screenX = e.x - camera, screenY = e.y;
        const fade = Math.max(0, 1 - age * 2.0);
        if(fade > 0){
          ctx.save();
          const isCritical = phase >= 3;
          const primaryColor = isCritical ? 'rgba(255, 65, 85, ' : 'rgba(255, 175, 55, ';
          const secondaryColor = isCritical ? 'rgba(195, 75, 255, ' : 'rgba(255, 235, 120, ';

          // Outward expanding blast ring
          const blastR = 24 + age * (isCritical ? 140 : 110);
          ctx.strokeStyle = primaryColor + (fade * 0.9) + ')';
          ctx.lineWidth = Math.max(1, (isCritical ? 4.5 : 3.5) * fade);
          ctx.beginPath();
          ctx.ellipse(screenX, screenY + 15, blastR, blastR * 0.48, 0, 0, Math.PI * 2);
          ctx.stroke();

          // Inner crisp energy ring
          const innerR = 12 + age * (isCritical ? 85 : 70);
          ctx.strokeStyle = secondaryColor + (fade * 0.95) + ')';
          ctx.lineWidth = Math.max(1, 2.5 * fade);
          ctx.beginPath();
          ctx.ellipse(screenX, screenY + 15, innerR, innerR * 0.48, 0, 0, Math.PI * 2);
          ctx.stroke();

          // Phase text banner overhead
          const bannerY = screenY - 50 - (renderer.reduced ? 0 : age * 16);
          const bannerFade = Math.max(0, 1 - age * 1.8);
          if(bannerFade > 0){
            const label = isCritical ? '⚡ PHASE III — CRITICAL' : '⚡ PHASE II — ENRAGED';
            ctx.font = 'bold ' + Math.round(12 * display.textScale) + 'px monospace';
            ctx.textAlign = 'center';
            ctx.strokeStyle = '#0a1715';
            ctx.lineWidth = 3.5;
            ctx.strokeText(label, screenX, bannerY);
            ctx.fillStyle = isCritical ? '#ff6b81' : '#ffd060';
            ctx.fillText(label, screenX, bannerY);
          }

          if(!renderer.reduced){
            // Corona flare spikes
            const spikes = isCritical ? 12 : 8;
            for(let s = 0; s < spikes; s++){
              const ang = s * (Math.PI * 2 / spikes) + age * 2.5;
              const innerDist = 18 + age * 28;
              const outerDist = innerDist + 16 * (1 - age * 0.8);
              const sx1 = screenX + Math.cos(ang) * innerDist;
              const sy1 = (screenY + 15) + Math.sin(ang) * (innerDist * 0.48);
              const sx2 = screenX + Math.cos(ang) * outerDist;
              const sy2 = (screenY + 15) + Math.sin(ang) * (outerDist * 0.48);
              ctx.strokeStyle = (s % 2 === 0 ? primaryColor : secondaryColor) + (fade * 0.85) + ')';
              ctx.lineWidth = Math.max(1, 2 * fade);
              ctx.beginPath();
              ctx.moveTo(sx1, sy1);
              ctx.lineTo(sx2, sy2);
              ctx.stroke();
            }

            // Erupting fiery / abyssal spark motes
            const sparks = isCritical ? 10 : 6;
            for(let m = 0; m < sparks; m++){
              const progress = (age * 3.4 + m / sparks) % 1;
              const sparkAngle = m * 1.6 + age * 4;
              const sparkDist = 15 + progress * (isCritical ? 75 : 55);
              const mx = screenX + Math.cos(sparkAngle) * sparkDist;
              const my = (screenY + 15) + Math.sin(sparkAngle) * (sparkDist * 0.45) - progress * 32;
              const mFade = fade * Math.sin(progress * Math.PI);
              if(mFade > 0){
                ctx.fillStyle = (m % 2 === 0 ? primaryColor : secondaryColor) + mFade + ')';
                ctx.beginPath();
                ctx.arc(mx, my, 2.2 * (1 - progress * 0.4), 0, Math.PI * 2);
                ctx.fill();
              }
            }
          }
          ctx.restore();
        }
      }
      if(effectRows[e.kind]!==undefined && e.kind!=='third_strike' && e.kind!=='finisher_cast' && e.kind!=='ultimate_anticipation' && e.kind!=='heavy_recovery' && e.kind!=='shield_absorb' && e.kind!=='mark_target' && e.kind!=='thaw' && e.kind!=='boss_stagger' && e.kind!=='boss_phase' && (!renderer.reduced && (e.kind!=='pickup'||display.lootSparkle)))fx(effectRows[e.kind],Math.min(5,Math.floor(age*6)),e.x-camera,e.y,['slam','quake','ultimate'].includes(e.kind)?240:95,1-age*.5);
      if(e.kind==='pickup' && (renderer.reduced || !display.lootSparkle))drawStaticPickup(ctx,e.x-camera,e.y);
      if(!display.cleanScreenshot && (e.value>0 || e.kind==='block' || e.kind==='perfect_guard' || e.kind==='treasure_escape' || e.kind==='rare_item' || e.kind==='rare_discovery') && e.kind!=='area' && !e.kind.endsWith('_hurt') && e.kind!=='slash' && e.kind!=='third_strike' && e.kind!=='finisher_cast' && e.kind!=='ultimate_anticipation' && e.kind!=='heavy_recovery' && e.kind!=='shield_absorb' && e.kind!=='mark_target' && e.kind!=='thaw' && e.kind!=='boss_stagger' && e.kind!=='boss_phase' && (['pickup','resource','heal','barrier','treasure_escape','rare_item','rare_discovery'].includes(e.kind)?display.optionalCombatText:display.damageNumbers)){
        ctx.font='bold '+(13*display.textScale)+'px monospace';
        ctx.textAlign='center';
        const {color,label}=combatTextProperties(e);
        ctx.fillStyle=color;ctx.strokeStyle='#14221d';ctx.lineWidth=3;
        const drift = (renderer.reduced || !display.damageMotion) ? 0 : age*38*motion;
        ctx.strokeText(label,e.x-camera,e.y-drift);
        ctx.fillText(label,e.x-camera,e.y-drift);
      }
    });
    const isMovingFootstep = (run.player.pose === 'run' && now - footstep > 320) || (run.player.pose === 'guard_walk' && now - footstep > 460);
    if(run.status==='fighting' && !run.paused && isMovingFootstep && run.player.jump===0){const floorMat=run.floor||run.level?.rooms?.[run.room]?.floor||'stone';if(window.RiftAudio.step)window.RiftAudio.step(floorMat,0);else window.RiftAudio.play('step',0);footstep=now;}
    window.RiftAudio.tick();
    if(now-transitionAt<500&&!renderer.reduced){ctx.fillStyle='#091914';ctx.globalAlpha=Math.max(0,.65*(1-(now-transitionAt)/500))*display.flashIntensity;ctx.fillRect(0,0,960,540);ctx.globalAlpha=1;}
  }
  renderer.ready.then(()=>requestAnimationFrame(render)).catch(()=>{});
  window.RiftRenderer=renderer;
})();
