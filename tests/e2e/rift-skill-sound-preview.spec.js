const {test,expect}=require('@playwright/test');
test('skill sound previews use combat cues, respect mute and leave the saved run unchanged',async({page})=>{
 await page.goto('/abyss/rift?scenario=spawn-hazards&subclass=elementalist');await expect(page.locator('#rift-start')).toBeEnabled();
 const before=await(await page.request.get('/api/abyss/rift')).json();
 await page.locator('#rift-loadout-preview').click();
 const entries=page.locator('#rift-glossary-entries article');expect(await entries.count()).toBeGreaterThan(2);
 await page.evaluate(()=>{window.previewCalls=[];const original=RiftAudio.preview;RiftAudio.preview=async(...args)=>{previewCalls.push(args);return original(...args);};RiftAudio.set('muted',false);});
 for(const entry of await entries.all()){
  const id=await entry.getAttribute('data-skill'),skill=[...before.build.skills,...before.build.signatures,...(before.build.ultimate?[before.build.ultimate]:[])].find(s=>s.id===id);
  await entry.locator('.rift-skill-sound').click();await expect(entry.locator('.rift-skill-sound-status')).toContainText('Previewing '+skill.name);
  expect(await page.evaluate(()=>previewCalls.at(-1))).toEqual(['effects',skill.kind]);
 }
 expect(await page.evaluate(()=>RiftAudio.context.state)).toBe('running');
 await page.evaluate(()=>RiftAudio.set('muted',true));await entries.first().locator('.rift-skill-sound').click();await expect(entries.first().locator('.rift-skill-sound-status')).toContainText('Sound is muted');
 await page.evaluate(()=>{RiftAudio.set('muted',false);RiftAudio.set('effects',0);});await entries.first().locator('.rift-skill-sound').click();await expect(entries.first().locator('.rift-skill-sound-status')).toContainText('Effects volume is zero');
 await page.evaluate(()=>RiftAudio.set('effects',.65));await entries.first().locator('.rift-skill-sound').click();await page.locator('#rift-skill-glossary > summary').click();await expect.poll(()=>page.evaluate(()=>RiftAudio.context.state)).toBe('suspended');
 expect((await(await page.request.get('/api/abyss/rift')).json()).run).toEqual(before.run);
 await page.locator('#rift-loadout-preview').click();await page.setViewportSize({width:390,height:1600});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
