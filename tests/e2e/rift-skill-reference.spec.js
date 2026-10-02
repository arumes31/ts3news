const {test,expect}=require('@playwright/test');

test('skill reference describes server targeting and recovery before starting',async({page})=>{
  await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-skill-glossary > summary').click();
  const entry=id=>page.locator('#rift-glossary-entries [data-skill="'+id+'"]');
  await expect(entry('guard')).toContainText('Self-targeted');await expect(entry('guard')).toContainText('protective barrier');
  const descriptions=await page.evaluate(async()=>{const data=await(await fetch('/api/abyss/rift')).json();return data.build.skills.map(s=>({id:s.id,ref:s.reference}));});
  for(const s of descriptions){if(s.ref.target==='area'){await expect(entry(s.id)).toContainText(s.ref.depth+' lane-depth units');await expect(entry(s.id)).toContainText('either direction');}if(s.ref.target==='projectile'){await expect(entry(s.id)).toContainText('facing direction');await expect(entry(s.id)).toContainText(s.ref.depth+' lane-depth units');}}
  const text=await page.evaluate(()=>window.RiftAbilities.describe({kind:'heal',role:'finisher',reference:{target:'self',healing:.15,barrier:false}}, {class:'bloodblade'}));
  expect(text).toContain('Direct healing: 15%');expect(text).toContain('4% of maximum HP per charge');
  const pierce=await page.evaluate(()=>window.RiftAbilities.describe({kind:'slash',pierce:.3,reference:{target:'area',horizontal:155,depth:60,healing:0,barrier:false}},{}));expect(pierce).toContain('Base armor piercing: 30%');
  const bonus=await page.evaluate(()=>window.RiftAbilities.describe({kind:'slash',role:'finisher',reference:{target:'area',horizontal:155,depth:60,healing:0,barrier:false}},{class:'marksman'}));expect(bonus).toContain('60% additional armor piercing against the marked target');
  await page.locator('#rift-glossary-search').fill('projectile');await expect(page.locator('#rift-glossary-entries article:visible')).not.toHaveCount(0);
  await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
