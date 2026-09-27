const {test,expect}=require('@playwright/test');
const path=require('node:path');
test('combat event cues create sources, respect mute, and retire on pause',async({page})=>{
 await page.setContent('<button>Activate</button>');
 await page.addScriptTag({path:path.resolve(__dirname,'../../internal/bot/webassets/rift_audio.js')});
 await page.locator('button').click();
 const result=await page.evaluate(async()=>{
  const a=RiftAudio;
  const kinds=['miss_reaction','charge_warning','charge_rush','charge_recovery','ground_strike','airborne_launch','juggle_hit','backstab','guard_break','dodge_action','first_hit_grace','enemy_aware','alert_propagate','flank_attempt','pack_attack','summon_spawn','summon_punish'];
  const results=[];
  for(const kind of kinds){
   await a.setActive(true,0);
   const before=a.voices;
   const accepted=a.play(kind,.5);
   const added=a.voices-before;
   a.muted=true;const mutedBefore=a.voices;const muted=a.play(kind,0);const mutedAdded=a.voices-mutedBefore;a.muted=false;
   await a.setActive(false);
   results.push({kind,accepted,added,muted,mutedAdded,remaining:a.voices});
  }
  return results;
 });
 for(const cue of result){
  expect(cue.accepted,cue.kind).toBe(true);expect(cue.added,cue.kind).toBeGreaterThan(0);
  expect(cue.muted,cue.kind).toBe(false);expect(cue.mutedAdded,cue.kind).toBe(0);expect(cue.remaining,cue.kind).toBe(0);
 }
});
