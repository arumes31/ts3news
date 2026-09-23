const {test,expect}=require('@playwright/test');

test('health cost preview respects charges, surviving health and class changes',async({page})=>{
  await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();
  const show=async(hp,charges,cls='voidwalker')=>page.evaluate(async({hp,charges,cls})=>{const run=(await(await fetch('/api/abyss/rift')).json()).run;run.build.class=cls;run.player.max_hp=400;run.player.hp=hp;run.resource=charges;window.RiftHUD.update(run,false,true);window.RiftLoadouts.init(run.build,()=>true);},{hp,charges,cls});
  await show(200,3);await expect(page.locator('#rift-health-cost')).toHaveText('Finisher health cost: 20.0 HP · leaves at least 1 HP');
  await expect(page.locator('[data-ability-role="finisher"]')).toHaveAccessibleName(/Spends 20.0 HP/);
  await expect(page.locator('[data-ability-role="finisher"] .rift-health-cost-label')).toHaveText('−20.0 HP');await expect(page.locator('#rift-health-cost-warning')).toContainText('200.0 → 180.0 HP');await expect(page.locator('#rift-health-cost-warning')).toHaveAttribute('data-low','false');
  await page.locator('#rift-skill-glossary > summary').click();await expect(page.locator('#rift-class-cost-reference')).toContainText('5% of maximum health');
  await show(7,1);await expect(page.locator('#rift-health-cost')).toContainText('6.0 HP');
  await expect(page.locator('#rift-health-cost-warning')).toContainText('Low-health cast: 7.0 → 1.0 HP');await expect(page.locator('#rift-health-cost-warning')).toHaveAttribute('data-low','true');
  await page.setViewportSize({width:390,height:844});await page.locator('#rift-health-cost-warning').scrollIntoViewIfNeeded();await page.locator('#rift-health-cost-warning').screenshot({path:'test-results/voidwalker-cost-warning.png'});
  await show(1,3);await expect(page.locator('#rift-health-cost')).toContainText('0.0 HP');
  await expect(page.locator('#rift-health-cost-warning')).toContainText('1.0 → 1.0 HP');await expect(page.locator('[data-ability-role="finisher"] .rift-health-cost-label')).toBeHidden();
  await show(200,0);await expect(page.locator('#rift-health-cost')).toHaveText('Finisher health cost: 0.0 HP · no charges to spend');
  await expect(page.locator('[data-ability-role="finisher"]')).not.toHaveAccessibleName(/Spends/);await expect(page.locator('#rift-health-cost-warning')).toBeHidden();
  await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
  await show(200,3,'vanguard');await expect(page.locator('#rift-health-cost')).toBeHidden();await expect(page.locator('#rift-class-cost-reference')).toBeHidden();
});
