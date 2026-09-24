(function () {
  'use strict';
  const root = document.getElementById('rift-app'), canvas = document.getElementById('rift-canvas'), ctx = canvas.getContext('2d');
  const images = {}, effectRows = { slash:0, third_strike:0, finisher_cast:4, ultimate_anticipation:4, hit:0, hit_blade:0, hit_blunt:0, hit_pierce:0, hit_arcane:0, hit_fist:0, hit_ranged:0, fire:1, slam:1, quake:1, ice:2, shield:3, heal:3, block:3, perfect_guard:3, radiant:3, rune:3, void:4, poison:4, ultimate:4, pack:2, sigil_pickup:3, beacon_captured:3, beacons_complete:5, spirit_arrived:3, ritual_interrupt:4, ritual_pulse:4, ritual_complete:3, projectile_impact:0, cover_hit:0, cover_break:0, lane_hurt:1, lane_lost:4, lanes_protected:3, rune_correct:3, rune_wrong:4, rune_gate_open:5, lantern_hurt:1, lantern_extinguished:4, lantern_protected:3, companion_freed:3, rescue_complete:5, cage_break:0, guardians_defeated:3, guardian_unlinked:4, collapse_hit:1, collapse_escaped:3, totem_break:4, generator_break:2, generator_shutdown:3, relic_pickup:3, relic_delivered:5, pickup:5, clear:5, treasure_escape:5, rare_item:5, rare_discovery:5 };
  const bestiary=window.RiftBestiary,catalogImages={},display=window.RiftDisplay;
  const styles = ['vanguard','berserker','marksman','beastmaster','elementalist','chronomancer','oracle','geomancer','bloodblade','voidwalker','runesmith','alchemist'];
  const foundations = {warrior:'vanguard',ranger:'marksman',arcanist:'elementalist',warden:'oracle',reaver:'bloodblade',artificer:'runesmith'};
  const deaths = new Map();
  const decalColors={fire:'#b86a40',ice:'#91cbd8',poison:'#92ad54',void:'#9170b7',radiant:'#d8ca85',rune:'#aa92c8'};
  let decals=[];
  let animationTime = 0, decorationTime = 0, motion = 1, cameraRecovering = false;
  let previewStyle = 'vanguard';
  let previewLevel = null, transitionAt = -1000, impactAt=-Infinity;
  // Authored atlas panels have slightly different row heights. Crop inside
  // each panel to keep neighboring regions out of the battlefield.
  const regionRows = [0,.179,.363,.559,.755,1];
  let snapshot = null, previous = null, received = 0, camera = 0, seen = 0, runID = '', effects = [], last = 0, footstep = 0;
  const renderer = { reduced: window.matchMedia('(prefers-reduced-motion: reduce)').matches, ready: null, frameCount: 0, rangeSkill: null };
  // Include nearby boss sprite extents without moving the player out of view.
  function cameraFrame(run){
    if(!run)return {target:220,min:0,max:640,bosses:[]};
    const arena=run.practice?.arena||run.level?.rooms?.[run.room],lead=arena?.camera_lead||350;
    const base=Math.max(0,Math.min(640,run.player.x-lead));
    const nearby=run.enemies.filter(actor=>actor.kind==='boss'&&actor.hp>0&&Math.abs(actor.x-run.player.x)<=600).sort((a,b)=>Math.abs(a.x-run.player.x)-Math.abs(b.x-run.player.x));
    let left=run.player.x-64,right=run.player.x+64;const bosses=[];
    for(const boss of nearby){
      const nextLeft=Math.min(left,boss.x-104),nextRight=Math.max(right,boss.x+104);
      if(nextRight-nextLeft>960)continue;
      left=nextLeft;right=nextRight;bosses.push(boss.id);
    }
    if(!bosses.length){
      // A 96-unit tracking band absorbs short reversals, including at room edges.
      // Paused snapshots retain explicit framing for previews and saved resumes.
      const target=run.paused?base:Math.max(0,Math.min(640,Math.max(run.player.x-lead-48,Math.min(run.player.x-lead+48,camera))));
      return {target,min:0,max:640,bosses};
    }
    const low=Math.max(-104,right-960),high=Math.min(744,left);
    return {target:Math.max(low,Math.min(high,base)),min:low,max:high,bosses};
  }
  renderer.atlasDiagnostics=new URLSearchParams(location.search).get('riftAtlasDebug')==='1'?{frames:0,errors:[]}:null;
  if(renderer.atlasDiagnostics)renderer.checkAtlasBounds=function(img,sx,sy,sw,sh){
    const valid=[sx,sy,sw,sh].every(Number.isFinite)&&sx>=0&&sy>=0&&sw>0&&sh>0&&sx+sw<=img.width+.001&&sy+sh<=img.height+.001;
    if(!valid){
      const errors=renderer.atlasDiagnostics.errors;
      errors.push({image:img.src||'(unloaded)',source:[sx,sy,sw,sh],size:[img.width,img.height]});
      if(errors.length>32)errors.shift();
    }
    return valid;
  };
  renderer.setRangeSkill = function(skill){ renderer.rangeSkill = skill; };
  renderer.getRangeSkill = function(){ return renderer.rangeSkill; };
  renderer.build = build => { previewStyle = build.class; };
  renderer.preview = level => { previewLevel = level; };
  function combatTextProperties(e) {
    let color='#fff0bb',label=String(Math.round(e.value||0));
    if(e.kind==='hurt'){color='#ffb2a0';}
    else if(e.kind==='block'||e.kind==='perfect_guard'){color=e.kind==='perfect_guard'?'#ffd700':'#7fd7ff';label=(e.kind==='perfect_guard'?'⭐ Perfect Guard':'🛡️ Guarded')+(Math.round(e.value||0)>0?' -'+Math.round(e.value):'');}
    else if(e.kind==='beacon_captured'){color='#b9eaff';label='Beacon '+Math.round(e.value)+' / 3';}
    else if(e.kind==='sigil_pickup'){color='#9df6d3';label='Sigil '+Math.round(e.value)+' / 3';}
    else if(e.kind==='pickup'){color='#ffe082';label='+'+Math.round(e.value)+' gold';}
    else if(e.kind==='elemental_reaction'){color='#c9b8ff';label='REACTION ×1.2';}
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
  const criticalAtlasKeys = ['area','boss','regions','props','heroesA','heroesB','mobs','items','effects','sigil','totem','relic','generator','spirit','cage','lantern','terrainCover','platformSurface'];
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
    spikes: { kind: 'spikes', pattern: 'triangle-studs', label: 'Triangular studs' },
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

  function projectileAngle(vx,vy,isEnemy){return vx||vy?Math.atan2(vy||0,vx||0):isEnemy?Math.PI:0;}
  function drawProjectileShape(ctx, x, y, isEnemy, vx, vy, kind) {
    const angle = projectileAngle(vx,vy,isEnemy);
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
      case 'spikes': {
        ctx.strokeStyle=color;ctx.lineWidth=1.5;
        for(let py=y+8;py<y+h;py+=14)for(let px=x+8;px<x+w;px+=16){ctx.beginPath();ctx.moveTo(px-4,py+3);ctx.lineTo(px,py-4);ctx.lineTo(px+4,py+3);ctx.closePath();ctx.stroke();}
        break;
      }
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
  const hazardLegend=document.querySelector('#rift-hazard-legend dl');
  if(hazardLegend){
    for(const profile of Object.values(hazardPatternProfiles)){
      const name=document.createElement('dt'),sample=document.createElement('canvas'),description=document.createElement('dd');
      sample.width=84;sample.height=36;sample.dataset.kind=profile.kind;sample.setAttribute('aria-hidden','true');
      const sampleContext=sample.getContext('2d');sampleContext.fillStyle='#000000';sampleContext.fillRect(0,0,84,36);
      sampleContext.save();sampleContext.beginPath();sampleContext.rect(0,0,84,36);sampleContext.clip();drawHazardPattern(sampleContext,profile.kind,0,0,84,36,false,'#ffffff');sampleContext.restore();
      name.append(sample,document.createTextNode(profile.kind.charAt(0).toUpperCase()+profile.kind.slice(1)));const effects={ice:'Slowing still lasts 1.4 seconds.',poison:'Slowing still lasts 1.4 seconds.',thorns:'Slowing still lasts 1.4 seconds.',void:'The pull still moves you toward the zone center.'};
      description.textContent=profile.label+'. Guard reduces damage. '+(effects[profile.kind]||'No additional contact effect.');
      hazardLegend.append(name,description);
    }
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
    const entering=!replay&&(!snapshot||runID!==run.id||snapshot.room!==run.room||snapshot.level?.id!==run.level?.id);
    const changed = runID !== run.id || snapshot && run.counter < snapshot.counter;
    if (changed) { cameraRecovering=false; impactAt=-Infinity; runID = run.id; seen = replay ? run.counter : 0; effects = []; decals=[]; previous = null; deaths.clear(); }
    else previous = snapshot;
    if (previous && (previous.room !== run.room || previous.level?.id !== run.level?.id)) { previous = null; effects = []; decals=[]; deaths.clear(); camera=0; cameraRecovering=false; }
    if(entering)transitionAt=animationTime;else if(changed)transitionAt=-1000;
    snapshot = run; received = performance.now();
    if(entering&&run.status==='fighting'&&!run.practice&&run.player.hp>0){
      const flourish=entryFlourishes[run.build?.class];
      if(flourish){effects.push({kind:'class_entry',subclass:run.build.class,x:run.player.x,y:run.player.y-(run.player.elevation||0),started:animationTime});renderer.classEntryCount=(renderer.classEntryCount||0)+1;window.RiftAudio.play(flourish.sound,0);}
    }
    if(run.paused)camera=cameraFrame(run).target;
    [run.player,...run.enemies].forEach(unit=>{if(unit.hp<=0&&!deaths.has(unit.id))deaths.set(unit.id,replay?animationTime-1000:animationTime);});
    (run.events || []).forEach(event => {
      if (event.id <= seen) return;
      seen = event.id;
      if(!replay&&(event.kind==='slam'||event.kind==='third_strike'||event.kind==='ultimate_anticipation'||event.kind==='boss_phase'||(event.kind==='finisher_cast'&&event.value>0)||event.kind==='hurt'&&event.value>0))impactAt=performance.now();
      if(!replay&&event.value>0&&decalColors[event.kind])decals.push({kind:event.kind,x:event.x,y:event.y+30-(event.elevation||0),started:animationTime});
      if (event.kind !== 'area') effects.push({ ...event, y:event.y-(event.elevation||0), visualFacing: run.player.facing, started: animationTime });
      const floorMat = run.floor || run.level?.rooms?.[run.room]?.floor || 'stone';
      const extraArg = event.kind === 'hit' ? (run.build?.weapon || run.build?.class || 'blade') : event.value;
      const dx = run.player ? (event.x - run.player.x) : 0;
      const dy = run.player ? ((event.y || run.player.y) - run.player.y) : 0;
      const dist = Math.hypot(dx, dy);
      if(event.kind!=='projectile_expire')window.RiftAudio.play(event.kind, dx / 700, extraArg, floorMat, dist);
    });
    if(decals.length>40)decals=decals.slice(-40);
    if (effects.length > 40) effects = effects.slice(-40);
    window.RiftAudio.area((run.level?.region||0)*3+run.room);
  };
  function drawAtlas(img,sx,sy,sw,sh,dx,dy,dw,dh){
    const diagnostics=renderer.atlasDiagnostics;
    const valid=!diagnostics||renderer.checkAtlasBounds(img,sx,sy,sw,sh);
    ctx.drawImage(img,sx,sy,sw,sh,dx,dy,dw,dh);
    if(diagnostics){
      diagnostics.frames++;
      ctx.save();ctx.globalAlpha=1;ctx.strokeStyle=valid?'#55e7e2':'#ff4fc3';ctx.lineWidth=1;ctx.setLineDash([]);ctx.strokeRect(dx,dy,dw,dh);ctx.restore();
    }
  }
  const entryFlourishes={
    vanguard:{row:3,points:4,sound:'shield'},berserker:{row:0,points:3,sound:'slash'},
    marksman:{row:0,points:4,sound:'arrow'},beastmaster:{row:3,points:3,sound:'pack'},
    elementalist:{row:2,points:6,sound:'fire'},chronomancer:{row:4,points:8,sound:'ice'},
    oracle:{row:5,points:6,sound:'heal'},geomancer:{row:1,points:4,sound:'quake'},
    bloodblade:{row:0,points:5,sound:'slash'},voidwalker:{row:4,points:3,sound:'void'},
    runesmith:{row:4,points:4,sound:'rune'},alchemist:{row:2,points:5,sound:'poison'}
  };
  const victoryStances={
    vanguard:{name:'Shield salute',frame:7,angle:0,lift:0,color:'#b8d9ff'},
    berserker:{name:'Battle triumph',frame:15,angle:-.08,lift:0,color:'#ffb08d'},
    marksman:{name:'Archer salute',frame:8,angle:-.04,lift:0,color:'#cdeaa0'},
    beastmaster:{name:'Pack salute',frame:15,angle:.06,lift:0,color:'#dac28e'},
    elementalist:{name:'Arcane ascent',frame:11,angle:0,lift:7,color:'#cbb5ff'},
    chronomancer:{name:'Time suspended',frame:11,angle:-.06,lift:5,color:'#9ce4e3'},
    oracle:{name:'Grace blessing',frame:15,angle:0,lift:4,color:'#fff0b0'},
    geomancer:{name:'Stone resolve',frame:7,angle:.04,lift:0,color:'#d8bf94'},
    bloodblade:{name:'Crimson salute',frame:8,angle:.08,lift:0,color:'#ffa3b4'},
    voidwalker:{name:'Void ascension',frame:11,angle:.06,lift:9,color:'#baa1ff'},
    runesmith:{name:'Rune salute',frame:7,angle:-.06,lift:0,color:'#ffd391'},
    alchemist:{name:'Mixture toast',frame:11,angle:-.1,lift:0,color:'#a9e7b4'}
  };
  function sprite(row, col, x, y, size, flip, alpha, atlas = 'heroesA') {
    const img = images[atlas]; if (!img) return;
    ctx.save(); ctx.globalAlpha = alpha === undefined ? 1 : alpha; ctx.translate(Math.round(x),Math.round(y)); ctx.scale(flip < 0 ? -1 : 1,1);
    drawAtlas(img,col*img.width/16,row*img.height/6,img.width/16,img.height/6,-size/2,-size*.91,size,size); ctx.restore();
  }
  function fx(row, frame, x, y, size, alpha) {
    const img = images.effects; if (!img) return;
    ctx.save(); ctx.globalAlpha = alpha*(row===5?1:display.effectIntensity); drawAtlas(img,frame*img.width/6,row*img.height/6,img.width/6,img.height/6,Math.round(x-size/2),Math.round(y-size/2),size,size); ctx.restore();
  }
  // A separate canvas keeps reference playback out of expedition state and audio.
  renderer.drawSkillPreview = function(target,skill,build,elapsed){
    const context=target.getContext('2d'),still=renderer.reduced||display.motionIntensity===0;
    const age=still?.35:Math.min(1,Math.max(0,elapsed/750)),frame=still?2:Math.min(5,Math.floor(age*6));
    const index=Math.max(0,styles.indexOf(foundations[build.class]||build.class)),hero=images[index<6?'heroesA':'heroesB'];
    context.clearRect(0,0,target.width,target.height);context.fillStyle='#101d24';context.fillRect(0,0,target.width,target.height);
    context.strokeStyle='#486068';context.beginPath();context.moveTo(12,132);context.lineTo(308,132);context.stroke();
    const column=skill.kind==='slash'?8+(still?1:Math.min(2,Math.floor(age*3))):11;
    if(hero)context.drawImage(hero,column*hero.width/16,(index%6)*hero.height/6,hero.width/16,hero.height/6,24,42,96,96);
    const row=effectRows[skill.kind],self=['heal','shield'].includes(skill.kind),x=self?72:190;
    context.save();context.globalAlpha=display.effectIntensity*(still?1:1-age*.5);
    if(skill.kind==='arrow'){
      const arrowX=still?190:120+age*150;context.strokeStyle='#f5e8b9';context.lineWidth=3;
      context.beginPath();context.moveTo(arrowX-24,88);context.lineTo(arrowX,88);context.lineTo(arrowX-8,82);context.moveTo(arrowX,88);context.lineTo(arrowX-8,94);context.stroke();
    }else if(row!==undefined&&images.effects){
      const img=images.effects,size=['slam','quake','ultimate'].includes(skill.kind)?150:95;
      context.drawImage(img,frame*img.width/6,row*img.height/6,img.width/6,img.height/6,x-size/2,88-size/2,size,size);
    }
    context.restore();target.dataset.frame=String(frame);target.dataset.kind=skill.kind;target.dataset.reduced=String(still);
    return still;
  };
  function surfaceHeight(x,y,run=snapshot){
    const arena=run?.practice?.arena||run?.level?.rooms?.[run.room];let height=0;
    for(const p of arena?.platforms||[]){const distance=Math.min(x-p.x,p.x+p.w-x,y-p.y,p.y+p.h-y);height=Math.max(height,p.rise*Math.max(0,Math.min(1,distance/p.ramp)));}return height;
  }
  function foregroundCoverOpacity(x,y,width,height,base){
    const p=snapshot?.player;
    if(!p||p.y>base)return 1;
    const jump=p.jump>0?Math.sin((.65-p.jump)/.65*Math.PI)*52:0;
    const feet=p.y-(p.elevation??surfaceHeight(p.x,p.y))-jump;
    const overlapX=Math.min(x+width,p.x+32)-Math.max(x,p.x-32);
    const overlapY=Math.min(y+height,feet)-Math.max(y,feet-90);
    // Spatial falloff avoids a hard opacity toggle as the player skirts an edge.
    return 1-.65*Math.max(0,Math.min(1,overlapX/24,overlapY/24));
  }
  function fadedCoverFootprint(o,opacity,tall){
    if(opacity>=1)return;
    ctx.save();ctx.strokeStyle='#a5d6c1';ctx.globalAlpha=1-opacity;ctx.lineWidth=2;ctx.setLineDash(tall?[]:[5,4]);
    ctx.strokeRect(o.x-camera,o.y,o.w,o.h);ctx.restore();
  }
  let interactionPrompts = null;
  // Capture drawing state now; draw instructions after world sprites and effects.
  function interactionPrompt(text,x,y,backplate=false){
    if(display.cleanScreenshot)return;
    if(!interactionPrompts){ctx.strokeText(text,x,y);ctx.fillText(text,x,y);return;}
    interactionPrompts.push({text,x,y,backplate,transform:ctx.getTransform(),font:ctx.font,align:ctx.textAlign,baseline:ctx.textBaseline,fill:ctx.fillStyle,stroke:ctx.strokeStyle,width:ctx.lineWidth,alpha:ctx.globalAlpha});
  }
  function drawInteractionPrompts(){
    for(const p of interactionPrompts||[]){
      ctx.save();ctx.setTransform(p.transform);ctx.font=p.font;ctx.textAlign=p.align;ctx.textBaseline=p.baseline;ctx.fillStyle=p.fill;ctx.strokeStyle=p.stroke;ctx.lineWidth=p.width;ctx.globalAlpha=p.alpha;
      if(p.backplate){
        const metrics=ctx.measureText(p.text),ascent=metrics.actualBoundingBoxAscent||10,descent=metrics.actualBoundingBoxDescent||2;
        const left=p.align==='center'?p.x-metrics.width/2:p.align==='right'?p.x-metrics.width:p.x;
        ctx.fillStyle='#071b16';ctx.fillRect(left-5,p.y-ascent-3,metrics.width+10,ascent+descent+6);ctx.fillStyle=p.fill;
      }
      ctx.strokeText(p.text,p.x,p.y);ctx.fillText(p.text,p.x,p.y);ctx.restore();
    }
    interactionPrompts=null;
  }
  function actor(unit, now) {
    const worldY=unit.y;
    unit={...unit,y:unit.y-(unit.elevation??surfaceHeight(unit.x,unit.y))};
    if(unit.kind==='spirit'&&unit.id==='escort-spirit'){
      const goal=snapshot.room_objective,old=previous?.room_objective?.escort,t=snapshot.paused?1:Math.max(0,Math.min(1,(now-received)/110));
      const x=(old?old.x+(unit.x-old.x)*t:unit.x)-camera,y=unit.y,bob=renderer.reduced?0:Math.sin(decorationTime/260)*3*motion;
      ctx.save();ctx.strokeStyle=goal.contested?'#ffd18a66':'#b5f5dc55';ctx.lineWidth=1;
      if(!goal.complete){ctx.setLineDash([4,6]);ctx.beginPath();ctx.ellipse(x,y,150,90,0,0,Math.PI*2);ctx.stroke();ctx.setLineDash([]);}
      ctx.drawImage(images.spirit,x-43,y-104+bob,86,108);
      if(!display.cleanScreenshot){ctx.font='bold 11px monospace';ctx.textAlign='center';ctx.fillStyle=goal.contested?'#ffdb9a':'#d5ffea';ctx.strokeStyle='#10251d';ctx.lineWidth=4;const label=!['fighting','cleared'].includes(snapshot.status)?'ESCORT ENDED':goal.complete?'SPIRIT SAFE':goal.contested?'CLEAR NEARBY ENEMIES':goal.escort_moving?'FOLLOWING YOU':'STAY NEAR THE SPIRIT';interactionPrompt(label,x,y-110);}
      ctx.restore();return;
    }
    if(unit.kind==='lantern'&&['ward-lantern','upper-ward','lower-ward'].includes(unit.id)){
      const goal=snapshot.room_objective,x=unit.x-camera,y=unit.y,dead=unit.hp<=0,lane=goal.lanes?.find(l=>l.ward.id===unit.id),contested=lane?lane.contested:goal.contested,title=lane?unit.name.toUpperCase():'LANTERN';
      ctx.save();
      if(dead)ctx.filter='grayscale(1) brightness(.35)';
      ctx.drawImage(images.lantern,x-42,y-108,84,112);ctx.filter='none';
      if(!dead){ctx.globalAlpha=(renderer.reduced ? .22 : .2+.04*Math.sin(decorationTime/240)*motion)*unit.hp/unit.max_hp;ctx.fillStyle='#ffda86';ctx.beginPath();ctx.ellipse(x,y-55,22,28,0,0,Math.PI*2);ctx.fill();ctx.globalAlpha=1;}
      if(!display.cleanScreenshot){ctx.fillStyle='#281f16';ctx.fillRect(x-30,y-119,60,6);ctx.fillStyle=contested?'#ffab83':'#ffe29a';ctx.fillRect(x-30,y-119,60*unit.hp/unit.max_hp,6);ctx.font='bold 10px monospace';ctx.textAlign='center';ctx.fillStyle='#ffe6b5';ctx.strokeStyle='#21190e';ctx.lineWidth=3;const label=dead?title+' LOST':goal.complete?title+' PROTECTED':title+' '+Math.ceil(unit.hp)+'%';interactionPrompt(label,x,y-126);}
      ctx.restore();return;
    }
    if(unit.kind==='cage'){
      const captive=snapshot.room_objective?.captives?.find(c=>c.cage_id===unit.id);if(!captive)return;
      const x=unit.x-camera,y=unit.y,age=captive.freed?Math.max(0,snapshot.clock-captive.freed_at):0;
      if(captive.freed&&age>=3)return;
      ctx.save();
      if(captive.freed)ctx.globalAlpha=Math.max(0,1-age/3);
      const rise=captive.freed&&!renderer.reduced?age*30*motion:0;
      ctx.drawImage(images.spirit,x-27,y-76-rise,54,70);
      if(!captive.freed){
        const shake=unit.pose==='hit'&&unit.pose_time>0&&!renderer.reduced?Math.sin(animationTime/25)*2*motion:0;
        ctx.drawImage(images.cage,x-37+shake,y-105,74,111);
      }
      if(!display.cleanScreenshot){
        ctx.font='bold 10px monospace';ctx.textAlign='center';ctx.fillStyle='#d5ffea';ctx.strokeStyle='#10251d';ctx.lineWidth=3;const label=captive.freed?'FREE · '+captive.name:captive.name+' · BREAK CAGE';interactionPrompt(label,x,y-121-rise);
        if(!captive.freed){ctx.fillStyle='#241d30';ctx.fillRect(x-25,y-114,50,5);ctx.fillStyle='#a9e5ff';ctx.fillRect(x-25,y-114,50*unit.hp/unit.max_hp,5);}
      }
      ctx.restore();return;
    }
    if(unit.kind==='totem'||unit.kind==='generator'){
      if(unit.hp<=0)return;
      const x=unit.x-camera,y=unit.y,hit=unit.pose==='hit'&&unit.pose_time>0;
      ctx.save();
      ctx.drawImage(images[unit.kind],x-35+(hit&&!renderer.reduced?Math.sin(animationTime/25)*2*motion:0),y-82,70,84);
      if(!display.cleanScreenshot){
        ctx.fillStyle='#241d30';ctx.fillRect(x-25,y-90,50,5);
        ctx.fillStyle='#cb9bff';ctx.fillRect(x-25,y-90,50*unit.hp/unit.max_hp,5);
        ctx.font='bold '+(10*display.textScale)+'px monospace';ctx.textAlign='center';ctx.fillStyle='#ecd6ff';ctx.strokeStyle='#171020';ctx.lineWidth=3;
        interactionPrompt(unit.name,x,y-96);
      }
      ctx.restore();return;
    }
    const shared=unit.art_key?bestiary.profile(unit):null;
    const index = Math.max(0,styles.indexOf(foundations[unit.kind] || unit.kind));
    const atlas = unit.id === 'player' ? (index < 6 ? 'heroesA' : 'heroesB') : 'mobs';
    const row = unit.id === 'player' ? index%6 : ({goblin:0,archer:1,knight:2,boss:3,wolf:4,spore:5}[unit.kind] ?? 0);
    const size = unit.kind === 'boss' ? 168 : shared&&['rat','bat','slime','spider','goblin'].includes(shared.rig) ? 80 : unit.kind === 'wolf' ? 63 : 101;
    let x = unit.x, y = unit.y;
    if (previous && snapshot && !snapshot.paused && unit.hp > 0) {
      const old = unit.id === 'player' ? previous.player : previous.enemies.find(e => e.id === unit.id);
      let t = Math.min(1,(now-received)/110);
      if(old && (snapshot.level?.rooms?.[snapshot.room]?.drop_edges||[]).some(edge=>old.x>=edge.x&&old.x<=edge.x+edge.w&&old.y<=edge.y&&worldY>=edge.landing_y))t=renderer.reduced?1:t*t;
      if (old) { const oldY=old.y-(old.elevation??surfaceHeight(old.x,old.y,previous));x = old.x+(x-old.x)*t; y = oldY+(y-oldY)*t; }
    }
    if (unit.hp <= 0) {
      const drawX = x - camera;
      ctx.fillStyle='#03110b70'; ctx.beginPath(); ctx.ellipse(drawX,y+2,size*.28,7,0,0,Math.PI*2); ctx.fill();
      const fallen=animationTime-(deaths.get(unit.id)??0);
      if (unit.id === 'player') {
        const colorAlpha = Math.max(0.18, 0.9 - Math.min(1, fallen / 900) * 0.72);
        const silhouetteAlpha = Math.max(0.55, 0.85 - Math.min(1, fallen / 2000) * 0.2);
        renderer.lastDefeatedPlayerSilhouette = { x: Math.round(drawX), y: Math.round(y), fallen, colorAlpha: Number(colorAlpha.toFixed(3)), silhouetteAlpha: Number(silhouetteAlpha.toFixed(3)), preserved: true, reduced: !!renderer.reduced };
        const fallenFrame = fallen < 320 ? 13 : 14;

        // Preserved silhouette backing layer
        ctx.save();
        if (typeof ctx.filter === 'string') {
          ctx.filter = 'brightness(15%) contrast(150%)';
        }
        ctx.shadowColor = 'rgba(126, 245, 208, 0.35)';
        ctx.shadowBlur = renderer.reduced ? 0 : 4;
        sprite(row, fallenFrame, drawX, y, size, unit.facing, silhouetteAlpha, atlas);
        ctx.restore();

        // Fading color details on top
        sprite(row, fallenFrame, drawX, y, size, unit.facing, colorAlpha, atlas);

        // Rising ethereal soul motes
        if (!renderer.reduced) {
          const motes = 3;
          for (let m = 0; m < motes; m++) {
            const progress = ((animationTime / 700) + m / motes) % 1;
            const mx = drawX + Math.sin(m * 2.3 + animationTime / 300) * 14;
            const my = (y - 8) - progress * 42;
            const mFade = (1 - progress) * 0.5 * Math.min(1, fallen / 400);
            if (mFade > 0) {
              ctx.fillStyle = 'rgba(160, 245, 220, ' + mFade + ')';
              ctx.beginPath();
              ctx.arc(mx, my, 1.8 * (1 - progress * 0.4), 0, Math.PI * 2);
              ctx.fill();
            }
          }
        }
        return;
      }
      if(shared)catalogActor(unit,'defeat',drawX,y,size,fallen<700?.85:.4);
      else sprite(row,fallen<320?13:14,drawX,y,size,unit.facing,fallen<700?.85:.4,atlas);
      return;
    }
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
    const celebrating=unit.id==='player'&&(unit.pose==='victory'||snapshot.status==='complete'||(snapshot.status==='cleared'&&snapshot.room===3))&&unit.pose!=='run';
    const stance=celebrating?(victoryStances[snapshot.build?.class]||victoryStances.vanguard):null;
    if(stance)col=stance.frame;
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
      const shadowScale=renderer.reduced?1:1-Math.min(1,Math.max(0,jump/52))*.42;
      ctx.fillStyle = '#03110b70';
      ctx.beginPath();
      ctx.ellipse(drawX, y + 2, size * 0.28 * shadowScale, 7 * shadowScale, 0, 0, Math.PI * 2);
      ctx.fill();
    }
    if(!renderer.reduced&&motion>0&&!(unit.jump>0)&&['run','guard_walk'].includes(unit.pose)){
      const depth=.75+Math.max(0,Math.min(1,(y-250)/285))*.5;
      ctx.save();ctx.fillStyle='#c7ad79';
      for(let i=0;i<3;i++){
        const phase=(decorationTime/380+i/3)%1;
        ctx.globalAlpha=(1-phase)*.2;
        ctx.beginPath();ctx.ellipse(drawX-(unit.facing||1)*(10+phase*24)*depth,y+3-phase*4,(2+phase*4)*depth,(1+phase*2)*depth,0,0,Math.PI*2);ctx.fill();
      }
      ctx.restore();
    }
    ctx.save();
    if(unit.id!=='player'&&unit.pose==='hit'&&!renderer.reduced)ctx.filter='brightness('+(1+.5*Math.max(0,Math.min(1,(unit.pose_time||0)/.2)))+')';
    const victoryLift=stance?stance.lift+(!renderer.reduced&&motion>0?Math.sin(decorationTime/420)*1.5*motion:0):0;
    if(stance){ctx.translate(drawX,y);ctx.rotate(stance.angle*(unit.facing||1));ctx.translate(-drawX,-y);}
    if(shared)catalogActor(unit,unit.pose,drawX,y-jump+landSquash+recoverySquash+guardStride-ultimateHover-victoryLift,size,1);else sprite(row,col,drawX,y-jump+landSquash+recoverySquash+guardStride-ultimateHover-victoryLift,size,unit.facing,1,atlas);
    ctx.restore();
    if(unit.id==='player'&&snapshot.room_objective?.kind==='carry_relic'&&snapshot.room_objective.carrying)ctx.drawImage(images.relic,drawX-18,y-jump-112,36,36);
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
    if (stance) {
      renderer.lastVictoryPose = { x: Math.round(drawX), y: Math.round(y), pose: unit.pose, poseTime: unit.pose_time, reduced: !!renderer.reduced, subclass:snapshot.build?.class, stance:stance.name, frame:col, angle:stance.angle, lift:victoryLift, color:stance.color };
      ctx.save();
      const auraPulse = renderer.reduced ? 0 : Math.sin(animationTime / 180) * 3;
      ctx.fillStyle = stance.color+'38';
      ctx.beginPath();
      ctx.ellipse(drawX, y + 2, 28 + auraPulse, 8 + auraPulse * 0.25, 0, 0, Math.PI * 2);
      ctx.fill();

      ctx.strokeStyle = stance.color+'bf';
      ctx.lineWidth = 1.5;
      ctx.beginPath();
      ctx.ellipse(drawX, y + 2, 28 + auraPulse, 8 + auraPulse * 0.25, 0, 0, Math.PI * 2);
      ctx.stroke();

      if (!renderer.reduced) {
        const motes = 4;
        for (let m = 0; m < motes; m++) {
          const mProgress = ((animationTime / 500) + m / motes) % 1;
          const mx = drawX + Math.sin(m * 1.8 + animationTime / 250) * 18;
          const my = y - mProgress * 55;
          const mAlpha = Math.sin(mProgress * Math.PI) * 0.8;
          ctx.fillStyle = stance.color;ctx.globalAlpha=mAlpha;
          ctx.beginPath();
          ctx.arc(mx, my, 2 * (1 - mProgress * 0.3), 0, Math.PI * 2);
          ctx.fill();
        }
      }
      ctx.restore();
    }
      if(snapshot.room_objective?.kind==='linked_guardians'&&snapshot.room_objective.targets.includes(unit.id)&&unit.hp>0){
        const active=snapshot.room_objective.bond_active,ty=y-jump-size*.9-34;
        ctx.save();ctx.strokeStyle=active?'#a9e5ff':'#ffe0a6';ctx.lineWidth=2;ctx.beginPath();ctx.arc(drawX,ty,9,0,Math.PI*2);ctx.stroke();
        if(active){ctx.beginPath();ctx.arc(drawX+7,ty,9,0,Math.PI*2);ctx.stroke();}
        if(!display.cleanScreenshot){ctx.font='bold 9px monospace';ctx.textAlign='center';ctx.fillStyle=active?'#a9e5ff':'#ffe0a6';ctx.strokeStyle='#10202e';ctx.lineWidth=3;const label=active?'LINKED · 50% GUARD':'GUARDIAN · UNLINKED';ctx.strokeText(label,drawX,ty-16);ctx.fillText(label,drawX,ty-16);}
        ctx.restore();
      }
      if(snapshot.room_objective?.kind==='interrupt_ritual'&&unit.hp>0){
        const channel=snapshot.room_objective.channels.find(c=>c.enemy_id===unit.id);
        if(channel){
          const ty=y-jump-size*.9-34,color=channel.seconds>=6?'#ffad87':'#dab2ff';
          ctx.save();ctx.strokeStyle=color;ctx.lineWidth=2;ctx.beginPath();ctx.arc(drawX,ty,10,-Math.PI/2,-Math.PI/2+Math.PI*2*channel.seconds/8);ctx.stroke();
          if(!display.cleanScreenshot){ctx.font='bold 9px monospace';ctx.textAlign='center';ctx.fillStyle=color;ctx.strokeStyle='#21102f';ctx.lineWidth=3;const label='RITUAL '+(8-channel.seconds).toFixed(1)+'s';ctx.strokeText(label,drawX,ty-16);ctx.fillText(label,drawX,ty-16);}
          ctx.restore();
        }
      }
      if(snapshot.room_objective?.kind==='marked_hunt'&&snapshot.room_objective.targets.includes(unit.id)&&unit.hp>0){
        const tx=drawX,ty=y-jump-size*.9-32;
        ctx.save();ctx.strokeStyle='#ffca78';ctx.fillStyle='#241707';ctx.lineWidth=2;
        ctx.beginPath();ctx.moveTo(tx,ty-10);ctx.lineTo(tx+10,ty);ctx.lineTo(tx,ty+10);ctx.lineTo(tx-10,ty);ctx.closePath();ctx.fill();ctx.stroke();
        ctx.beginPath();ctx.moveTo(tx-4,ty);ctx.lineTo(tx+4,ty);ctx.moveTo(tx,ty-4);ctx.lineTo(tx,ty+4);ctx.stroke();
        ctx.strokeStyle='#ffca7899';ctx.beginPath();ctx.ellipse(tx,y+2,27,8,0,0,Math.PI*2);ctx.stroke();
        if(!display.cleanScreenshot){ctx.font='bold 9px monospace';ctx.textAlign='center';ctx.fillStyle='#ffe0a6';ctx.strokeStyle='#241707';ctx.lineWidth=3;ctx.strokeText('HUNT TARGET',tx,ty-15);ctx.fillText('HUNT TARGET',tx,ty-15);}
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
      if (!display.cleanScreenshot && unit.kind === 'treasure' && unit.fleeing && unit.hp > 0) {
        const panicLift = renderer.reduced || motion === 0 ? 0 : Math.sin(decorationTime / 160) * 2;
        const panicY = y - jump - size * .9 - 32 - panicLift;
        ctx.save();
        ctx.font = 'bold ' + (10 * display.textScale) + 'px monospace';
        ctx.textAlign = 'center';
        ctx.fillStyle = '#ffd477';
        ctx.strokeStyle = '#0a1715';
        ctx.lineWidth = 3;
        ctx.strokeText('FLEEING!', drawX, panicY);
        ctx.fillText('FLEEING!', drawX, panicY);
        ctx.restore();
      }
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

        const precision=snapshot.build.class==='marksman'&&snapshot.resource>0&&unit.hp>0;
        if(!renderer.reduced||precision){
          const bx = (precision?size*.42:rx) * 0.82, by = (precision?size*.52:ry) * 0.82;
          const bLen = 6;
          ctx.strokeStyle = precision?'#ffe39b':'#a6ffe5';
          ctx.lineWidth = 1.5;
          ctx.beginPath();
          ctx.moveTo(targetX - bx, targetY - by + bLen); ctx.lineTo(targetX - bx, targetY - by); ctx.lineTo(targetX - bx + bLen, targetY - by);
          ctx.moveTo(targetX + bx - bLen, targetY - by); ctx.lineTo(targetX + bx, targetY - by); ctx.lineTo(targetX + bx, targetY - by + bLen);
          ctx.moveTo(targetX - bx, targetY + by - bLen); ctx.lineTo(targetX - bx, targetY + by); ctx.lineTo(targetX - bx + bLen, targetY + by);
          ctx.moveTo(targetX + bx - bLen, targetY + by); ctx.lineTo(targetX + bx, targetY + by); ctx.lineTo(targetX + bx, targetY + by - bLen);
          ctx.stroke();
        }

        if(snapshot.build.class==='beastmaster'&&unit.hp>0&&!display.cleanScreenshot){ctx.font='bold '+(10*display.textScale)+'px monospace';ctx.textAlign='center';ctx.fillStyle='#bce8a5';ctx.strokeStyle='#10221d';ctx.lineWidth=3;interactionPrompt('PACK TARGET',targetX,y+19,true);}
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
      drawAtlas(img,source.x*img.width,source.y*img.height,source.width*img.width,source.height*img.height,-size/2,-size*.91,size,size);ctx.restore();
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
    const framing=cameraFrame(snapshot);
    let targetCamera=framing.target;
    if(!snapshot?.paused){
      const player=snapshot?.player;
      const recoiling=player&&(player.pose==='hit'&&player.pose_time>0||player.knockdown>0||Math.abs(player.recoil_x||0)>0);
      if(recoiling){
        cameraRecovering=true;
        // Hold through impact unless the player or a framed boss would leave view.
        const low=Math.max(framing.min,player.x-928),high=Math.min(framing.max,player.x-32);
        targetCamera=Math.max(low,Math.min(high,camera));
      }
      camera=display.cameraSmooth||cameraRecovering?camera+(targetCamera-camera)*Math.min(1,dt*8):targetCamera;
      if(!recoiling&&Math.abs(targetCamera-camera)<.25)cameraRecovering=false;
    }
    camera=Math.max(framing.min,Math.min(framing.max,camera));
    renderer.cameraFraming={x:camera,bosses:framing.bosses};
    const backgroundX=Math.min(0,-camera*.35),backgroundWidth=Math.max(1184,960-backgroundX);
    // Slow background parallax retains the full walkable foreground.
    const region = snapshot?.level && ['fighting','cleared'].includes(snapshot.status) ? snapshot.level.region : previewLevel?.region;
    const background = region !== undefined ? images.regions : snapshot?.room===2 ? images.boss : images.area;
    if(region !== undefined){
      const row=Math.floor(region/2), top=regionRows[row], bottom=regionRows[row+1];
      drawAtlas(background,region%2*background.width/2+2,top*background.height+2,background.width/2-4,(bottom-top)*background.height-4,backgroundX,0,backgroundWidth,540);
    }else drawAtlas(background,0,0,background.width,background.height,backgroundX,0,backgroundWidth,540);
    if(region===3&&!renderer.reduced&&motion>0&&display.flashIntensity>0){
      const phase=decorationTime%8000-1000;
      if(phase>=0&&phase<240){
        const cycle=Math.floor(decorationTime/8000),x=260+(cycle%3)*220-camera*.35;
        ctx.save();ctx.strokeStyle='#b8c9ec';ctx.lineWidth=2;
        ctx.globalAlpha=.28*(1-phase/240)*display.flashIntensity;
        ctx.beginPath();ctx.moveTo(x,35);ctx.lineTo(x-18,83);ctx.lineTo(x+8,77);ctx.lineTo(x-12,135);ctx.stroke();ctx.restore();
      }
    }
    if(region===5&&!renderer.reduced&&motion>0){
      ctx.save();ctx.strokeStyle='#76c7cb';ctx.lineWidth=1;ctx.globalAlpha=.16;
      for(let i=0;i<8;i++){
        const x=90+i*139-camera*.35,y=340+(i*47)%160;
        const radius=24+Math.sin(decorationTime*.001+i)*5;
        ctx.beginPath();ctx.ellipse(x,y,radius,3,0,0,Math.PI*2);ctx.stroke();
      }
      ctx.restore();
    }
    if (snapshot && snapshot.room === 1) { ctx.fillStyle='#61532316';ctx.fillRect(0,0,960,540); }
    if (!renderer.reduced && display.particles) {
      for(let i=0;i<Math.round(22*display.particleIntensity);i++) {
        const x=((i*157+decorationTime*.004*(i%3+1)-camera*.35)%1000+1000)%1000;
        const vertical=region===1?-decorationTime*.012:region===2?decorationTime*.018:0;
        const y=80+((i*41+vertical)%380+380)%380+Math.sin(decorationTime*.0005+i)*14*motion;
        ctx.globalAlpha=.3+Math.sin(decorationTime*.001+i)*.2*motion;
        ctx.fillStyle=region===1?'#d6aaa0':region===2?'#e1f5ff':i%3?'#a9ce8c':'#ffd98a';
        ctx.fillRect(x,y,region===2?3:2,2);
      }
      ctx.globalAlpha=1;
    }
    if (!snapshot) { const index=Math.max(0,styles.indexOf(foundations[previewStyle]||previewStyle));sprite(index%6,renderer.reduced?0:Math.floor(decorationTime/650)%2,630,400,113,-1,1,index<6?'heroesA':'heroesB');return; }
    interactionPrompts=[];
    const hazardOverlays=[];
    const run=snapshot;
    const arena=run.practice?.arena||run.level?.rooms[run.room];
    // Rear scenery only: all actors, pickups, attacks and warnings draw afterward.
    for(const vent of arena?.steam_vents||[]){
      const x=vent.x-camera,y=vent.y+vent.h;
      ctx.save();ctx.fillStyle='#39463d';ctx.fillRect(x+vent.w*.3,y-4,vent.w*.4,4);
      ctx.strokeStyle='#88958a';ctx.lineWidth=1;ctx.beginPath();ctx.moveTo(x+vent.w*.3,y-4);ctx.lineTo(x+vent.w*.7,y-4);ctx.stroke();
      ctx.beginPath();ctx.rect(x,vent.y,vent.w,vent.h);ctx.clip();ctx.fillStyle='#ced8c8';
      for(let puff=0;puff<5;puff++){
        const rise=renderer.reduced||motion===0?(puff+.5)/5:((decorationTime*.00022+puff/5)%1);
        ctx.globalAlpha=.18*(1-rise);
        ctx.beginPath();ctx.ellipse(x+vent.w*(.5+Math.sin(puff*2)*.12*rise),y-rise*vent.h,vent.w*(.12+rise*.22),6+rise*12,0,0,Math.PI*2);ctx.fill();
      }
      ctx.restore();
    }
    // The decorative floor continues past the playable space; mark its real rim.
    ctx.save();ctx.fillStyle='#071a1640';
    ctx.fillRect(25-camera,305,1550,10);ctx.fillRect(25-camera,490,1550,12);
    ctx.fillRect(25-camera,315,10,175);ctx.fillRect(1565-camera,315,10,175);
    ctx.strokeStyle='#071a168f';ctx.lineWidth=4;ctx.strokeRect(35-camera,315,1530,175);
    ctx.strokeStyle='#b5c9a885';ctx.lineWidth=1;ctx.strokeRect(35-camera,315,1530,175);
    ctx.strokeStyle='#b5c9a860';ctx.lineWidth=1;ctx.beginPath();
    for(let x=45;x<1565;x+=24){ctx.moveTo(x-camera,307);ctx.lineTo(x+6-camera,313);ctx.moveTo(x-camera,492);ctx.lineTo(x+6-camera,498);}
    ctx.stroke();
    if(run.player.x<50||run.player.x>1550||run.player.y<330||run.player.y>475){
      ctx.font='bold 10px monospace';ctx.textAlign='center';ctx.fillStyle='#dce6cf';ctx.strokeStyle='#102419';ctx.lineWidth=3;
      interactionPrompt('ARENA EDGE',Math.max(70,Math.min(890,run.player.x-camera)),run.player.y<330?292:run.player.y>475?518:run.player.y-100);
    }
    ctx.restore();
    if(display.layoutGrid&&!display.cleanScreenshot){
      ctx.save();
      ctx.beginPath();ctx.rect(35-camera,315,1530,175);ctx.clip();
      ctx.strokeStyle='#c8f5e052';ctx.lineWidth=1;ctx.beginPath();
      for(let x=50;x<1565;x+=50){ctx.moveTo(x-camera,315);ctx.lineTo(x-camera,490);}
      for(let y=350;y<490;y+=50){ctx.moveTo(35-camera,y);ctx.lineTo(1565-camera,y);}
      ctx.stroke();ctx.restore();
      ctx.save();ctx.strokeStyle='#d2ffe0';ctx.lineWidth=2;ctx.setLineDash([6,4]);ctx.strokeRect(35-camera,315,1530,175);ctx.setLineDash([]);
      ctx.font='10px monospace';ctx.textAlign='center';ctx.fillStyle='#d2ffe0';ctx.strokeStyle='#071c14';ctx.lineWidth=3;
      for(let x=100;x<1565;x+=100){const sx=x-camera;if(sx<25||sx>935)continue;ctx.strokeText('X '+x,sx,310);ctx.fillText('X '+x,sx,310);}
      ctx.textAlign='left';for(let y=350;y<490;y+=50){ctx.strokeText('Y '+y,8,y-3);ctx.fillText('Y '+y,8,y-3);}
      const label='GRID 50 · Fighter '+Math.round(run.player.x)+', '+Math.round(run.player.y)+' · Bounds X 35–1565 / Y 315–490';
      ctx.fillStyle='#071c14df';ctx.fillRect(8,500,610,20);ctx.fillStyle='#d2ffe0';ctx.fillText(label,14,514);ctx.restore();
    }

    if(!display.cleanScreenshot&&run.practice&&['movement','jump'].includes(run.practice.mode)){
      ctx.save();ctx.strokeStyle='#e3f9ac';ctx.lineWidth=4;ctx.setLineDash([10,7]);ctx.beginPath();ctx.moveTo(run.practice.goal_x-camera,250);ctx.lineTo(run.practice.goal_x-camera,535);ctx.stroke();ctx.setLineDash([]);ctx.fillStyle='#e3f9ac';ctx.font='bold 14px monospace';ctx.textAlign='center';ctx.fillText('FINISH',run.practice.goal_x-camera,240);ctx.restore();
    }
    decals=decals.filter(d=>now-d.started<1800);
    if(!renderer.reduced&&motion>0)decals.forEach(d=>{
      const x=d.x-camera;if(x<-30||x>990)return;
      ctx.save();ctx.translate(x,d.y);ctx.scale(1,.4);ctx.strokeStyle=decalColors[d.kind];ctx.lineWidth=2;
      ctx.globalAlpha=.55*(1-(now-d.started)/1800)*display.effectIntensity;
      ctx.beginPath();
      if(d.kind==='ice'){
        for(let i=0;i<6;i++){const a=i*Math.PI/3;ctx.moveTo(0,0);ctx.lineTo(Math.cos(a)*22,Math.sin(a)*22);}
      }else if(d.kind==='poison'){
        for(let i=0;i<3;i++){const a=i*Math.PI*2/3,cx=Math.cos(a)*10,cy=Math.sin(a)*10;ctx.moveTo(cx+9,cy);ctx.arc(cx,cy,9,0,Math.PI*2);}
      }else if(d.kind==='radiant'){
        ctx.moveTo(0,-25);ctx.lineTo(22,0);ctx.lineTo(0,25);ctx.lineTo(-22,0);ctx.closePath();
      }else if(d.kind==='rune'){
        ctx.rect(-17,-17,34,34);ctx.moveTo(-17,-17);ctx.lineTo(17,17);ctx.moveTo(17,-17);ctx.lineTo(-17,17);
      }else{
        if(d.kind==='fire')ctx.setLineDash([7,4]);
        ctx.arc(0,0,22,0,Math.PI*2);
        if(d.kind==='void'){ctx.moveTo(11,0);ctx.arc(0,0,11,0,Math.PI*2);}
      }
      ctx.stroke();ctx.restore();
    });
    for(const p of arena?.platforms||[]){
      const x=p.x-camera,y=p.y,w=p.w,h=p.h,r=p.ramp,z=p.rise;
      const outer=[[x,y],[x+w,y],[x+w,y+h],[x,y+h]],top=[[x+r,y+r-z],[x+w-r,y+r-z],[x+w-r,y+h-r-z],[x+r,y+h-r-z]];
      ctx.save();ctx.strokeStyle='#24392e';ctx.lineWidth=2;
      const face=(points,shade)=>{ctx.beginPath();points.forEach(([px,py],i)=>i?ctx.lineTo(px,py):ctx.moveTo(px,py));ctx.closePath();ctx.save();ctx.clip();ctx.drawImage(images.platformSurface,x,y,w,h);ctx.fillStyle='rgba(10,24,17,'+shade+')';ctx.fillRect(x,y,w,h);ctx.restore();ctx.stroke();};
      for(let i=0;i<4;i++)face([outer[i],outer[(i+1)%4],top[(i+1)%4],top[i]],[.08,.22,.28,.14][i]);
      face(top,.02);ctx.lineWidth=1;
      ctx.strokeStyle='#bfccb0';ctx.beginPath();ctx.moveTo(...top[0]);ctx.lineTo(...top[1]);ctx.stroke();
      if(!display.cleanScreenshot){ctx.font='bold 10px monospace';ctx.textAlign='center';ctx.fillStyle='#dfebcf';ctx.strokeStyle='#1b3024';ctx.lineWidth=3;ctx.strokeText('RAISED · SLOPED EDGES',x+w/2,y-9);ctx.fillText('RAISED · SLOPED EDGES',x+w/2,y-9);}ctx.restore();
    }
    (arena?.hazards||[]).forEach(h=>{
      if(h.disabled||run.status!=='fighting'){
        ctx.save();const x=h.x-camera;
        // Static ground residue replaces the live warning silhouette after shutdown.
        ctx.fillStyle='#26392e';ctx.globalAlpha=.55;ctx.beginPath();ctx.ellipse(x+h.w/2,h.y+h.h/2,h.w*.46,h.h*.36,0,0,Math.PI*2);ctx.fill();
        ctx.strokeStyle='#53645a';ctx.globalAlpha=.7;ctx.lineWidth=1;ctx.setLineDash([]);ctx.beginPath();
        for(let i=0;i<3;i++){const sx=x+h.w*(.16+i*.25),sy=h.y+h.h*(.35+(i%2)*.23);ctx.moveTo(sx,sy);ctx.lineTo(sx+h.w*.09,sy-h.h*.1);ctx.lineTo(sx+h.w*.17,sy+h.h*.08);}
        ctx.stroke();ctx.globalAlpha=1;
        if(!display.cleanScreenshot&&display.hazardLabels){ctx.font='bold 9px monospace';ctx.textAlign='center';ctx.fillStyle='#b2d2c6';ctx.strokeStyle='#10221d';ctx.lineWidth=3;interactionPrompt('OFF',h.x+h.w/2-camera,h.y+h.h/2+3,true);}
        ctx.restore();return;
      }
      if(h.generator_id){const source=run.enemies.find(e=>e.id===h.generator_id&&e.hp>0);if(source){ctx.save();ctx.strokeStyle='#73dddf99';ctx.lineWidth=1;ctx.setLineDash([4,4]);ctx.beginPath();ctx.moveTo(source.x-camera,source.y);ctx.lineTo(h.x+h.w/2-camera,h.y+h.h/2);ctx.stroke();ctx.restore();}}

      const phase=(run.clock+h.offset)%h.period, warning=phase<1.2, active=phase>=1.2&&phase<1.2+h.duration&&run.status==='fighting';
      const x=h.x-camera,color={spikes:'#d6dce4',fire:'#ff9a52',ice:'#9be5ff',rune:'#d1acff',poison:'#c7ee76',thorns:'#b5d780',radiant:'#d7dfff',void:'#b194ff'}[h.kind]||'#ffbf70';
      ctx.save();ctx.fillStyle=color;ctx.strokeStyle=color;ctx.lineWidth=active?3:1;ctx.globalAlpha=active?.55:warning?.18:.06;ctx.fillRect(x,h.y,h.w,h.h);ctx.globalAlpha=active?1:warning?.7:.2;
      ctx.setLineDash(warning?[5,4]:[]);ctx.strokeRect(x,h.y,h.w,h.h);ctx.setLineDash([]);
      if(h.tile&&!warning&&!active){
        ctx.save();ctx.globalAlpha=.35;ctx.fillStyle='#427d65';ctx.fillRect(x,h.y,h.w,h.h);ctx.globalAlpha=.9;ctx.strokeStyle='#addec7';ctx.lineWidth=2;ctx.strokeRect(x+2,h.y+2,h.w-4,h.h-4);ctx.restore();
      }
      if(h.kind==='spikes'){
        const extension=renderer.reduced?(active?1:0):phase<1.2?Math.max(0,(phase-1.02)/.18):active?1:Math.max(0,1-(phase-1.2-h.duration)/.18);
        ctx.save();ctx.globalAlpha=1;
        for(let row=0;row<2;row++)for(let sx=x+12;sx<x+h.w-6;sx+=22){
          const sy=h.y+h.h*(row?.78:.35),height=16*extension;
          ctx.fillStyle='#394148';ctx.fillRect(sx-5,sy,10,3);
          if(height>0){ctx.fillStyle='#cbd3dd';ctx.strokeStyle='#53606c';ctx.lineWidth=1;ctx.beginPath();ctx.moveTo(sx-5,sy);ctx.lineTo(sx,sy-height);ctx.lineTo(sx+5,sy);ctx.closePath();ctx.fill();ctx.stroke();}
        }
        ctx.restore();
      }
      if(h.kind==='poison'&&phase>=1.2&&phase<1.2+h.duration+.45){
        const fading=Math.max(0,phase-1.2-h.duration)/.45,drift=renderer.reduced?0:Math.sin(phase*3)*2*motion;
        ctx.save();ctx.beginPath();ctx.rect(x,h.y,h.w,h.h);ctx.clip();ctx.fillStyle='#b8d778';ctx.globalAlpha=.24*(1-fading);
        for(let puff=0;puff<7;puff++){
          const px=x+h.w*(.13+(puff%4)*.24),py=h.y+h.h*(puff<4?.4:.7);
          const spread=renderer.reduced?1:1+fading*.35*motion;
          ctx.beginPath();ctx.ellipse(px+drift,py,h.w*.18*spread,h.h*.32*spread,0,0,Math.PI*2);ctx.fill();
        }
        ctx.restore();
      }
      if(display.hazardPatterns!==false&&(warning||active)){ctx.save();ctx.beginPath();ctx.rect(x,h.y,h.w,h.h);ctx.clip();drawHazardPattern(ctx,h.kind,x,h.y,h.w,h.h,active,color);ctx.restore();}
      if(display.hazardContrast&&(warning||active)){ctx.globalAlpha=1;ctx.strokeStyle='#fff8d8';ctx.lineWidth=3;ctx.setLineDash(active?[]:[8,4]);ctx.strokeRect(x-2,h.y-2,h.w+4,h.h+4);ctx.setLineDash([]);}
      if(!display.cleanScreenshot&&display.hazardLabels){
        // The frozen combat clock also drives damage; do not count down with wall time.
        const remaining=active?1.2+h.duration-phase:warning?1.2-phase:h.period-phase+1.2;
        const seconds=(Math.ceil(Math.max(0,remaining-1e-9)*10)/10).toFixed(1)+'s';
        const label=active?(h.jumpable===true?'JUMP':'MOVE')+' · '+seconds:warning?h.kind.toUpperCase()+' IN '+seconds:'SAFE · '+seconds;
        ctx.globalAlpha=1;ctx.font='bold '+(10*display.textScale)+'px monospace';ctx.textAlign='center';ctx.fillStyle=warning||active?color:'#b5edce';ctx.strokeStyle='#10221d';ctx.lineWidth=3;
        interactionPrompt(label,x+h.w/2,h.y-6,true);
      }
      if(warning||active)hazardOverlays.push({x,y:h.y,w:h.w,h:h.h,warning,color});
      if(active&&h.kind!=='spikes'&&!renderer.reduced)fx(effectRows[h.kind]??3,Math.floor(now/90)%6,x+h.w/2,h.y+h.h/2,h.w,.7);
      ctx.restore();
    });
    run.enemies.forEach(e => {
      const rangedWindup = e.kind === 'archer' || e.kind === 'boss' && e.art_key && (e.attacks + 1) % 2 === 0;
      if (!display.cleanScreenshot && e.hp > 0 && e.windup > 0 && rangedWindup) {
        ctx.save();
        ctx.strokeStyle = '#ffbd81';
        ctx.lineWidth = 2;
        ctx.setLineDash([7, 5]);
        ctx.beginPath();
        ctx.moveTo(e.x - camera, e.y - (e.elevation||0) - 30);
        ctx.lineTo(run.player.x - camera, run.player.y - (run.player.elevation||0) - 30);
        ctx.stroke();
        ctx.restore();
      }
      if(!display.cleanScreenshot&&e.hp>0 && e.windup>0 && e.kind==='boss'&&!rangedWindup) {
        const attackName=e.attack_name||(e.art_key&&(e.attacks+1)%2===0?'Aimed Volley':'Ground Slam'),targetY=e.target_y-surfaceHeight(e.target_x,e.target_y);
        ctx.fillStyle='#c8783b55';ctx.strokeStyle='#ffce7d';ctx.lineWidth=2;ctx.beginPath();ctx.ellipse(e.target_x-camera,targetY,125,62,0,0,Math.PI*2);ctx.fill();ctx.stroke();
        ctx.fillStyle='#ffe2b0';ctx.font='bold '+(12*display.textScale)+'px monospace';ctx.textAlign='center';ctx.fillText(attackName.toUpperCase()+' · JUMP OR MOVE',e.target_x-camera,targetY+4);
      }
    });
    const activeRangeSkill = renderer.rangeSkill || (display.skillRange && run.build?.skills?.[0] ? run.build.skills[0] : null);
    if (!display.cleanScreenshot && activeRangeSkill && run.player && ['fighting','cleared'].includes(run.status)) {
      const p = run.player, px = p.x - camera, py = p.y-(p.elevation||0), ref = activeRangeSkill.reference;
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
      const p = run.player, px = p.x - camera, py = p.y - (p.elevation||0) - (p.jump || 0) * 120;
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
    if(run.room_objective?.kind==='rune_gate'){
      const goal=run.room_objective,combat=run.enemies.some(e=>e.hp>0),next=goal.sequence[goal.collected];
      ctx.save();
      for(const seal of goal.pickups){const x=seal.x-camera,y=seal.y,color=seal.collected?'#bbf5c9':!combat&&seal.id===next?'#ffe19e':'#b6aed3';
        ctx.globalAlpha=combat ? .45 : 1;ctx.strokeStyle=color;ctx.fillStyle='#17132599';ctx.lineWidth=seal.id===next?3:1;ctx.beginPath();ctx.ellipse(x,y,28,18,0,0,Math.PI*2);ctx.fill();ctx.stroke();ctx.drawImage(images.sigil,x-17,y-38,34,34);
        if(!display.cleanScreenshot){ctx.font='bold 12px monospace';ctx.textAlign='center';ctx.fillStyle=color;ctx.strokeStyle='#171325';ctx.lineWidth=3;const label='SEAL '+seal.id+(seal.collected?' ✓':'');interactionPrompt(label,x,y-43);}
      }
      ctx.globalAlpha=1;const gx=1450-camera,gy=330;ctx.strokeStyle=goal.complete?'#bbf5c9':'#b6aed3';ctx.lineWidth=7;ctx.beginPath();ctx.moveTo(gx-42,gy);ctx.lineTo(gx-42,gy-80);ctx.quadraticCurveTo(gx,gy-138,gx+42,gy-80);ctx.lineTo(gx+42,gy);ctx.stroke();
      if(!goal.complete){ctx.lineWidth=2;ctx.strokeStyle='#cfb6ef';for(let n=-24;n<=24;n+=12){ctx.beginPath();ctx.moveTo(gx+n,gy);ctx.lineTo(gx+n,gy-85);ctx.stroke();}}
      if(!display.cleanScreenshot){ctx.font='bold 11px monospace';ctx.textAlign='center';ctx.fillStyle='#f0deff';ctx.strokeStyle='#171325';ctx.lineWidth=3;const label=goal.complete?'GATE OPEN':goal.sequence.join(' → ');interactionPrompt(label,gx,gy-126);}
      ctx.restore();
    }
    if(run.room_objective?.kind==='split_defense'){
      ctx.save();
      for(const lane of run.room_objective.lanes){const w=lane.ward,x=w.x-camera,color=lane.contested?'#ffab83':w.id==='upper-ward'?'#9ee8ef':'#d9bbff';ctx.strokeStyle=color;ctx.lineWidth=2;ctx.setLineDash([7,6]);ctx.beginPath();ctx.moveTo(x,w.y);ctx.lineTo(1500-camera,w.y);ctx.stroke();ctx.setLineDash([]);ctx.beginPath();ctx.ellipse(x,w.y,45,22,0,0,Math.PI*2);ctx.stroke();
        if(!display.cleanScreenshot&&(x<24||x>936)){ctx.font='bold 11px monospace';ctx.textAlign=x<24?'left':'right';ctx.fillStyle=color;ctx.strokeStyle='#171325';ctx.lineWidth=3;const label=(x<24?'← ':'')+w.name.toUpperCase()+' '+Math.ceil(w.hp)+'%'+(lane.contested?' !':'')+(x>936?' →':'');interactionPrompt(label,x<24?25:935,w.y);}
      }
      ctx.restore();
    }
    if(run.room_objective?.kind==='protect_lantern'){
      const goal=run.room_objective,z=goal.zone,x=z.x-camera;
      ctx.save();ctx.strokeStyle=goal.contested?'#ffab83':'#ffe29a88';ctx.lineWidth=2;ctx.setLineDash([6,5]);ctx.beginPath();ctx.ellipse(x,z.y,z.radius_x,z.radius_y,0,0,Math.PI*2);ctx.stroke();ctx.setLineDash([]);
      if(!goal.complete&&goal.lantern.hp>0&&!display.cleanScreenshot&&(x<24||x>936)){ctx.font='bold 11px monospace';ctx.textAlign=x<24?'left':'right';ctx.fillStyle='#ffe6b5';ctx.strokeStyle='#21190e';ctx.lineWidth=3;const edge=x<24?25:935,label=x<24?'← LANTERN':'LANTERN →';interactionPrompt(label,edge,340);}
      ctx.restore();
    }
    if(run.room_objective?.kind==='linked_guardians'&&run.room_objective.bond_active){
      const pair=run.room_objective.targets.map(id=>run.enemies.find(e=>e.id===id));
      if(pair.every(e=>e&&e.hp>0)){ctx.save();ctx.strokeStyle='#a9e5ff';ctx.lineWidth=2;ctx.setLineDash([7,5]);ctx.beginPath();ctx.moveTo(pair[0].x-camera,pair[0].y-35);ctx.lineTo(pair[1].x-camera,pair[1].y-35);ctx.stroke();ctx.restore();}
    }
    if(run.room_objective?.kind==='escape_collapse'){
      const goal=run.room_objective,edge=goal.collapse_x-camera,z=goal.zone,x=z.x-camera;
      ctx.save();
      if(!goal.complete&&goal.collapse_x>0){
        ctx.fillStyle='#281218aa';ctx.fillRect(-camera,300,goal.collapse_x,220);ctx.strokeStyle='#ffa377';ctx.lineWidth=3;ctx.beginPath();ctx.moveTo(edge,300);ctx.lineTo(edge,520);ctx.stroke();
        ctx.strokeStyle='#ee865b99';ctx.lineWidth=1;
        for(let y=310;y<520;y+=32){ctx.beginPath();ctx.moveTo(edge-38,y);ctx.lineTo(edge-13,y+9);ctx.lineTo(edge-29,y+19);ctx.lineTo(edge,y+28);ctx.stroke();}
        if(!display.cleanScreenshot){ctx.font='bold 12px monospace';ctx.textAlign='left';ctx.fillStyle='#ffd2ad';ctx.strokeStyle='#281218';ctx.lineWidth=4;const tx=Math.max(20,Math.min(780,edge+12));interactionPrompt('COLLAPSE →',tx,305);}
      }
      ctx.strokeStyle='#b5f5dc';ctx.lineWidth=2;ctx.beginPath();ctx.ellipse(x,z.y,z.radius_x,z.radius_y,0,0,Math.PI*2);ctx.stroke();
      if(!display.cleanScreenshot){ctx.font='bold 11px monospace';ctx.textAlign='center';ctx.fillStyle='#d5ffea';ctx.strokeStyle='#10251d';ctx.lineWidth=4;const label=goal.complete?'ESCAPED':'ESCAPE EXIT';interactionPrompt(label,x,z.y-42);if(x>936){ctx.textAlign='right';interactionPrompt('EXIT →',935,340);}}
      ctx.restore();
    }
    if(run.room_objective?.kind==='interrupt_ritual'&&!run.room_objective.complete){
      for(const channel of run.room_objective.channels){
        const enemy=run.enemies.find(e=>e.id===channel.enemy_id);if(!enemy||enemy.hp<=0)continue;
        const x=enemy.x-camera,y=enemy.y,warning=channel.seconds>=6;
        ctx.save();ctx.strokeStyle=warning?'#ffad87':'#dab2ff88';ctx.fillStyle=warning?'#d95d3926':'#9c5ed911';ctx.lineWidth=warning?2:1;
        ctx.beginPath();ctx.ellipse(x,y,150,85,0,0,Math.PI*2);ctx.fill();ctx.stroke();
        ctx.beginPath();ctx.ellipse(x,y,25,14,0,0,Math.PI*2);ctx.stroke();
        for(let n=0;n<6;n++){const a=n*Math.PI/3;ctx.beginPath();ctx.moveTo(x+Math.cos(a)*30,y+Math.sin(a)*18);ctx.lineTo(x+Math.cos(a)*40,y+Math.sin(a)*24);ctx.stroke();}
        ctx.restore();
      }
    }
    if(run.room_objective?.kind==='escort_spirit'){
      const goal=run.room_objective,z=goal.zone,x=z.x-camera;
      ctx.save();ctx.strokeStyle='#b5f5dc';ctx.lineWidth=2;ctx.beginPath();ctx.ellipse(x,z.y,z.radius_x,z.radius_y,0,0,Math.PI*2);ctx.stroke();
      ctx.font='bold 11px monospace';ctx.textAlign='center';ctx.fillStyle='#d5ffea';ctx.strokeStyle='#10251d';ctx.lineWidth=4;interactionPrompt('SPIRIT EXIT',x,z.y-42);
      const sx=goal.escort.x-camera;if(!goal.complete&&(sx<24||sx>936)){ctx.textAlign=sx<24?'left':'right';const edge=sx<24?25:935,label=sx<24?'← SPIRIT':'SPIRIT →';interactionPrompt(label,edge,340);}
      ctx.restore();
    }
    if(run.room_objective?.kind==='moving_beacons'){
      const goal=run.room_objective,z=goal.zone,x=z.x-camera,y=z.y,color=goal.complete?'#baffd0':goal.charging?'#b9eaff':'#d1c3ff';
      ctx.save();ctx.fillStyle='#7496cf26';ctx.strokeStyle=color;ctx.lineWidth=2;
      ctx.beginPath();ctx.ellipse(x,y,z.radius_x,z.radius_y,0,0,Math.PI*2);ctx.fill();ctx.stroke();
      ctx.lineWidth=5;ctx.beginPath();ctx.ellipse(x,y,z.radius_x+5,z.radius_y+5,0,-Math.PI/2,-Math.PI/2+Math.PI*2*goal.seconds/3);ctx.stroke();
      ctx.drawImage(images.sigil,x-18,y-28,36,36);
      ctx.font='bold 11px monospace';ctx.textAlign='center';ctx.fillStyle=color;ctx.strokeStyle='#12182c';ctx.lineWidth=4;
      const label=goal.complete?'ALL BEACONS CAPTURED':'BEACON '+(goal.collected+1)+'/3 · '+Math.floor(goal.seconds)+'/3s';interactionPrompt(label,x,y-z.radius_y-65);
      if(!goal.complete&&(x<24||x>936)){const edge=x<24?25:935;ctx.textAlign=x<24?'left':'right';const direction=x<24?'← BEACON':'BEACON →';interactionPrompt(direction,edge,340);}
      ctx.restore();
    }
    if(run.room_objective?.kind==='carry_relic'){
      const goal=run.room_objective,z=goal.zone,x=z.x-camera,y=z.y;
      ctx.save();ctx.strokeStyle=goal.complete?'#baffd0':'#ffd17c';ctx.fillStyle='#b77e2626';ctx.lineWidth=3;
      ctx.beginPath();ctx.ellipse(x,y,z.radius_x,z.radius_y,0,0,Math.PI*2);ctx.fill();ctx.stroke();
      ctx.font='bold 11px monospace';ctx.textAlign='center';ctx.fillStyle='#ffe1a4';ctx.strokeStyle='#21190c';ctx.lineWidth=4;
      const label=goal.complete?'RELIC DELIVERED':'DELIVER RELIC';interactionPrompt(label,x,y-z.radius_y-10);
      if(goal.complete)ctx.drawImage(images.relic,x-20,y-35,40,40);
      if(!goal.complete){const target=goal.carrying?z:goal.relic,tx=target.x-camera;if(tx<24||tx>936){const edge=tx<24?25:935;ctx.textAlign=tx<24?'left':'right';const hint=(tx<24?'← ':'')+(goal.carrying?'EXIT SEAL':'RELIC')+(tx>936?' →':'');interactionPrompt(hint,edge,340);}}
      ctx.restore();
    }
    if(run.room_objective?.kind==='hold_circle'){
      const goal=run.room_objective,z=goal.zone,x=z.x-camera,y=z.y,color=goal.complete?'#baffd0':goal.contested?'#ffd078':'#9df6d3';
      ctx.save();ctx.fillStyle=goal.contested?'#c38c2526':'#56c89726';ctx.strokeStyle=color;ctx.lineWidth=2;
      ctx.beginPath();ctx.ellipse(x,y,z.radius_x,z.radius_y,0,0,Math.PI*2);ctx.fill();ctx.stroke();
      ctx.lineWidth=6;ctx.beginPath();ctx.ellipse(x,y,z.radius_x+5,z.radius_y+5,0,-Math.PI/2,-Math.PI/2+Math.PI*2*goal.seconds/goal.target);ctx.stroke();
      ctx.drawImage(images.sigil,x-16,y-24,32,32);
      ctx.font='bold 12px monospace';ctx.textAlign='center';ctx.fillStyle=color;ctx.strokeStyle='#081914';ctx.lineWidth=4;
      const label=goal.complete?'CHARGED':goal.contested?'CONTESTED':'HOLD '+Math.floor(goal.seconds)+' / '+goal.target+'s';interactionPrompt(label,x,y-z.radius_y-12);
      if(x<24||x>936){const edge=x<24?25:935;ctx.textAlign=x<24?'left':'right';const direction=x<24?'← CIRCLE':'CIRCLE →';interactionPrompt(direction,edge,355);}
      ctx.restore();
    }
    function drawSigil(pickup){
      const x=pickup.x-camera,y=pickup.y;
      ctx.save();ctx.strokeStyle='#9df6d3';ctx.lineWidth=2;ctx.beginPath();ctx.ellipse(x,y,23,9,0,0,Math.PI*2);ctx.stroke();
      const bob=renderer.reduced||!display.lootMotion?0:Math.sin(decorationTime/250+pickup.id)*3*motion;
      ctx.drawImage(images.sigil,x-24,y-45+bob,48,48);
      if(!display.cleanScreenshot){ctx.font='bold 11px monospace';ctx.textAlign='center';ctx.fillStyle='#caffeb';ctx.strokeStyle='#081914';ctx.lineWidth=3;const label='SIGIL '+pickup.id;interactionPrompt(label,x,y-50);}
      if(x<24||x>936){const edge=x<24?25:935;ctx.font='bold 12px monospace';ctx.textAlign=x<24?'left':'right';ctx.fillStyle='#b8ffdf';ctx.strokeStyle='#081914';ctx.lineWidth=4;const label=(x<24?'← ':'')+'SIGIL '+pickup.id+(x>936?' →':'');interactionPrompt(label,edge,315+pickup.id*19);}
      ctx.restore();
    }
    (run.drops||[]).forEach(drop => {
      if(drop.collected||drop.banked)return;
      const y=drop.y-(drop.elevation||0)-8+(renderer.reduced||!display.lootMotion?0:Math.sin(decorationTime/200)*3*motion),x=drop.x-camera;
      const legendary=window.RiftLoot.legendary(drop);
      if(legendary){ctx.strokeStyle='#ffc66d';ctx.lineWidth=3;ctx.beginPath();ctx.moveTo(x,y-25);ctx.lineTo(x+25,y);ctx.lineTo(x,y+25);ctx.lineTo(x-25,y);ctx.closePath();ctx.stroke();}
      if(display.lootSparkle)fx(5,0,x,y,34,.7);
      const icon=drop.gear?window.RiftLoot.icon(drop.gear.Slot):8,img=images.items,size=drop.gear?36:25;
      drawAtlas(img,icon%4*img.width/4,Math.floor(icon/4)*img.height/4,img.width/4,img.height/4,x-size/2,y-size/2,size,size);
    });
    for(const label of (!display.cleanScreenshot&&display.optionalCombatText)?window.RiftLoot.floorLabels((run.drops||[]).map(d=>({...d,y:d.y-(d.elevation||0)})),camera):[]){
      if(label.moved){ctx.strokeStyle='#80927b';ctx.lineWidth=1;ctx.beginPath();ctx.moveTo(label.drop.x-camera,label.drop.y-26);ctx.lineTo(label.x,label.y+17);ctx.stroke();}
      ctx.font='10px monospace';ctx.textAlign='center';ctx.fillStyle='#081914';ctx.fillRect(label.x-57,label.y,114,17);ctx.fillStyle=label.legendary?'#ffc66d':'#d7ecbb';ctx.fillText(label.text,label.x,label.y+12);
    }
    // The shaded cliff face is impassable; the outlined lower strip is safe ground.
    for(const edge of arena?.drop_edges||[]){
      const x=edge.x-camera,height=edge.landing_y-edge.y;
      ctx.save();
      ctx.fillStyle='#1b302d';ctx.fillRect(x,edge.y,edge.w,height);
      ctx.strokeStyle='#48625b';ctx.lineWidth=1;
      for(let row=12;row<height;row+=12){ctx.beginPath();ctx.moveTo(x,edge.y+row);ctx.lineTo(x+edge.w,edge.y+row);ctx.stroke();for(let col=(row%24?16:32);col<edge.w;col+=32){ctx.beginPath();ctx.moveTo(x+col,edge.y+row-12);ctx.lineTo(x+col,edge.y+row);ctx.stroke();}}
      ctx.fillStyle='#92b6a1';ctx.fillRect(x-2,edge.y-3,edge.w+4,5);
      ctx.fillStyle='#9df6d31a';ctx.fillRect(x,edge.landing_y,edge.w,18);
      ctx.strokeStyle='#a5e9ce';ctx.setLineDash([4,3]);ctx.strokeRect(x,edge.landing_y,edge.w,18);ctx.setLineDash([]);
      ctx.lineWidth=2;ctx.beginPath();ctx.moveTo(x+edge.w/2,edge.y+12);ctx.lineTo(x+edge.w/2,edge.landing_y-12);ctx.lineTo(x+edge.w/2-6,edge.landing_y-19);ctx.moveTo(x+edge.w/2,edge.landing_y-12);ctx.lineTo(x+edge.w/2+6,edge.landing_y-19);ctx.stroke();
      if(!display.cleanScreenshot){ctx.font='bold 10px monospace';ctx.textAlign='center';ctx.fillStyle='#caffeb';ctx.strokeStyle='#081914';ctx.lineWidth=3;interactionPrompt('DROP ↓',x+edge.w/2,edge.y-10);}
      ctx.restore();
    }
    const nearbyCover=window.RiftHUD.nearbyCover(run);
    const units=[...run.enemies,run.player];
    if(run.room_objective?.kind==='split_defense')units.push(...run.room_objective.lanes.map(l=>l.ward));
    if(run.room_objective?.kind==='protect_lantern')units.push(run.room_objective.lantern);
    if(run.room_objective?.kind==='escort_spirit')units.push(run.room_objective.escort);
    if(run.build?.class==='beastmaster'&&run.player.hp>0){
      const commandTarget=run.enemies.find(e=>e.id===run.marked&&e.hp>0);
      for(let i=0;i<Math.min(3,run.build?.pets||0);i++){
        const petX=run.player.x-run.player.facing*(55+i*36),facing=commandTarget?Math.sign(commandTarget.x-petX)||run.player.facing:run.player.facing;
        units.push({id:'pet'+i,kind:'wolf',x:petX,y:run.player.y+22+i*8,hp:1,max_hp:1,facing,pose:run.player.pose==='cast'?'cast':['run','guard_walk'].includes(run.player.pose)?'run':'idle',jump:0});
      }
    }
    (arena?.obstacles||[]).forEach(o=>units.push({y:o.y+o.h,cover:o}));
    (arena?.cover||[]).forEach(c=>units.push({y:c.y+c.h,terrain:c}));
    (arena?.high_cover||[]).forEach(o=>units.push({y:o.y+o.h,cover:o,tall:true}));
    for(const pickup of (run.room_objective?.kind==='sigils'?run.room_objective.pickups:[]))if(!pickup.collected)units.push({y:pickup.y,sigil:pickup});
    if(run.room_objective?.kind==='carry_relic'&&!run.room_objective.relic.collected)units.push({y:run.room_objective.relic.y,relic:run.room_objective.relic});
    units.sort((a,b)=>a.y-b.y).forEach(unit=>{
      if(unit.terrain){
        const c=unit.terrain,broken=c.material==='wood'&&c.hp<=0,index=c.material==='stone'?3:broken?2:c.hp<=c.max_hp/2?1:0;
        // Tight source bounds align the generated sprites to the actual footprint.
        const frames=[[48,107,542,434],[676,107,541,434],[16,906,600,242],[657,733,574,399]],frame=frames[index],width=c.w+12,height=broken?30:c.h+74;
        ctx.fillStyle='#03110a70';ctx.beginPath();ctx.ellipse(c.x+c.w/2-camera,c.y+c.h-2,c.w*.56,7,0,0,Math.PI*2);ctx.fill();
        const opacity=broken?1:foregroundCoverOpacity(c.x-6,c.y+c.h-height,width,height,c.y+c.h);
        ctx.save();ctx.globalAlpha=opacity;
        ctx.drawImage(images.terrainCover,...frame,c.x-6-camera,c.y+c.h-height,width,height);ctx.restore();fadedCoverFootprint(c,opacity,true);
        if(!display.cleanScreenshot&&!broken){const dx=c.x+c.w/2-run.player.x,dy=c.y+c.h/2-run.player.y,targeted=Math.abs(dx)<150&&Math.abs(dy)<65&&dx*run.player.facing>=-8;if(targeted){ctx.font='bold 10px monospace';ctx.textAlign='center';ctx.fillStyle='#f5e4c3';ctx.strokeStyle='#201a14';ctx.lineWidth=3;const label=c.material==='stone'?'STONE · PERMANENT':'WOOD '+Math.ceil(c.hp)+' / '+c.max_hp;interactionPrompt(label,c.x+c.w/2-camera,c.y+c.h-height-13);if(c.material==='wood'){ctx.fillStyle='#30241b';ctx.fillRect(c.x-camera,c.y+c.h-height-7,c.w,4);ctx.fillStyle='#dca766';ctx.fillRect(c.x-camera,c.y+c.h-height-7,c.w*c.hp/c.max_hp,4);}}}
        return;
      }
      if(unit.relic){ctx.drawImage(images.relic,unit.relic.x-camera-24,unit.relic.y-45,48,48);return;}
      if(unit.sigil){drawSigil(unit.sigil);return;}
      if(!unit.cover){actor(unit,wallNow);return;}
      const o=unit.cover,img=images.props,index=[0,1,2,3,4,5,6,3,3,7][run.level?.region||0],sw=img.width/4,sh=img.height/2;
      ctx.fillStyle='#03110a70';ctx.beginPath();ctx.ellipse(o.x+o.w/2-camera,o.y+o.h-3,o.w*.58,9,0,0,Math.PI*2);ctx.fill();
      // Each sprite's base lies at 90% of its atlas cell. Align it with
      // the collision footprint so jumping and circling cover read clearly.
      const width=o.w+14,height=o.h+(unit.tall?100:38);
      const opacity=foregroundCoverOpacity(o.x-7,o.y+o.h-height*.9,width,height,o.y+o.h);
      ctx.save();ctx.globalAlpha=opacity;
      drawAtlas(img,index%4*sw,Math.floor(index/4)*sh,sw,sh,o.x-7-camera,o.y+o.h-height*.9,width,height);ctx.restore();fadedCoverFootprint(o,opacity,unit.tall);
      if(!display.cleanScreenshot&&nearbyCover?.obstacle===o){ctx.save();ctx.strokeStyle=unit.tall?'#d6e5e9':'#a5e9ce';ctx.lineWidth=unit.tall?3:2;ctx.setLineDash(unit.tall?[]:[5,4]);ctx.strokeRect(o.x-camera,o.y,o.w,o.h);ctx.setLineDash([]);ctx.font='bold 10px monospace';ctx.textAlign='center';ctx.fillStyle=unit.tall?'#e2edf1':'#beffe4';ctx.strokeStyle='#102419';ctx.lineWidth=3;const label=unit.tall?'TALL · BLOCKS SHOTS':'LOW · VAULT';interactionPrompt(label,o.x+o.w/2-camera,o.y+o.h-height*.9-8);ctx.restore();}
    });
    run.projectiles.forEach(shot=>{
      const p={...shot,y:shot.y-(shot.elevation||0)};
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
        if(p.kind==='arrow'){ctx.save();ctx.translate(p.x-camera,p.y-30);ctx.rotate(projectileAngle(p.vx,p.vy,p.enemy));ctx.fillStyle='#d8b3e9';ctx.fillRect(-12,-1.5,22,3);ctx.beginPath();ctx.moveTo(16,0);ctx.lineTo(6,-6);ctx.lineTo(6,6);ctx.closePath();ctx.fillStyle='#fff0bb';ctx.fill();ctx.strokeStyle='#14221d';ctx.lineWidth=1.5;ctx.stroke();ctx.restore();}
        else if(p.kind==='pack')sprite(4,2+Math.floor(now/70)%4,p.x-camera,p.y,70,p.vx,.85,'mobs');
        else fx(effectRows[p.kind]??1,Math.floor(now/80)%3,p.x-camera,p.y-28,58,.95);
      }
    });
    effects=effects.filter(e=>now-e.started<750);
    effects.forEach(e=>{
      const age=(now-e.started)/750;
      if(e.kind==='class_entry'){
        const flourish=entryFlourishes[e.subclass],color=victoryStances[e.subclass].color,x=e.x-camera,y=e.y;
        const radius=renderer.reduced?29:18+age*22*motion,turn=renderer.reduced?0:age*.5*motion;
        ctx.save();ctx.strokeStyle=color;ctx.lineWidth=2;ctx.globalAlpha=(1-age)*display.effectIntensity;
        ctx.beginPath();
        for(let i=0;i<flourish.points;i++){const angle=i*Math.PI*2/flourish.points+turn,px=x+Math.cos(angle)*radius,py=y+Math.sin(angle)*radius*.35;if(i)ctx.lineTo(px,py);else ctx.moveTo(px,py);}
        ctx.closePath();ctx.stroke();ctx.restore();
        fx(flourish.row,renderer.reduced?2:Math.min(5,Math.floor(age*6)),x,y-30,72,(1-age)*.65*(flourish.row===5?display.effectIntensity:1));
        renderer.lastClassEntry={subclass:e.subclass,row:flourish.row,points:flourish.points,color,radius,turn,reduced:!!renderer.reduced};
      }
      if(e.kind==='projectile_expire'&&!renderer.reduced&&motion>0&&age<.4){
        const x=e.x-camera,y=e.y-28,progress=age/.4;
        if(x>=0&&x<=960&&y>=0&&y<=540){
          ctx.save();ctx.strokeStyle='#b9ccd1';ctx.lineWidth=1.5;ctx.globalAlpha=(1-progress)*.6*display.effectIntensity;
          ctx.beginPath();ctx.arc(x,y,3+progress*9,0,Math.PI*2);ctx.stroke();ctx.restore();
        }
      }
      if(e.kind==='projectile_impact'&&!renderer.reduced&&motion>0&&age<.4){
        const x=e.x-camera,y=e.y-28,progress=age/.4;
        if(x>=-24&&x<=984&&y>=-24&&y<=564){
          ctx.save();ctx.strokeStyle='#ffe1b5';ctx.lineWidth=2;ctx.globalAlpha=(1-progress)*display.effectIntensity;
          for(let i=0;i<6;i++){const angle=i*Math.PI/3,inner=4+progress*8,outer=inner+8*(1-progress);ctx.beginPath();ctx.moveTo(x+Math.cos(angle)*inner,y+Math.sin(angle)*inner);ctx.lineTo(x+Math.cos(angle)*outer,y+Math.sin(angle)*outer);ctx.stroke();}
          ctx.restore();
        }
      }
      if(e.kind==='slash'&&!renderer.reduced&&motion>0&&now-e.started<250){
        const progress=(now-e.started)/250;
        ctx.save();ctx.translate(e.x-camera,e.y);ctx.scale(e.visualFacing<0?-1:1,1);
        ctx.strokeStyle='#f7dfac';ctx.lineWidth=3;ctx.globalAlpha=(1-progress)*.8;
        ctx.beginPath();ctx.arc(0,0,32,-1.3+progress*.7,1.1+progress*.7);ctx.stroke();ctx.restore();
      }
      if(e.kind==='pickup'&&!renderer.reduced&&motion>0&&display.lootSparkle){
        const x=e.x-camera,y=e.y,targetX=run.player.x-camera,targetY=run.player.y-(run.player.elevation||0)-18;
        const midX=(x+targetX)/2,midY=Math.min(y,targetY)-28,t=Math.min(1,age*1.5),u=1-t;
        ctx.save();ctx.strokeStyle='#ffe9a6';ctx.lineWidth=2;ctx.globalAlpha=(1-age)*.65;
        ctx.beginPath();ctx.moveTo(x,y);ctx.quadraticCurveTo(midX,midY,targetX,targetY);ctx.stroke();
        ctx.fillStyle='#ffe9a6';ctx.beginPath();ctx.arc(u*u*x+2*u*t*midX+t*t*targetX,u*u*y+2*u*t*midY+t*t*targetY,3,0,Math.PI*2);ctx.fill();ctx.restore();
      }
      if(e.kind==='arrival'){
        const still=renderer.reduced||motion===0,radius=still?32:18+age*44;
        ctx.save();ctx.strokeStyle='#b8f2dc';ctx.lineWidth=2;ctx.globalAlpha=still?.65:(1-age)*.8;
        ctx.beginPath();ctx.ellipse(e.x-camera,e.y+4,radius,radius*.3,0,0,Math.PI*2);ctx.stroke();ctx.restore();
      }
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
      if(e.kind==='victory'){
        renderer.lastVictoryEvent = { x: e.x, y: e.y, age, started: e.started, reduced: !!renderer.reduced };
        const screenX = e.x - camera, screenY = e.y;
        const fade = Math.max(0, 1 - age * 1.8);
        if(fade > 0){
          ctx.save();
          // Expanding golden triumph rings
          const laurelR = 28 + age * 85;
          ctx.strokeStyle = 'rgba(255, 215, 0, ' + (fade * 0.9) + ')';
          ctx.lineWidth = Math.max(1, 3.5 * fade);
          ctx.beginPath();
          ctx.ellipse(screenX, screenY + 25, laurelR, laurelR * 0.42, 0, 0, Math.PI * 2);
          ctx.stroke();

          const innerR = 14 + age * 50;
          ctx.strokeStyle = 'rgba(255, 245, 180, ' + (fade * 0.95) + ')';
          ctx.lineWidth = Math.max(1, 2 * fade);
          ctx.beginPath();
          ctx.ellipse(screenX, screenY + 25, innerR, innerR * 0.42, 0, 0, Math.PI * 2);
          ctx.stroke();

          // Overhead herald banner: "🏆 VICTORY!"
          const bannerY = screenY - 55 - (renderer.reduced ? 0 : age * 18);
          const bannerFade = Math.max(0, 1 - age * 1.5);
          if(bannerFade > 0){
            ctx.font = 'bold ' + Math.round(14 * display.textScale) + 'px monospace';
            ctx.textAlign = 'center';
            ctx.strokeStyle = '#0a1715';
            ctx.lineWidth = 4;
            ctx.strokeText('🏆 VICTORY!', screenX, bannerY);
            ctx.fillStyle = '#ffe066';
            ctx.fillText('🏆 VICTORY!', screenX, bannerY);
          }

          if(!renderer.reduced){
            // Radiant laurel celebration rays
            const rays = 8;
            for(let r = 0; r < rays; r++){
              const ang = r * (Math.PI * 2 / rays) + age * 2.2;
              const innerDist = 20 + age * 30;
              const outerDist = innerDist + 18 * (1 - age * 0.7);
              const rx1 = screenX + Math.cos(ang) * innerDist;
              const ry1 = (screenY - 10) + Math.sin(ang) * (innerDist * 0.55);
              const rx2 = screenX + Math.cos(ang) * outerDist;
              const ry2 = (screenY - 10) + Math.sin(ang) * (outerDist * 0.55);
              ctx.strokeStyle = 'rgba(255, 230, 100, ' + (fade * 0.8) + ')';
              ctx.lineWidth = Math.max(1, 2 * fade);
              ctx.beginPath();
              ctx.moveTo(rx1, ry1);
              ctx.lineTo(rx2, ry2);
              ctx.stroke();
            }

            // Ascending celebration sparkle motes
            const motes = 8;
            for(let m = 0; m < motes; m++){
              const progress = (age * 2.8 + m / motes) % 1;
              const mx = screenX + Math.sin(m * 2.2 + age * 5) * 32;
              const my = (screenY + 20) - progress * 75;
              const mFade = fade * Math.sin(progress * Math.PI);
              if(mFade > 0){
                ctx.fillStyle = m % 2 === 0 ? 'rgba(255, 225, 100, ' + mFade + ')' : 'rgba(255, 255, 220, ' + mFade + ')';
                ctx.beginPath();
                ctx.arc(mx, my, 2.5 * (1 - progress * 0.4), 0, Math.PI * 2);
                ctx.fill();
              }
            }
          }
          ctx.restore();
        }
      }
      if(e.kind==='geomancer_terrain'){
        const radius=renderer.reduced?28:12+age*48*motion;ctx.save();ctx.strokeStyle='#e7c18b';ctx.globalAlpha=Math.max(0,1-age);ctx.lineWidth=2;ctx.beginPath();ctx.ellipse(e.x-camera,e.y,radius,radius*.4,0,0,Math.PI*2);ctx.moveTo(e.x-camera-8,e.y);ctx.lineTo(e.x-camera,e.y-9);ctx.lineTo(e.x-camera+8,e.y);ctx.lineTo(e.x-camera,e.y+9);ctx.closePath();ctx.stroke();ctx.restore();
      }
      if(effectRows[e.kind]!==undefined && e.kind!=='third_strike' && e.kind!=='finisher_cast' && e.kind!=='ultimate_anticipation' && e.kind!=='heavy_recovery' && e.kind!=='shield_absorb' && e.kind!=='mark_target' && e.kind!=='thaw' && e.kind!=='boss_stagger' && e.kind!=='boss_phase' && e.kind!=='victory' && (!renderer.reduced && (e.kind!=='pickup'||display.lootSparkle)))fx(effectRows[e.kind],Math.min(5,Math.floor(age*6)),e.x-camera,e.y,['slam','quake','ultimate'].includes(e.kind)?240:95,1-age*.5);
      if(e.kind==='pickup' && (renderer.reduced || !display.lootSparkle))drawStaticPickup(ctx,e.x-camera,e.y);
      if(!display.cleanScreenshot && (e.value>0 || e.kind==='block' || e.kind==='perfect_guard' || e.kind==='treasure_escape' || e.kind==='rare_item' || e.kind==='rare_discovery') && e.kind!=='area' && !e.kind.endsWith('_hurt') && e.kind!=='slash' && e.kind!=='third_strike' && e.kind!=='finisher_cast' && e.kind!=='ultimate_anticipation' && e.kind!=='heavy_recovery' && e.kind!=='shield_absorb' && e.kind!=='mark_target' && e.kind!=='thaw' && e.kind!=='boss_stagger' && e.kind!=='boss_phase' && e.kind!=='victory' && (['elemental_reaction','beacon_captured','sigil_pickup','pickup','resource','heal','barrier','treasure_escape','rare_item','rare_discovery'].includes(e.kind)?display.optionalCombatText:display.damageNumbers)){
        ctx.font='bold '+(13*display.textScale)+'px monospace';
        ctx.textAlign='center';
        const {color,label}=combatTextProperties(e);
        ctx.fillStyle=color;ctx.strokeStyle='#14221d';ctx.lineWidth=3;
        const drift = (renderer.reduced || !display.damageMotion) ? 0 : age*38*motion;
        ctx.strokeText(label,e.x-camera,e.y-drift);
        ctx.fillText(label,e.x-camera,e.y-drift);
      }
    });
    // Preserve the danger footprint above large sprites without covering their bodies.
    for(const h of hazardOverlays){
      ctx.save();ctx.globalAlpha=1;ctx.setLineDash(h.warning?[8,4]:[]);
      ctx.strokeStyle='#071b16';ctx.lineWidth=5;ctx.strokeRect(h.x,h.y,h.w,h.h);
      ctx.strokeStyle=display.hazardContrast?'#fff8d8':h.color;ctx.lineWidth=display.hazardContrast?3:2;ctx.strokeRect(h.x,h.y,h.w,h.h);ctx.restore();
    }
    drawInteractionPrompts();
    const isMovingFootstep = (run.player.pose === 'run' && now - footstep > 320) || (run.player.pose === 'guard_walk' && now - footstep > 460);
    if(run.status==='fighting' && !run.paused && isMovingFootstep && run.player.jump===0){const floorMat=run.floor||run.level?.rooms?.[run.room]?.floor||'stone';if(window.RiftAudio.step)window.RiftAudio.step(floorMat,0);else window.RiftAudio.play('step',0);footstep=now;}
    window.RiftAudio.tick();
    if(now-transitionAt<500&&!renderer.reduced&&display.flashIntensity>0){
      const fade=Math.max(0,1-(now-transitionAt)/500)*display.flashIntensity;
      const color=/^#[0-9a-f]{6}$/i.test(run.level?.color||'')?run.level.color:'#a6ce7b';
      ctx.save();ctx.fillStyle='#091914';ctx.globalAlpha=.65*fade;ctx.fillRect(0,0,960,540);
      ctx.fillStyle=color;ctx.globalAlpha=.12*fade;ctx.fillRect(0,0,960,540);
      ctx.strokeStyle=color;ctx.globalAlpha=.8*fade;ctx.lineWidth=4;ctx.strokeRect(6,6,948,528);ctx.restore();
    }
  }
  renderer.ready.then(()=>requestAnimationFrame(render)).catch(()=>{});
  window.RiftRenderer=renderer;
})();
