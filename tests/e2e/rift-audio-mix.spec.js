const {test,expect}=require('@playwright/test');

test('audio mix migrates parent levels, saves independent channels and resets without unmuting',async({page})=>{
  await page.addInitScript(()=>{if(!sessionStorage.getItem('mixSeed')){localStorage.setItem('riftAudio:effects','0.4');localStorage.setItem('riftAudio:ambience','0.2');localStorage.setItem('riftAudio:voice','"bad"');sessionStorage.setItem('mixSeed','1');}});
  await page.goto('/abyss/rift');await page.locator('.rift-settings > summary').click();
  await expect(page.locator('#rift-music-volume')).toHaveValue('20');await expect(page.locator('#rift-voice-volume')).toHaveValue('40');
  await page.locator('#rift-music-volume').fill('13');await page.locator('#rift-interface-volume').fill('77');await page.locator('#rift-mono-audio').check();await page.locator('#rift-effects-volume').fill('10');
  await page.reload();await page.locator('.rift-settings > summary').click();await expect(page.locator('#rift-voice-volume')).toHaveValue('40');await expect(page.locator('#rift-music-volume')).toHaveValue('13');await expect(page.locator('#rift-interface-volume')).toHaveValue('77');await expect(page.locator('#rift-mono-audio')).toBeChecked();
  await page.locator('#rift-sound').click();await page.locator('#rift-reset-audio').click();
  await expect(page.locator('#rift-sound')).toHaveText('Sound off');await expect(page.locator('#rift-music-volume')).toHaveValue('35');await expect(page.locator('#rift-mono-audio')).not.toBeChecked();
});

test('audio previews leave a paused expedition paused and return its context to silence',async({page})=>{
  await page.goto('/abyss/rift');await page.locator('#rift-start').click();await page.keyboard.press('Escape');
  await page.locator('.rift-settings > summary').click();await page.locator('[data-audio-preview="voice"]').click();
  await expect(page.locator('#rift-audio-preview-status')).toContainText('Previewing');
  expect((await(await page.request.get('/api/abyss/rift')).json()).run.paused).toBe(true);
  await expect.poll(()=>page.evaluate(()=>window.RiftAudio.context.state)).toBe('suspended');
  await page.locator('#rift-start').click();await expect.poll(()=>page.evaluate(()=>window.RiftAudio.context.state)).toBe('running');await page.keyboard.press('Escape');
});

test('cancelling a pending audio preview does not leave a late-unlocked context running',async({page})=>{
  await page.goto('/abyss/rift');
  const state=await page.evaluate(async()=>{
    const audio=window.RiftAudio,unlock=audio.unlock;let release;const gate=new Promise(resolve=>release=resolve);
    audio.unlock=async()=>{await gate;return unlock();};
    const pending=audio.preview('music');await audio.setActive(false);release();await pending;return audio.context.state;
  });expect(state).toBe('suspended');
});

test('music, creature and interface signals use separate real audio buses and mono centers them',async({page})=>{
  await page.addInitScript(()=>{window.audioEdges=[];const original=AudioNode.prototype.connect;AudioNode.prototype.connect=function(target,...args){window.audioEdges.push([this,target]);return original.call(this,target,...args);};});
  await page.goto('/abyss/rift');
  const result=await page.evaluate(async()=>{
    const audio=window.RiftAudio;for(const [key,value] of Object.entries({effects:.11,ambience:.22,music:.33,voice:.44,interface:.55,mono:true}))audio.set(key,value);
    await audio.setActive(true,0);
    const music=window.audioEdges.filter(([source,target])=>source instanceof GainNode&&source.gain.value===Math.fround(.017)&&target instanceof GainNode).map(([,target])=>target.gain.value);
    const channels={};for(const kind of ['slash','boss_roar','ui']){window.audioEdges=[];audio.play(kind,.8);channels[kind]=window.audioEdges.filter(([source])=>source instanceof StereoPannerNode).map(([source,target])=>({pan:source.pan.value,gain:target.gain.value}));}
    await audio.setActive(false);return {music,channels,voicesAfterPause:audio.voices};
  });
  expect(result.music.length).toBe(3);for(const gain of result.music)expect(gain).toBeCloseTo(.33);
  expect(result.voicesAfterPause).toBe(0);
  for(const [kind,gain] of [['slash',.11],['boss_roar',.44],['ui',.55]]){expect(result.channels[kind].length).toBeGreaterThan(0);for(const node of result.channels[kind]){expect(node.pan).toBe(0);expect(node.gain).toBeCloseTo(gain);}}
});
