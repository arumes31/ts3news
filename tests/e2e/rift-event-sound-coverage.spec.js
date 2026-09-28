const {test,expect}=require('@playwright/test');
const path=require('node:path');
test('arrival and class resource events synthesize sound and stop on pause',async({page})=>{
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.setContent('<button id="activate">Activate audio</button>');
 await page.addScriptTag({path:path.resolve(__dirname,'../../internal/bot/webassets/rift_audio.js')});
 await page.click('#activate');
 const result=await page.evaluate(async()=>{
  const audio=window.RiftAudio;await audio.setActive(true,0);
  const context=audio.context,create=context.createOscillator.bind(context),frequencies=[];
  context.createOscillator=function(){const oscillator=create(),start=oscillator.start.bind(oscillator);oscillator.start=function(...args){frequencies.push(oscillator.frequency.value);return start(...args);};return oscillator;};
  const counts={};
  for(const cue of ['arrival','barrier','resource','vanguard_guard','wave_gate_warning','wave_gate_close','wave_gate_open','wave_rest_enter','wave_rest_leave']){const before=frequencies.length;audio.play(cue,0);counts[cue]=frequencies.length-before;}
  await audio.setActive(false);
  const before=frequencies.length;audio.play('resource',0);
  return {counts,voices:audio.voices,pausedSounds:frequencies.length-before};
 });
 for(const count of Object.values(result.counts))expect(count).toBeGreaterThan(0);
 expect(result.voices).toBe(0);expect(result.pausedSounds).toBe(0);expect(errors).toEqual([]);
});
