const {test,expect}=require('@playwright/test');
test('precision announces readiness changes without speaking every cooldown tick',async({page})=>{
 await page.goto('/abyss/rift?subclass=marksman&scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();
 const messages=await page.evaluate(async()=>{
  const {run}=await(await fetch('/api/abyss/rift')).json();run.paused=false;run.resource=2;run.marked=run.enemies.find(e=>e.hp>0).id;const finisher=run.build.signatures.find(s=>s.role==='finisher');
  const node=document.querySelector('#rift-precision-state');const messages=[];const observer=new MutationObserver(()=>messages.push(node.textContent));observer.observe(node,{childList:true,subtree:true,characterData:true});
  for(let i=0;i<30;i++){run.skill_timers[finisher.id]=3-i/10;RiftHUD.update(run,true);await Promise.resolve();}
  run.skill_timers[finisher.id]=0;RiftHUD.update(run,true);await Promise.resolve();for(let i=0;i<10;i++){run.player.mana=finisher.cost-10+i;RiftHUD.update(run,true);await Promise.resolve();}
  run.player.mana=finisher.cost;RiftHUD.update(run,true);await Promise.resolve();
  run.paused=true;RiftHUD.update(run,false);await Promise.resolve();observer.disconnect();return messages;
 });
 expect(messages).toHaveLength(5);expect(messages[0]).toContain('Cooling down');expect(messages[1]).toContain('Ready');expect(messages[2]).toContain('More mana needed');expect(messages[3]).toContain('Ready');expect(messages[4]).toContain('Paused');
});
