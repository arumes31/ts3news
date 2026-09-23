const { test, expect } = require('@playwright/test');

test('displays area effects affecting the player in combat signals and battlefield', async ({ page }) => {
  // 1. Check default non-combat state
  await page.goto('/abyss/rift');
  await expect(page.locator('#rift-start')).toBeEnabled();

  const areaSignal = page.locator('#rift-area-effects');
  await expect(areaSignal).toBeVisible();
  await expect(areaSignal).toHaveAttribute('role', 'status');
  await expect(areaSignal).toHaveText('No active area effects');
  await expect(areaSignal).toHaveAttribute('data-effect-state', 'none');

  // 2. Open hazard practice drill where player spawns inside a fire hazard zone
  await page.goto('/abyss/rift?practice=hazard');
  await expect(page.locator('#rift-start')).toBeEnabled();

  await page.locator('#rift-start').click();
  await expect(page.locator('#rift-overlay')).toBeHidden();

  // The hazard drill has an active fire hazard covering the starting position (X: 110, Y: 365, W: 100, H: 90)
  // Within a couple of combat ticks, player enters either warning or active hazard state
  await expect.poll(async () => {
    return await areaSignal.getAttribute('data-effect-state');
  }, { timeout: 7000 }).toMatch(/^(active|warning|evading)$/);

  await expect(areaSignal).toHaveText(/Fire/i);

  // 3. Jump to evade area hazard
  await page.keyboard.press('Space');
  // While airborne (jump > 0.1), area effect reflects evading state if hazard is active or warning
  const evaluatedEvade = await page.evaluate(() => {
    const run = {
      status: 'fighting',
      clock: 1.5,
      player: { x: 150, y: 400, jump: 0.5 },
      practice: { arena: { hazards: [{ x: 100, y: 350, w: 100, h: 100, kind: 'poison', jumpable: true, period: 3.5, offset: 0, duration: 1 }] } },
    };
    return window.RiftHUD?.detectPlayerAreaEffects?.(run);
  });
  expect(evaluatedEvade).toEqual({
    kind: 'poison',
    state: 'evading',
    name: 'Poison hazard',
    label: 'Area effect: Poison (Evading via jump)',
  });

  // 4. Verify boss area attack detection
  const bossArea = await page.evaluate(() => {
    const run = {
      status: 'fighting',
      player: { x: 200, y: 350, jump: 0 },
      enemies: [{ hp: 100, windup: 1.2, kind: 'boss', attack_name: 'Ground Slam', target_x: 210, target_y: 350 }],
    };
    return window.RiftHUD?.detectPlayerAreaEffects?.(run);
  });
  expect(bossArea).toEqual({
    kind: 'boss_area',
    state: 'active',
    name: 'Boss Ground Slam',
    label: 'In area effect: Ground Slam blast zone',
  });

  // 5. Verify clean screenshot mode hides combat signals
  const screenshotBtn = page.locator('#rift-screenshot-toggle');
  await screenshotBtn.click();
  await expect(screenshotBtn).toHaveAttribute('aria-pressed', 'true');
  await expect(areaSignal).not.toBeVisible();

  // Restore HUD
  await screenshotBtn.click();
  await expect(screenshotBtn).toHaveAttribute('aria-pressed', 'false');
  await expect(areaSignal).toBeVisible();
});

test('airborne hazard status respects explicit jumpability',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 const states=await page.evaluate(()=>[true,false,undefined].map(jumpable=>RiftHUD.detectPlayerAreaEffects({status:'fighting',clock:1.4,player:{x:150,y:400,jump:.4},enemies:[],skill_timers:{},practice:{arena:{hazards:[{x:100,y:350,w:100,h:100,kind:'poison',period:7,offset:0,duration:1,jumpable}]}}})));
 expect(states.map(s=>s.state)).toEqual(['evading','active','active']);
 expect(states[1].label).toBe('In area effect: Poison hazard (Active)');expect(states[2].label).toBe(states[1].label);
});

test('cleared rooms suppress hazard warnings but retain a remaining slow',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 const states=await page.evaluate(()=>[.5,1.4].flatMap(clock=>[0,.8].map(slowed=>RiftHUD.detectPlayerAreaEffects({status:'cleared',clock,player:{x:150,y:400,jump:0},enemies:[{kind:'boss',hp:100,windup:1,target_x:150,target_y:400}],slow_source:'ice',skill_timers:{slowed},practice:{arena:{hazards:[{x:100,y:350,w:100,h:100,kind:'fire',period:7,offset:0,duration:1,jumpable:true}]}}}))));
 expect(states.map(s=>s?.state||null)).toEqual([null,'debuff',null,'debuff']);expect(states[1].label).toBe('Area effect: Slowed by Ice (0.8s)');
});
