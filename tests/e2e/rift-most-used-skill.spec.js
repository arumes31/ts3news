const {test,expect}=require('@playwright/test');
test('most-used skill uses confirmed counts, includes ties and explains missing records',async({page})=>{
 await page.goto('/abyss/rift?scenario=terminal-defeated');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.locator('.rift-run-statistics > summary').click();
 const render=async uses=>page.evaluate(async uses=>{const {run}=await(await fetch('/api/abyss/rift')).json();const skills=[...run.build.skills,...run.build.signatures,run.build.ultimate].filter(Boolean);run.stats.skill_uses=Object.fromEntries(uses.map((count,i)=>[skills[i].id,count]));run.stats.skills_cast=uses.reduce((a,b)=>a+b,0);RiftHUD.update(run,false);return skills.map(s=>s.name);},uses);
 const row=page.locator('#rift-statistics dt').filter({hasText:'Most used recorded skill'}).locator('xpath=following-sibling::dd[1]');
 const names=await render([2,7,1]);await expect(row).toContainText(names[1]);await expect(row).toContainText('7 casts');await expect(row).not.toContainText(names[2]);
 await render([7,7,1]);await expect(row).toContainText(names[0]);await expect(row).toContainText(names[1]);await expect(row).toContainText('Tie');
 await render([0,0,0]);await expect(row).toHaveText('No confirmed skill casts.');
 await page.evaluate(async()=>{const {run}=await(await fetch('/api/abyss/rift')).json();run.stats.skill_uses={};run.stats.skills_cast=10;RiftHUD.update(run,false);});
 await expect(row).toHaveText('Per-skill records unavailable.');
});
