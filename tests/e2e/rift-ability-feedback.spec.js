const {test,expect}=require('@playwright/test');

test('ability roles and cooldowns have visual and accessible feedback',async({page})=>{
  await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();
  const guard=page.locator('#rift-skills button').first();
  await page.keyboard.press('1');await expect(guard).toHaveAccessibleName(/Iron Guard · [\d.]+ seconds cooldown/);
  await expect(guard.locator('.rift-cooldown-ring')).toBeVisible();
  await expect(page.locator('[data-ability-role="builder"]')).toHaveAccessibleName(/Builder/);
  await expect(page.locator('[data-ability-role="finisher"]')).toHaveAccessibleName(/Finisher/);
  await expect(page.locator('[data-ability-role="ultimate"]')).toHaveAccessibleName(/Ultimate/);
  await page.keyboard.press('Escape');
  await page.evaluate(()=>{const button=document.querySelector('#rift-skills button');window.RiftAbilities.update(button,{id:'guard',name:'Iron Guard',cost:10,cooldown:8},{skill_timers:{guard:4},player:{mana:100}},'4.0 seconds cooldown');});
  await expect(guard.locator('.rift-cooldown-ring')).toHaveCSS('--cooldown-progress','50%');
  await expect(guard).toHaveAccessibleName('Iron Guard · 4.0 seconds cooldown');
  await page.evaluate(()=>{const button=document.querySelector('#rift-skills button');window.RiftAbilities.update(button,{id:'guard',name:'Iron Guard',cost:10,cooldown:8},{skill_timers:{},player:{mana:0}},'10 more mana needed');});
  await expect(guard.locator('.rift-cooldown-ring')).toBeHidden();await expect(guard).toHaveAccessibleName(/more mana needed/);
  await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
