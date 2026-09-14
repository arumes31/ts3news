const {test,expect}=require('@playwright/test');

test('bestiary sound previews use combat cues without starting a run',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 const roster=(await(await page.request.get('/api/abyss/rift')).json()).bestiary,boss=roster.find(unit=>unit.kind==='boss');
 await page.locator('#rift-monsters').locator('..').locator('summary').first().click();
 await page.getByRole('button',{name:'Inspect '+boss.name,exact:true}).click();
 const stats=page.locator('#rift-monster-stats');await expect(stats).toContainText('Abyss tier');await expect(stats).toContainText(boss.tier);
 const before=await page.evaluate(()=>window.RiftAudio.played);await page.getByRole('button',{name:'Preview roar',exact:true}).click();await expect(page.locator('#rift-monster-sound-status')).toContainText('Previewing roar');
 expect(await page.evaluate(()=>window.RiftAudio.played)).toBe(before+1);await expect.poll(()=>page.evaluate(()=>window.RiftAudio.context.state)).toBe('suspended');
 expect((await(await page.request.get('/api/abyss/rift')).json()).run).toBeNull();
 await page.evaluate(()=>window.RiftAudio.set('muted',true));await page.getByRole('button',{name:'Preview defeat',exact:true}).click();await expect(page.locator('#rift-monster-sound-status')).toContainText('muted');expect(await page.evaluate(()=>window.RiftAudio.played)).toBe(before+1);
});

test('closing monster details cancels a pending sound unlock',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();const roster=(await(await page.request.get('/api/abyss/rift')).json()).bestiary;
 await page.locator('.rift-bestiary > summary').click();await page.getByRole('button',{name:'Inspect '+roster.find(unit=>unit.kind==='boss').name,exact:true}).click();
 await page.evaluate(()=>{const audio=window.RiftAudio,unlock=audio.unlock;const gate=new Promise(resolve=>window.releaseMonsterAudio=resolve);audio.unlock=async()=>{await gate;return unlock();};});
 await page.getByRole('button',{name:'Preview roar',exact:true}).click();await page.locator('#rift-monster-close').click();await page.evaluate(()=>window.releaseMonsterAudio());
 await expect.poll(()=>page.evaluate(()=>window.RiftAudio.context?.state)).toBe('suspended');expect(await page.evaluate(()=>window.RiftAudio.played)).toBe(0);await expect(page.locator('#rift-monster-detail')).toBeHidden();
});

test('ranged creature previews play its adapted projectile cue',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();const roster=(await(await page.request.get('/api/abyss/rift')).json()).bestiary,unit=roster.find(unit=>unit.kind==='archer');
 await page.locator('.rift-bestiary > summary').click();await page.getByRole('button',{name:'Inspect '+unit.name,exact:true}).click();
 await page.evaluate(()=>{window.previewCues=[];const preview=window.RiftAudio.previewCue;window.RiftAudio.previewCue=kind=>{window.previewCues.push(kind);return preview(kind);};});
 await page.getByRole('button',{name:'Preview projectile',exact:true}).click();await expect(page.locator('#rift-monster-sound-status')).toContainText('Previewing projectile');expect(await page.evaluate(()=>window.previewCues)).toEqual([unit.shot||'arrow']);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
