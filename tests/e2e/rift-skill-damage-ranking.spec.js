const {test,expect}=require('@playwright/test');
test('damage ranking uses confirmed attribution and includes ties',async({page})=>{
 await page.goto('/abyss/rift?scenario=terminal-defeated');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('.rift-run-statistics > summary').click();
 const render=async damage=>page.evaluate(async damage=>{const {run}=await(await fetch('/api/abyss/rift')).json();const skills=run.build.skills;run.stats.skill_damage=Object.fromEntries(damage.map((value,i)=>[skills[i].id,value]));RiftHUD.update(run,false);return skills.map(s=>s.name);},damage);
 const row=page.locator('#rift-statistics dt').filter({hasText:'Highest recorded skill damage'}).locator('xpath=following-sibling::dd[1]');
 const names=await render([10,50,20]);await expect(row).toContainText(names[1]);await expect(row).toContainText('50 damage');await expect(row).not.toContainText(names[0]);
 await render([50,50,0]);await expect(row).toContainText('Tie');await expect(row).toContainText(names[0]);await expect(row).toContainText(names[1]);
 await render([]);await expect(row).toHaveText('No confirmed skill casts.');
});
