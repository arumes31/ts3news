const {test,expect}=require('@playwright/test');

test('mission briefing shows server encounter counts and scaling across difficulty bands',async({page})=>{
  await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
  const levels=(await(await page.request.get('/api/abyss/rift')).json()).levels;
  await page.locator('#rift-mission-preview > summary').click();
  for(const id of [1,26,52,78,100]){
    await page.locator('[data-level="'+id+'"]').click();const level=levels.find(level=>level.id===id);
    await expect(page.locator('#rift-mission-difficulty')).toContainText(level.difficulty);
    await expect(page.locator('#rift-mission-loot')).toContainText('Tier 1 — Epic · Tier 2 — Epic · Tier 3 — Legendary');
    await expect(page.locator('#rift-mission-loot')).toContainText('not guaranteed rarities');
    const total=level.rooms.reduce((sum,room)=>sum+room.encounter.enemies,0);
    await expect(page.locator('#rift-mission-difficulty')).toContainText(total+' across the mission');
    for(let i=0;i<3;i++){
      const preview=level.rooms[i].encounter,card=page.locator('#rift-room-previews article').nth(i);
      await expect(card).toContainText('Planned defenders: '+preview.enemies);
      await expect(card).toContainText('Health ×'+preview.health_multiplier.toFixed(3));
      await expect(card).toContainText('Damage ×'+preview.damage_multiplier.toFixed(3));
    }
  }
  await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
  await page.evaluate(async()=>{const data=await(await fetch('/api/abyss/rift')).json();const level=data.levels[0];level.rooms.forEach(room=>{delete room.encounter;delete room.loot_rarity_ceiling;});window.RiftMission.update(level,data.build,true);});
  await expect(page.locator('#rift-mission-difficulty')).toContainText('not saved with this expedition');
  await expect(page.locator('#rift-room-previews')).not.toContainText('Planned defenders');
  await expect(page.locator('#rift-mission-loot')).toContainText('not saved with this expedition');
});

test('mission links select the route and previews match all three saved terrain layouts',async({page})=>{
  await page.goto('/abyss/rift?mission=87');await expect(page.locator('#rift-start')).toHaveText('Enter mission 87');
  await expect(page.locator('[data-level="87"]')).toHaveAttribute('aria-current','true');
  const level=(await(await page.request.get('/api/abyss/rift')).json()).levels.find(level=>level.id===87);
  await page.locator('#rift-mission-preview > summary').click();
  const rooms=page.locator('#rift-room-previews article');await expect(rooms).toHaveCount(3);
  for(let i=0;i<3;i++){
    await expect(rooms.nth(i).locator('h4')).toContainText(level.rooms[i].name);
    await expect(rooms.nth(i).locator('[data-terrain="cover"]')).toHaveCount(level.rooms[i].obstacles.length);
    await expect(rooms.nth(i).locator('[data-terrain="hazard"]')).toHaveCount(level.rooms[i].hazards.length);
    expect(await rooms.nth(i).locator('[data-terrain="cover"]').first().getAttribute('x')).toBe(String(level.rooms[i].obstacles[0].x));
  }
  await expect(page.locator('#rift-mission-class')).toContainText('Vanguard');
  await expect(page.locator('#rift-mission-link')).toHaveValue(/\/abyss\/rift\?mission=87$/);
  await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});

test('invalid mission links fall back and active expeditions take priority over incoming links',async({page})=>{
  await page.goto('/abyss/rift?mission=999');await expect(page.locator('#rift-mission-link-status')).toContainText('invalid');await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('[data-level="12"]').click();await page.locator('#rift-start').click();await page.keyboard.press('Escape');
  await page.goto('/abyss/rift?mission=65');await expect(page.locator('[data-level="12"]')).toHaveAttribute('aria-current','true');
  await expect(page.locator('#rift-mission-link-status')).toContainText('saved expedition');
  expect((await(await page.request.get('/api/abyss/rift')).json()).run.level.id).toBe(12);
});

test('route preview preference survives reload and link copy uses only the mission',async({page})=>{
  await page.goto('/abyss/rift?mission=21&subclass=elementalist');await expect(page.locator('#rift-start')).toBeEnabled();
  await page.locator('#rift-preview-default').check();await expect(page.locator('#rift-mission-preview')).toHaveAttribute('open','');
  await page.reload();await expect(page.locator('#rift-mission-preview')).toHaveAttribute('open','');
  await page.evaluate(()=>Object.defineProperty(navigator,'clipboard',{configurable:true,value:{writeText:async value=>window.copiedMission=value}}));
  await page.locator('#rift-copy-mission').click();expect(await page.evaluate(()=>window.copiedMission)).toMatch(/\/abyss\/rift\?mission=21$/);
  await expect(page.locator('#rift-mission-link')).not.toHaveValue(/subclass/);
});

test('route briefing explains every required room objective and wave totals',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-mission-preview > summary').click();
 const expectations=[[2,'Capture three moving beacons'],[3,'Collect all three sigils'],[4,'15 uncontested seconds'],[5,'bank only after the final wave'],[6,'Totems grant no monster kills or loot'],[7,'30% slower while carrying'],[8,'permanently disable their linked floor hazards'],[9,'Surviving unmarked enemies retreat without granting kills or loot'],[10,'Stay near the spirit and clear nearby threats']];
 for(const [id,description] of expectations){
  await page.locator('[data-level="'+id+'"]').click();const card=page.locator('#rift-room-previews article').nth(1);await expect(card).toContainText(description);await expect(card).not.toContainText('Clear the patrol, then bank and continue.');
  if(id===5){await expect(card).toContainText('across three waves');await expect(page.locator('#rift-mission-difficulty')).toContainText('planned enemies per tier');await expect(card).not.toContainText('initial defenders');}
 }
 await page.locator('[data-level="1"]').click();await expect(page.locator('#rift-room-previews article').nth(0)).toContainText('Clear the patrol, then bank and continue.');await expect(page.locator('#rift-room-previews article').nth(2)).toContainText('Defeat the guardian and its defenders.');
});
