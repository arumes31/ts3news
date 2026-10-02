const {test,expect}=require('@playwright/test');

test('ultimate ownership distinguishes selected, inactive, empty and unknown builds',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-skill-glossary > summary').click();
 await expect(page.locator('#rift-ultimate-ownership')).toHaveText('Owned ultimates: Rift Nova');await expect(page.locator('#rift-ultimate-selection')).toContainText('Selected for Brawl: Rift Nova');
 await expect(page.locator('#rift-ultimate-state')).toContainText('Rift Nova · Paused');
 const show=async mode=>page.evaluate(async mode=>{const data=await(await fetch('/api/abyss/rift')).json();delete data.build.ultimate;if(mode==='empty')data.build.owned_ultimates=[];if(mode==='unknown')delete data.build.owned_ultimates;window.RiftLoadouts.init(data.build,()=>true);},mode);
 await show('inactive');await expect(page.locator('#rift-ultimate-ownership')).toContainText('Rift Nova');await expect(page.locator('#rift-ultimate-selection')).toContainText('No active ultimate selected');
 await show('empty');await expect(page.locator('#rift-ultimate-ownership')).toHaveText('No ultimates owned.');
 await show('unknown');await expect(page.locator('#rift-ultimate-ownership')).toContainText('ownership is unavailable');
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
