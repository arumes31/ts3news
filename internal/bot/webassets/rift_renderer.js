(function () {
  'use strict';
  const root = document.getElementById('rift-app'), canvas = document.getElementById('rift-canvas'), ctx = canvas.getContext('2d');
  const images = {}, effectRows = { slash:0, hit:0, fire:1, slam:1, quake:1, ice:2, shield:3, heal:3, block:3, radiant:3, rune:3, void:4, poison:4, ultimate:4, pack:2, pickup:5, clear:5 };
  const styles = ['vanguard','berserker','marksman','beastmaster','elementalist','chronomancer','oracle','geomancer','bloodblade','voidwalker','runesmith','alchemist'];
  const foundations = {warrior:'vanguard',ranger:'marksman',arcanist:'elementalist',warden:'oracle',reaver:'bloodblade',artificer:'runesmith'};
  const deaths = new Map();
  let animationTime = 0;
  let previewStyle = 'vanguard';
  let snapshot = null, previous = null, received = 0, camera = 0, seen = 0, runID = '', effects = [], last = 0, footstep = 0;
  const renderer = { reduced: window.matchMedia('(prefers-reduced-motion: reduce)').matches, ready: null, frameCount: 0 };
  renderer.build = build => { previewStyle = build.class; };
  renderer.ready = Promise.all(['area','boss','heroesA','heroesB','mobs','items','effects'].map(key => new Promise((resolve, reject) => {
    const img = new Image(); img.onload = () => { images[key] = img; resolve(); }; img.onerror = () => reject(new Error('Could not load '+key+' artwork. Reload to try again.')); img.src = root.dataset[key];
  })));
  renderer.snapshot = function (run, replay) {
    if (!run) return;
    const changed = runID !== run.id;
    if (changed) { runID = run.id; seen = replay ? run.counter : 0; effects = []; previous = null; deaths.clear(); }
    else previous = snapshot;
    if (previous && previous.room !== run.room) { previous = null; effects = []; deaths.clear(); }
    snapshot = run; received = performance.now();
    [run.player,...run.enemies].forEach(unit=>{if(unit.hp<=0&&!deaths.has(unit.id))deaths.set(unit.id,replay?animationTime-1000:animationTime);});
    (run.events || []).forEach(event => {
      if (event.id <= seen) return;
      seen = event.id;
      if (event.kind !== 'area') effects.push({ ...event, started: animationTime });
      window.RiftAudio.play(event.kind, (event.x - run.player.x) / 700);
    });
    if (effects.length > 40) effects = effects.slice(-40);
    window.RiftAudio.area(run.room);
  };
  function sprite(row, col, x, y, size, flip, alpha, atlas = 'heroesA') {
    const img = images[atlas]; if (!img) return;
    ctx.save(); ctx.globalAlpha = alpha === undefined ? 1 : alpha; ctx.translate(Math.round(x),Math.round(y)); ctx.scale(flip < 0 ? -1 : 1,1);
    ctx.drawImage(img,col*img.width/16,row*img.height/6,img.width/16,img.height/6,-size/2,-size*.91,size,size); ctx.restore();
  }
  function fx(row, frame, x, y, size, alpha) {
    const img = images.effects; if (!img) return;
    ctx.save(); ctx.globalAlpha = alpha; ctx.drawImage(img,frame*img.width/6,row*img.height/6,img.width/6,img.height/6,Math.round(x-size/2),Math.round(y-size/2),size,size); ctx.restore();
  }
  function actor(unit, now) {
    const index = Math.max(0,styles.indexOf(foundations[unit.kind] || unit.kind));
    const atlas = unit.id === 'player' ? (index < 6 ? 'heroesA' : 'heroesB') : 'mobs';
    const row = unit.id === 'player' ? index%6 : ({goblin:0,archer:1,knight:2,boss:3,wolf:4,spore:5}[unit.kind] ?? 0);
    const size = unit.kind === 'boss' ? 168 : unit.kind === 'goblin' ? 80 : unit.kind === 'wolf' ? 63 : 101;
    let x = unit.x, y = unit.y;
    if (previous && snapshot && unit.hp > 0) {
      const old = unit.id === 'player' ? previous.player : previous.enemies.find(e => e.id === unit.id);
      const t = Math.min(1,(now-received)/110);
      if (old) { x = old.x+(x-old.x)*t; y = old.y+(y-old.y)*t; }
    }
    ctx.fillStyle='#03110b70'; ctx.beginPath(); ctx.ellipse(x-camera,y+2,size*.28,7,0,0,Math.PI*2); ctx.fill();
    if (unit.hp <= 0) { const fallen=animationTime-(deaths.get(unit.id)??0);sprite(row,fallen<320?13:14,x-camera,y,size,unit.facing,fallen<700?.85:.4,atlas);return; }
    let col = renderer.reduced ? 0 : Math.floor(animationTime/650)%2;
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
    sprite(row,col,x-camera,y-jump,size,unit.facing,1,atlas);
    if (unit.guard || unit.id === 'player' && snapshot.barrier > 0) fx(3,1,x-camera,y-size*.4,80,.55);
    if (unit.id !== 'player' && unit.kind !== 'wolf') {
      ctx.fillStyle='#0a1715dc'; ctx.fillRect(x-camera-24,y-size*.9-8,48,5);
      ctx.fillStyle=unit.kind==='boss'?'#e9a35c':'#bc7055'; ctx.fillRect(x-camera-23,y-size*.9-7,46*unit.hp/unit.max_hp,3);
      if(snapshot.marked===unit.id){ctx.fillStyle='#8fe1cc';ctx.beginPath();ctx.moveTo(x-camera,y-size-12);ctx.lineTo(x-camera-4,y-size-18);ctx.lineTo(x-camera+4,y-size-18);ctx.fill();}
    }
  }
  function render(now) {
    requestAnimationFrame(render);
    if (!images.area || document.hidden || !ctx) return;
    renderer.frameCount++;
    ctx.imageSmoothingEnabled = false;
    const dt = Math.min(.05,(now-last)/1000); last = now;
    if (!snapshot || !snapshot.paused) animationTime += dt*1000;
    const wallNow = now; now = animationTime;
    const targetCamera = snapshot ? Math.max(0,Math.min(640,snapshot.player.x-350)) : 220;
    camera += (targetCamera-camera)*Math.min(1,dt*8);
    // Slow background parallax retains the full walkable foreground.
    const background = snapshot && snapshot.room === 2 && images.boss ? images.boss : images.area;
    ctx.drawImage(background,0,0,background.width,background.height,-camera*.35,0,1184,540);
    if (snapshot && snapshot.room === 1) { ctx.fillStyle='#61532316';ctx.fillRect(0,0,960,540); }
    if (!renderer.reduced) {
      for(let i=0;i<22;i++) { const x=(i*157+now*.004*(i%3+1))%1000; const y=80+(i*41)%300+Math.sin(now*.0005+i)*14; ctx.globalAlpha=.3+Math.sin(now*.001+i)*.2; ctx.fillStyle=i%3?'#a9ce8c':'#ffd98a';ctx.fillRect(x,y,2,2); }
      ctx.globalAlpha=1;
    }
    if (!snapshot) { const index=Math.max(0,styles.indexOf(foundations[previewStyle]||previewStyle));sprite(index%6,renderer.reduced?0:Math.floor(now/650)%2,630,400,113,-1,1,index<6?'heroesA':'heroesB');return; }
    const run=snapshot;
    run.enemies.forEach(e => {
      if(e.hp>0 && e.windup>0 && e.kind==='boss') {
        ctx.fillStyle='#c8783b55';ctx.strokeStyle='#ffce7d';ctx.lineWidth=2;ctx.beginPath();ctx.ellipse(e.target_x-camera,e.target_y,125,62,0,0,Math.PI*2);ctx.fill();ctx.stroke();
        ctx.fillStyle='#ffe2b0';ctx.font='bold 12px monospace';ctx.textAlign='center';ctx.fillText('JUMP OR MOVE',e.target_x-camera,e.target_y+4);
      }
    });
    (run.drops||[]).forEach(drop => {
      if(drop.collected)return;
      const y=drop.y-8+(renderer.reduced?0:Math.sin(now/200)*3),x=drop.x-camera;
      fx(5,0,x,y,34,.7);
      const slots={weapon:0,offhand:1,head:4,helmet:4,chest:5,armor:5,feet:6,boots:6,hands:7,gloves:7,ring:12,amulet:13,relic:15};
      const icon=drop.gear?(slots[String(drop.gear.Slot).toLowerCase()]??9):8,img=images.items,size=drop.gear?36:25;
      ctx.drawImage(img,icon%4*img.width/4,Math.floor(icon/4)*img.height/4,img.width/4,img.height/4,x-size/2,y-size/2,size,size);
      if(drop.gear){ctx.font='10px monospace';ctx.textAlign='center';ctx.fillStyle='#081914';ctx.fillRect(x-57,y-33,114,17);ctx.fillStyle='#d7ecbb';ctx.fillText('ABYSS GEAR',x,y-21);}
    });
    const units=[...run.enemies,run.player];
    if(run.build.class==='beastmaster'&&run.player.hp>0){for(let i=0;i<Math.min(3,run.build.pets||0);i++)units.push({id:'pet'+i,kind:'wolf',x:run.player.x-run.player.facing*(55+i*36),y:run.player.y+22+i*8,hp:1,max_hp:1,facing:run.player.facing,pose:run.player.pose==='cast'?'cast':run.player.pose==='run'?'run':'idle',jump:0});}
    units.sort((a,b)=>a.y-b.y).forEach(unit=>actor(unit,wallNow));
    run.projectiles.forEach(p=>{
      if(p.kind==='arrow'){ctx.fillStyle='#d8b3e9';ctx.fillRect(p.x-camera-12,p.y-30,25,3);}
      else if(p.kind==='pack')sprite(4,2+Math.floor(now/70)%4,p.x-camera,p.y,70,p.vx,.85,'mobs');
      else fx(effectRows[p.kind]??1,Math.floor(now/80)%3,p.x-camera,p.y-28,58,.95);
    });
    effects=effects.filter(e=>now-e.started<750);
    effects.forEach(e=>{
      const age=(now-e.started)/750;
      if(effectRows[e.kind]!==undefined && (!renderer.reduced || e.kind==='pickup'))fx(effectRows[e.kind],Math.min(5,Math.floor(age*6)),e.x-camera,e.y,['slam','quake','ultimate'].includes(e.kind)?240:95,1-age*.5);
      if(e.value>0 && e.kind!=='area'){ctx.font='bold 14px monospace';ctx.textAlign='center';ctx.fillStyle=e.kind==='hurt'?'#ffb2a0':'#fff0bb';ctx.strokeStyle='#14221d';ctx.lineWidth=3;const label=e.kind==='pickup'?'+'+Math.round(e.value):String(Math.round(e.value));ctx.strokeText(label,e.x-camera,e.y-age*38);ctx.fillText(label,e.x-camera,e.y-age*38);}
    });
    if(run.status==='fighting' && !run.paused && run.player.pose==='run' && run.player.jump===0 && now-footstep>320){window.RiftAudio.play('step',0);footstep=now;}
    window.RiftAudio.tick();
  }
  renderer.ready.then(()=>requestAnimationFrame(render)).catch(()=>{});
  window.RiftRenderer=renderer;
})();
