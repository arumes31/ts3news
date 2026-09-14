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
  const renderer = { reduced: window.matchMedia('(prefers-reduced-motion: reduce)').matches, ready: null, frameCount: 0 };
  renderer.build = build => { previewStyle = build.class; };
  renderer.preview = level => { previewLevel = level; };
  const baseImages = Promise.all(['area','boss','regions','props','heroesA','heroesB','mobs','items','effects'].map(key => new Promise((resolve, reject) => {
    const img = new Image(); img.onload = () => { images[key] = img; resolve(); }; img.onerror = () => reject(new Error('Could not load '+key+' artwork. Reload to try again.')); img.src = key==='props'?document.getElementById('rift-props-asset').href:root.dataset[key];
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
    if (unit.pose === 'hit') col = 12;
    if (unit.knockdown > 0) col = 13;
    if (unit.id === 'player' && snapshot.status === 'cleared' && unit.pose !== 'run') col = 15;
    const jump = unit.jump > 0 ? Math.sin((.65-unit.jump)/.65*Math.PI)*52 : 0;
    if(shared)catalogActor(unit,unit.pose,x-camera,y-jump,size,1);else sprite(row,col,x-camera,y-jump,size,unit.facing,1,atlas);
    if (unit.guard || unit.id === 'player' && snapshot.barrier > 0) fx(3,1,x-camera,y-size*.4,80,.55);
    if (unit.id !== 'player' && unit.kind !== 'wolf') {
      if(display.healthBars){ctx.fillStyle='#0a1715dc'; ctx.fillRect(x-camera-24,y-size*.9-8,48,5);ctx.fillStyle=unit.kind==='boss'?'#e9a35c':'#bc7055'; ctx.fillRect(x-camera-23,y-size*.9-7,46*unit.hp/unit.max_hp,3);}
      if(unit.art_key&&(display.enemyNames==='all'||display.enemyNames==='boss'&&unit.kind==='boss')){ctx.font=(10*display.textScale)+'px monospace';ctx.textAlign='center';ctx.fillStyle='#e9efce';ctx.strokeStyle='#0a1715';ctx.lineWidth=3;ctx.strokeText(unit.name,x-camera,y-size*.9-14);ctx.fillText(unit.name,x-camera,y-size*.9-14);}
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
    if(run.practice&&['movement','jump'].includes(run.practice.mode)){
      ctx.save();ctx.strokeStyle='#e3f9ac';ctx.lineWidth=4;ctx.setLineDash([10,7]);ctx.beginPath();ctx.moveTo(run.practice.goal_x-camera,250);ctx.lineTo(run.practice.goal_x-camera,535);ctx.stroke();ctx.setLineDash([]);ctx.fillStyle='#e3f9ac';ctx.font='bold 14px monospace';ctx.textAlign='center';ctx.fillText('FINISH',run.practice.goal_x-camera,240);ctx.restore();
    }
    (arena?.hazards||[]).forEach(h=>{
      const phase=(run.clock+h.offset)%h.period, warning=phase<1.2, active=phase>=1.2&&phase<1.2+h.duration&&run.status==='fighting';
      const x=h.x-camera,color={fire:'#ff9a52',ice:'#9be5ff',rune:'#d1acff',poison:'#c7ee76',thorns:'#b5d780',radiant:'#d7dfff',void:'#b194ff'}[h.kind]||'#ffbf70';
      ctx.save();ctx.fillStyle=color;ctx.strokeStyle=color;ctx.lineWidth=active?3:1;ctx.globalAlpha=active?.55:warning?.18:.06;ctx.fillRect(x,h.y,h.w,h.h);ctx.globalAlpha=active?1:warning?.7:.2;
      ctx.setLineDash(warning?[5,4]:[]);ctx.strokeRect(x,h.y,h.w,h.h);ctx.setLineDash([]);
      if(display.hazardContrast&&(warning||active)){ctx.globalAlpha=1;ctx.strokeStyle='#fff8d8';ctx.lineWidth=3;ctx.setLineDash(active?[]:[8,4]);ctx.strokeRect(x-2,h.y-2,h.w+4,h.h+4);ctx.setLineDash([]);}
      if(display.hazardLabels&&(warning||active)){ctx.font='bold 9px monospace';ctx.textAlign='center';ctx.fillText(active?'JUMP':h.kind.toUpperCase(),x+h.w/2,h.y-5);}
      if(active&&!renderer.reduced)fx(effectRows[h.kind]??3,Math.floor(now/90)%6,x+h.w/2,h.y+h.h/2,h.w,.7);
      ctx.restore();
    });
    run.enemies.forEach(e => {
      if(e.hp>0 && e.windup>0 && e.kind==='boss') {
        ctx.fillStyle='#c8783b55';ctx.strokeStyle='#ffce7d';ctx.lineWidth=2;ctx.beginPath();ctx.ellipse(e.target_x-camera,e.target_y,125,62,0,0,Math.PI*2);ctx.fill();ctx.stroke();
        ctx.fillStyle='#ffe2b0';ctx.font='bold 12px monospace';ctx.textAlign='center';ctx.fillText('JUMP OR MOVE',e.target_x-camera,e.target_y+4);
      }
    });
    (run.drops||[]).forEach(drop => {
      if(drop.collected||drop.banked)return;
      const y=drop.y-8+(renderer.reduced||!display.lootMotion?0:Math.sin(decorationTime/200)*3*motion),x=drop.x-camera;
      const legendary=window.RiftLoot.legendary(drop);
      if(legendary){ctx.strokeStyle='#ffc66d';ctx.lineWidth=3;ctx.beginPath();ctx.moveTo(x,y-25);ctx.lineTo(x+25,y);ctx.lineTo(x,y+25);ctx.lineTo(x-25,y);ctx.closePath();ctx.stroke();}
      if(display.lootSparkle)fx(5,0,x,y,34,.7);
      const icon=drop.gear?window.RiftLoot.icon(drop.gear.Slot):8,img=images.items,size=drop.gear?36:25;
      ctx.drawImage(img,icon%4*img.width/4,Math.floor(icon/4)*img.height/4,img.width/4,img.height/4,x-size/2,y-size/2,size,size);
    });
    for(const label of window.RiftLoot.floorLabels(run.drops||[],camera)){
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
      if(p.kind==='arrow'){ctx.fillStyle='#d8b3e9';ctx.fillRect(p.x-camera-12,p.y-30,25,3);}
      else if(p.kind==='pack')sprite(4,2+Math.floor(now/70)%4,p.x-camera,p.y,70,p.vx,.85,'mobs');
      else fx(effectRows[p.kind]??1,Math.floor(now/80)%3,p.x-camera,p.y-28,58,.95);
    });
    effects=effects.filter(e=>now-e.started<750);
    effects.forEach(e=>{
      const age=(now-e.started)/750;
      if(effectRows[e.kind]!==undefined && (!renderer.reduced || e.kind==='pickup') && (e.kind!=='pickup'||display.lootSparkle))fx(effectRows[e.kind],Math.min(5,Math.floor(age*6)),e.x-camera,e.y,['slam','quake','ultimate'].includes(e.kind)?240:95,1-age*.5);
      if(e.value>0 && e.kind!=='area' && (display.damageNumbers||e.kind==='pickup')){ctx.font='bold '+(14*display.textScale)+'px monospace';ctx.textAlign='center';ctx.fillStyle=e.kind==='hurt'?'#ffb2a0':'#fff0bb';ctx.strokeStyle='#14221d';ctx.lineWidth=3;const label=e.kind==='pickup'?'+'+Math.round(e.value)+' gold':String(Math.round(e.value));ctx.strokeText(label,e.x-camera,e.y-(display.damageMotion?age*38*motion:0));ctx.fillText(label,e.x-camera,e.y-(display.damageMotion?age*38*motion:0));}
    });
    if(run.status==='fighting' && !run.paused && run.player.pose==='run' && run.player.jump===0 && now-footstep>320){window.RiftAudio.play('step',0);footstep=now;}
    window.RiftAudio.tick();
    if(now-transitionAt<500&&!renderer.reduced){ctx.fillStyle='#091914';ctx.globalAlpha=Math.max(0,.65*(1-(now-transitionAt)/500))*display.flashIntensity;ctx.fillRect(0,0,960,540);ctx.globalAlpha=1;}
  }
  renderer.ready.then(()=>requestAnimationFrame(render)).catch(()=>{});
  window.RiftRenderer=renderer;
})();
