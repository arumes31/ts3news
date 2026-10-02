const {test,expect}=require('@playwright/test');

async function savedRun(page){await page.goto('/abyss/rift');await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');return (await(await page.request.get('/api/abyss/rift')).json()).run;}

test('readiness cues wait for casting recovery and do not repeat on ordinary attacks or replay',async({page})=>{
  const run=await savedRun(page);
  const cues=await page.evaluate(run=>{
    const heard=[];window.RiftAudio.play=kind=>heard.push(kind);run.paused=false;run.status='fighting';run.clock=0;run.events=[];run.resource=0;run.player.cooldown=0;run.player.mana=100;run.skill_timers[run.build.ultimate.id]=2;
    const feedback=window.RiftFeedback;feedback.update(run,true,true);
    run.clock=1;run.skill_timers[run.build.ultimate.id]=0;run.resource=1;run.player.cooldown=.3;feedback.update(run,false,true);const waiting=heard.slice();
    run.clock=2;run.player.cooldown=0;feedback.update(run,false,true);const ready=heard.slice();
    run.player.cooldown=.3;feedback.update(run,false,true);run.player.cooldown=0;feedback.update(run,false,true);feedback.update(run,true,true);
    return {waiting,ready,final:heard};
  },run);
  expect(cues.waiting).toEqual([]);expect(cues.ready).toEqual(['ultimate_ready','finisher_ready']);expect(cues.final).toEqual(cues.ready);
});

test('low-health warnings have a combat-time cooldown and captions work while muted',async({page})=>{
  const run=await savedRun(page);await page.locator('.rift-settings > summary').click();await page.locator('#rift-combat-captions').check();
  const clocks=await page.evaluate(run=>{
    const heard=[];window.RiftAudio.set('muted',true);window.RiftAudio.play=kind=>heard.push({kind,clock:run.clock});run.paused=false;run.status='fighting';run.clock=0;run.events=[];run.player.hp=run.player.max_hp;
    window.RiftFeedback.update(run,true,true);
    for(const [clock,fraction] of [[1,.2],[2,.15],[13,.1]]){if(clock===2){run.paused=true;window.RiftFeedback.update(run,false,false);run.paused=false;window.RiftFeedback.update(run,true,true);}run.clock=clock;run.player.hp=run.player.max_hp*fraction;window.RiftFeedback.update(run,false,true);}
    return heard.filter(cue=>cue.kind==='low_health').map(cue=>cue.clock);
  },run);
  expect(clocks).toEqual([1,13]);await expect(page.locator('#rift-captions')).toContainText('Low health');
  await page.reload();await page.locator('.rift-settings > summary').click();await expect(page.locator('#rift-combat-captions')).toBeChecked();
});

test('each new warning cue produces a distinct synthesized tone sequence',async({page})=>{
  await page.goto('/abyss/rift');
  const signatures=await page.evaluate(async()=>{
    const audio=window.RiftAudio;await audio.setActive(true,0);const context=audio.context,create=context.createOscillator.bind(context),notes=[];
    context.createOscillator=()=>{const oscillator=create(),set=oscillator.frequency.setValueAtTime.bind(oscillator.frequency);oscillator.frequency.setValueAtTime=(frequency,time)=>{notes.push([frequency,oscillator.type]);return set(frequency,time);};return oscillator;};
    const signatures=[];for(const kind of ['low_health','ultimate_ready','finisher_ready']){notes.length=0;audio.play(kind,0);signatures.push(JSON.stringify(notes));}await audio.setActive(false);return signatures;
  });
  expect(signatures.every(value=>value!=='[]')).toBe(true);expect(new Set(signatures).size).toBe(3);
});

test('a burst of confirmed sound cues keeps captions and live announcements bounded',async({page})=>{
  const run=await savedRun(page);await page.locator('.rift-settings > summary').click();await page.locator('#rift-combat-captions').check();
  await page.evaluate(run=>{
    run.paused=false;window.RiftFeedback.update(run,true,true);
    run.events=['boss_roar','slam','pickup','clear','boss_roar','slam','pickup','clear'].map(kind=>({id:++run.counter,kind,x:400,y:400,value:0}));window.RiftFeedback.update(run,false,true);
  },run);
  await expect(page.locator('#rift-captions li')).toHaveCount(3);
  const spoken=await page.locator('#rift-captions [role="status"]').textContent();expect(spoken.split('. ').filter(Boolean).length).toBeLessThanOrEqual(3);
});

test('confirmed boss cues are captioned once and recovery clears old captions',async({page})=>{
  await page.addInitScript(()=>{window.AudioContext=undefined;window.webkitAudioContext=undefined;});
  await page.goto('/abyss/rift');await page.locator('.rift-settings > summary').click();await page.locator('#rift-combat-captions').check();
  let injected=false;
  await page.route('**/api/abyss/rift',async route=>{
    const response=await route.fetch(),data=await response.json();
    if(route.request().postDataJSON()?.kind==='step'&&!injected){injected=true;data.run.counter++;data.run.events=[{id:data.run.counter,kind:'boss_roar',x:900,y:400,value:0}];}
    await route.fulfill({response,json:data});
  });
  await page.locator('#rift-start').click();await expect(page.locator('#rift-captions li')).toHaveText(['Boss roar']);
  await page.keyboard.press('Escape');await expect(page.locator('#rift-captions')).toBeHidden();
  await page.locator('#rift-start').click();await expect(page.locator('#rift-captions')).toBeHidden();await page.keyboard.press('Escape');
});
