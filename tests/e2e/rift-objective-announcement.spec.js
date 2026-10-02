const {test,expect}=require('@playwright/test');
for(const scenario of ['ritual','collapse'])test(scenario+' keeps countdowns visible without repeated live announcements',async({page})=>{
 await page.goto('/abyss/rift?scenario='+scenario);await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();
 await page.evaluate(()=>{
  window.objectiveChanges={};window.objectiveObservers=[];
  for(const node of document.querySelectorAll('#rift-room-objective [role=status]')){
   if(node.getAttribute('aria-live')==='off')continue;
   const observer=new MutationObserver(()=>{(window.objectiveChanges[node.id]??=[]).push(node.textContent);});observer.observe(node,{subtree:true,childList:true,characterData:true});window.objectiveObservers.push(observer);
  }
 });
 await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 const progress=page.locator('#rift-room-objective-progress');const before=await progress.textContent();await expect.poll(()=>progress.textContent()).not.toBe(before);
 await page.waitForTimeout(2500);
 await expect(page.locator('#rift-room-objective-announcement')).toContainText(scenario==='ritual'?'Pulse imminent':'Caught in the collapse',{timeout:10000});
 await page.keyboard.press('Escape');await expect(progress).toContainText('Paused');
 const changes=await page.evaluate(()=>{window.objectiveObservers.forEach(o=>o.disconnect());return window.objectiveChanges;});
 for(const messages of Object.values(changes))expect(messages.length).toBeLessThanOrEqual(5);
 await expect(page.locator('#rift-room-objective-announcement')).toContainText('Paused');
 await expect(progress).toHaveAttribute('aria-live','off');
});
