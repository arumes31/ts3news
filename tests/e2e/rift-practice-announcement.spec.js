const {test,expect}=require('@playwright/test');
test('movement practice announces milestones and completion while retaining precise progress',async({page})=>{
 await page.goto('/abyss/rift?practice=movement');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.evaluate(()=>{
  window.practiceSpoken=[];window.practiceVisual=[];window.practiceObservers=[];
  for(const node of document.querySelectorAll('#rift-practice-progress,#rift-practice-announcement')){
   const observer=new MutationObserver(()=>{if(node.id==='rift-practice-progress')window.practiceVisual.push(node.textContent);if(node.getAttribute('aria-live')!=='off')window.practiceSpoken.push(node.textContent);});observer.observe(node,{childList:true,subtree:true,characterData:true});window.practiceObservers.push(observer);
  }
 });
 await page.keyboard.down('KeyD');try{await expect(page.locator('#rift-practice-progress')).toHaveText('Drill complete',{timeout:15000});}finally{await page.keyboard.up('KeyD');}
 const result=await page.evaluate(()=>{window.practiceObservers.forEach(o=>o.disconnect());return {spoken:window.practiceSpoken,visual:window.practiceVisual};});
 expect(result.visual.length).toBeGreaterThan(10);expect(result.spoken.length).toBeLessThanOrEqual(5);expect(result.spoken).toContain('Drill complete');expect(result.spoken.some(text=>text==='50% to finish')).toBe(true);
 await expect(page.locator('#rift-practice-progress')).toHaveAttribute('aria-live','off');
});
