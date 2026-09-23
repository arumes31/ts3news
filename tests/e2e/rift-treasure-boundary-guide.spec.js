const {test,expect}=require('@playwright/test');
test('treasure bestiary explains escape instead of cornered attacks',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 const data=await(await page.request.get('/api/abyss/rift')).json();
 const unit=data.bestiary.find(u=>u.kind==='treasure');expect(unit).toBeTruthy();
 await page.locator('.rift-bestiary > summary').click();await page.getByRole('button',{name:'Inspect '+unit.name,exact:true}).click();
 await expect(page.locator('#rift-monster-tip')).toContainText('escapes at either arena edge');
 await expect(page.locator('#rift-monster-stats')).toContainText('None: flees');
 await expect(page.getByRole('button',{name:'Preview attack',exact:true})).toHaveCount(0);
 await expect(page.getByRole('button',{name:'Preview escape',exact:true})).toBeVisible();
});
