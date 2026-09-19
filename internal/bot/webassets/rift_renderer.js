(function () {
  'use strict';
  const root = document.getElementById('rift-app'), canvas = document.getElementById('rift-canvas'), ctx = canvas.getContext('2d');
  const images = {}, effectRows = { slash:0, hit:0, fire:1, slam:1, quake:1, ice:2, shield:3, heal:3, block:3, radiant:3, rune:3, void:4, poison:4, ultimate:4, pack:2, pickup:5, clear:5 };
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
    else if(e.kind==='block'){color='#7fd7ff';label='🛡️ Guarded'+(Math.round(e.value||0)>0?' -'+Math.round(e.value):'');}
    else if(e.kind==='pickup'){color='#ffe082';label='+'+Math.round(e.value)+' gold';}
    else if(e.kind==='resource'){color='#7ef5d0';label='+'+Math.round(e.value)+' '+(snapshot?.build?.resource||'Charge');}
    else if(e.kind==='heal'){color='#a8f0b0';label='+'+Math.round(e.value)+' HP';}
    else if(e.kind==='barrier'){color='#c6a8f8';label='+'+Math.round(e.value)+' Barrier';}
    return {color,label};
  }
  renderer.combatText = combatTextProperties;
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
    const changed = runID !== run.id || snapshot && run.counter < snapshot.counter;
    if (changed) { impactAt=-Infinity; runID = run.id; seen = replay ? run.counter : 0; effects = []; previous = null; deaths.clear(); }
    else previous = snapshot;
    if (previous && (previous.room !== run.room || previous.level?.id !== run.level?.id)) { previous = null; effects = []; deaths.clear(); camera=0; transitionAt=animationTime; }
    snapshot = run; received = performance.now();
    [run.player,...run.enemies].forEach(unit=>{if(unit.hp<=0&&!deaths.has(unit.id))deaths.set(unit.id,replay?animationTime-1000:animationTime);});
    (run.events || []).forEach(event => {
      if (event.id <= seen) return;
      seen = event.id;
      if(!replay&&(event.kind==='slam'||event.kind==='hurt'&&event.value>0))impactAt=performance.now();
      if (event.kind !== 'area') effects.push({ ...event, started: animationTime });
      window.RiftAudio.play(event.kind, (event.x - run.player.x) / 700);
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
    ctx.fillStyle='#03110b70'; ctx.beginPath(); ctx.ellipse(x-camera,y+2,size*.28,7,0,0,Math.PI*2); ctx.fill();
    if (unit.hp <= 0) { const fallen=animationTime-(deaths.get(unit.id)??0);if(shared)catalogActor(unit,'defeat',x-camera,y,size,fallen<700?.85:.4);else sprite(row,fallen<320?13:14,x-camera,y,size,unit.facing,fallen<700?.85:.4,atlas);return; }
    let col = renderer.reduced ? 0 : Math.floor(decorationTime/650)%2;
    if (unit.pose === 'run') col = 2+Math.floor(animationTime/105)%4;
    if (unit.pose === 'attack') col = unit.pose_time > .25 ? 8 : unit.pose_time > .12 ? 9 : 10;
    if (unit.pose === 'cast') col = 11;
    if (unit.pose === 'windup') col = 8;
    if (unit.guard) col = 7;
    if (unit.jump > 0) col = 6;
    if (unit.pose === 'hit' && !unit.guard) col = 12;
    if (unit.knockdown > 0) col = 13;
    if (unit.id === 'player' && snapshot.status === 'cleared' && unit.pose !== 'run') col = 15;
    const jump = unit.jump > 0 ? Math.sin((.65-unit.jump)/.65*Math.PI)*52 : 0;
    if(shared)catalogActor(unit,unit.pose,x-camera,y-jump,size,1);else sprite(row,col,x-camera,y-jump,size,unit.facing,1,atlas);
    if (unit.guard || unit.id === 'player' && snapshot.barrier > 0) fx(3,1,x-camera,y-size*.4,80,.55);
    if (unit.guard && unit.pose === 'hit') fx(3,2,x-camera,y-size*.4,105,.85);
    if (unit.id !== 'player' && unit.kind !== 'wolf') {
      if(!display.cleanScreenshot&&display.healthBars){
        ctx.fillStyle='#0a1715dc'; ctx.fillRect(x-camera-24,y-size*.9-8,48,5);
        ctx.fillStyle=unit.kind==='boss'?'#e9a35c':'#bc7055'; ctx.fillRect(x-camera-23,y-size*.9-7,46*unit.hp/unit.max_hp,3);
        ctx.fillStyle='rgba(255,255,255,0.45)';
        ctx.fillRect(Math.round(x-camera-23+46*0.25),Math.round(y-size*.9-7),1,3);
        ctx.fillRect(Math.round(x-camera-23+46*0.50),Math.round(y-size*.9-7),1,3);
        if(unit.hp/unit.max_hp<=0.25){
          ctx.strokeStyle='#ffd79e'; ctx.lineWidth=1;
          ctx.strokeRect(x-camera-24.5,y-size*.9-8.5,49,6);
        }
      }
      if(!display.cleanScreenshot&&unit.art_key&&(display.enemyNames==='all'||display.enemyNames==='boss'&&unit.kind==='boss')){const displayName=(unit.kind==='boss'&&unit.name&&unit.name.length>26)?unit.name.slice(0,24)+'…':unit.name;const textY=y-size*.9-18;ctx.font=(10*display.textScale)+'px monospace';ctx.textAlign='center';ctx.fillStyle='#e9efce';ctx.strokeStyle='#0a1715';ctx.lineWidth=3;ctx.strokeText(displayName,x-camera,textY);ctx.fillText(displayName,x-camera,textY);}
      if(!display.cleanScreenshot&&unit.kind==='boss'&&unit.windup>0){const attack=unit.attack_name||(unit.art_key&&(unit.attacks+1)%2===0?'Aimed Volley':'Ground Slam');const attackY=y-size*.9-30;ctx.font='bold '+(10*display.textScale)+'px monospace';ctx.textAlign='center';ctx.fillStyle='#ffe599';ctx.strokeStyle='#0a1715';ctx.lineWidth=3;ctx.strokeText('⚡ '+attack+' ('+unit.windup.toFixed(1)+'s)',x-camera,attackY);ctx.fillText('⚡ '+attack+' ('+unit.windup.toFixed(1)+'s)',x-camera,attackY);}
      if(snapshot.marked===unit.id){ctx.fillStyle='#8fe1cc';ctx.beginPath();ctx.moveTo(x-camera,y-size-12);ctx.lineTo(x-camera-4,y-size-18);ctx.lineTo(x-camera+4,y-size-18);ctx.fill();}
    }
  }
  function catalogActor(unit,pose,x,y,size,alpha){
    const profile=bestiary.profile(unit);
    // Keep the expanded Brawl animations for matching existing species. All
    // other anatomy comes directly from Abyss's shared actor-frame provider.
    const localRow={goblin:0,wolf:4,knight:2}[profile.rig];
    let mapped=pose==='hit'?'hurt':pose==='windup'?'cast':pose==='knockdown'?'defeat':pose;
    if(localRow!==undefined){
      let col=renderer.reduced?0:Math.floor(decorationTime/650)%2;
      if(pose==='run')col=2+Math.floor(animationTime/105)%4;
      if(pose==='attack')col=unit.pose_time>.25?8:unit.pose_time>.12?9:10;
      if(pose==='windup')col=8;if(pose==='cast')col=11;if(pose==='hit')col=12;if(pose==='knockdown')col=13;if(pose==='defeat')col=14;
      sprite(localRow,col,x,y,size,unit.facing,alpha,'mobs');
    }else{
      const frame=bestiary.frame(unit,mapped,Math.floor((mapped==='idle'?decorationTime:animationTime)/(pose==='run'?110:200))),img=catalogImages[frame.asset],source=frame.source;
      if(!img||!source)return;
      const stride=pose==='run'&&!renderer.reduced?Math.sin(animationTime/65)*3*motion:0;
      ctx.save();ctx.globalAlpha=alpha;ctx.translate(Math.round(x),Math.round(y+stride));ctx.scale(unit.facing<0?-1:1,1);
      if(pose==='knockdown')ctx.rotate(-.55);
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
    if(run.build.class==='beastmaster'&&run.player.hp>0){for(let i=0;i<Math.min(3,run.build.pets||0);i++)units.push({id:'pet'+i,kind:'wolf',x:run.player.x-run.player.facing*(55+i*36),y:run.player.y+22+i*8,hp:1,max_hp:1,facing:run.player.facing,pose:run.player.pose==='cast'?'cast':run.player.pose==='run'?'run':'idle',jump:0});}
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
      if(effectRows[e.kind]!==undefined && (!renderer.reduced || e.kind==='pickup') && (e.kind!=='pickup'||display.lootSparkle))fx(effectRows[e.kind],Math.min(5,Math.floor(age*6)),e.x-camera,e.y,['slam','quake','ultimate'].includes(e.kind)?240:95,1-age*.5);
      if(!display.cleanScreenshot && (e.value>0 || e.kind==='block') && e.kind!=='area' && (['pickup','resource','heal','barrier'].includes(e.kind)?display.optionalCombatText:display.damageNumbers)){
        ctx.font='bold '+(13*display.textScale)+'px monospace';
        ctx.textAlign='center';
        const {color,label}=combatTextProperties(e);
        ctx.fillStyle=color;ctx.strokeStyle='#14221d';ctx.lineWidth=3;
        ctx.strokeText(label,e.x-camera,e.y-(display.damageMotion?age*38*motion:0));
        ctx.fillText(label,e.x-camera,e.y-(display.damageMotion?age*38*motion:0));
      }
    });
    if(run.status==='fighting' && !run.paused && run.player.pose==='run' && run.player.jump===0 && now-footstep>320){window.RiftAudio.play('step',0);footstep=now;}
    window.RiftAudio.tick();
    if(now-transitionAt<500&&!renderer.reduced){ctx.fillStyle='#091914';ctx.globalAlpha=Math.max(0,.65*(1-(now-transitionAt)/500))*display.flashIntensity;ctx.fillRect(0,0,960,540);ctx.globalAlpha=1;}
  }
  renderer.ready.then(()=>requestAnimationFrame(render)).catch(()=>{});
  window.RiftRenderer=renderer;
})();
