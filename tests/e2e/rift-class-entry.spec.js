const {test,expect}=require('@playwright/test');
const fs=require('fs');
test('subclass entries trigger once per live room, not reload or ordinary snapshots',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const result=await page.evaluate(async()=>{
  const base=(await(await fetch('/api/abyss/rift')).json()).run,drawn=[],sounds=[];window.RiftRenderer.reduced=true;window.RiftAudio.play=(kind)=>sounds.push(kind);
  const frame=()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve)));
  for(const name of ['vanguard','berserker','marksman','beastmaster','elementalist','chronomancer','oracle','geomancer','bloodblade','voidwalker','runesmith','alchemist']){
   const run=structuredClone(base);run.id='entry-'+name;run.status='fighting';run.paused=false;run.build.class=name;run.enemies=[];run.events=[];run.counter=0;
   const before=window.RiftRenderer.classEntryCount||0;window.RiftRenderer.snapshot(run,false);await frame();const first={...window.RiftRenderer.lastClassEntry};await new Promise(resolve=>setTimeout(resolve,40));const last={...window.RiftRenderer.lastClassEntry};window.RiftRenderer.snapshot(structuredClone(run),false);await frame();drawn.push({first,last,added:window.RiftRenderer.classEntryCount-before});
  }
  const last=structuredClone(base);last.id='entry-alchemist';last.status='fighting';last.build.class='alchemist';last.events=[];last.enemies=[];last.counter=0;last.room=1;
  const before=window.RiftRenderer.classEntryCount;window.RiftRenderer.snapshot(last,false);const changed=window.RiftRenderer.classEntryCount-before;last.room=2;window.RiftRenderer.snapshot(structuredClone(last),true);const replay=window.RiftRenderer.classEntryCount-before;return {drawn,sounds,changed,replay};
 });
 expect(result.drawn).toHaveLength(12);for(const row of result.drawn){expect(row.added).toBe(1);expect(row.first.reduced).toBe(true);expect(row.last.radius).toBe(row.first.radius);expect(row.last.turn).toBe(0);}
 expect(new Set(result.drawn.map(({first})=>[first.row,first.points,first.color].join(','))).size).toBe(12);expect(result.changed).toBe(1);expect(result.replay).toBe(1);expect(result.sounds).toEqual(['shield','slash','arrow','pack','fire','ice','heal','quake','slash','void','rune','poison','poison']);
 const png=await page.evaluate(async()=>{const run=(await(await fetch('/api/abyss/rift')).json()).run;run.id='entry-image';run.status='fighting';run.paused=true;run.events=[];run.enemies=[];run.build.class='runesmith';window.RiftRenderer.snapshot(run,false);await new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve)));return document.getElementById('rift-canvas').toDataURL('image/png');});fs.writeFileSync('test-results/class-entry-reduced.png',Buffer.from(png.split(',')[1],'base64'));
});

test('Oracle entry artwork obeys zero effect intensity',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const alphas=await page.evaluate(async()=>{
  const run=(await(await fetch('/api/abyss/rift')).json()).run,ctx=document.getElementById('rift-canvas').getContext('2d'),original=ctx.drawImage,alphas=[];
  ctx.drawImage=function(...args){if(args.length===9&&args[0].src===new URL(document.getElementById('rift-app').dataset.effects,location.href).href&&args[2]===args[0].height*5/6)alphas.push(this.globalAlpha);return original.apply(this,args);};
  window.RiftDisplay.effectIntensity=0;run.id='oracle-intensity';run.status='fighting';run.paused=true;run.build.class='oracle';run.enemies=[];run.drops=[];run.events=[];window.RiftRenderer.snapshot(run,false);await new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve)));ctx.drawImage=original;return alphas;
 });expect(alphas.length).toBeGreaterThan(0);expect(alphas.every(alpha=>alpha===0)).toBe(true);
});

test('normal entry expands without changing the saved player',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
 const result=await page.evaluate(async()=>{
  const run=(await(await fetch('/api/abyss/rift')).json()).run;run.id='moving-entry';run.status='fighting';run.paused=false;run.events=[];run.enemies=[];run.build.class='elementalist';const before=JSON.stringify(run.player);window.RiftRenderer.reduced=false;window.RiftRenderer.snapshot(run,false);await new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve)));const first={...window.RiftRenderer.lastClassEntry};await new Promise(resolve=>setTimeout(resolve,100));return {first,last:window.RiftRenderer.lastClassEntry,unchanged:JSON.stringify(run.player)===before};
 });expect(result.first.reduced).toBe(false);expect(result.last.radius).toBeGreaterThan(result.first.radius);expect(result.last.turn).toBeGreaterThan(result.first.turn);expect(result.unchanged).toBe(true);
});
