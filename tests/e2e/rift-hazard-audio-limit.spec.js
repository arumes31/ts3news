const {test,expect}=require('@playwright/test');
test('hazard warning and shutdown cues have separate bounded audio budgets',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 const result=await page.evaluate(async()=>{const a=RiftAudio;await a.setActive(true,0);const burst=kind=>Array.from({length:12},()=>a.play(kind,0)).filter(Boolean).length;const shutdown=burst('hazard_deactivation'),warnings=burst('hazard_warning');const slash=a.play('slash',0);await a.setActive(false);await a.setActive(true,0);const afterReset=burst('hazard_warning');await new Promise(resolve=>setTimeout(resolve,350));const afterExpiry=burst('hazard_warning');await a.setActive(false);return {shutdown,warnings,slash,afterReset,afterExpiry,voices:a.voices};});
 expect(result).toEqual({shutdown:1,warnings:2,slash:true,afterReset:2,afterExpiry:2,voices:0});
});
