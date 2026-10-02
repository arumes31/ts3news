const {test,expect}=require('@playwright/test');
test('guard announces breaks and recovery without speaking stamina and timer ticks',async({page})=>{
 await page.goto('/abyss/rift?scenario=terminal-defeated');await expect(page.locator('#rift-start')).toBeEnabled();
 const result=await page.evaluate(async()=>{
  const {run}=await(await fetch('/api/abyss/rift')).json();run.status='fighting';run.paused=false;run.player.hp=run.player.max_hp;run.events=[];run.skill_timers.guard_break_recovery=0;RiftHUD.update(run,true);
  const node=document.getElementById('rift-guard-state');if(!node)return {missing:true};
  const messages=[];const observer=new MutationObserver(()=>messages.push(node.textContent));observer.observe(node,{childList:true,subtree:true,characterData:true});
  for(let i=0;i<30;i++){run.player.guard_stamina=100-i;RiftHUD.update(run,true);await Promise.resolve();}
  const stamina=document.getElementById('rift-guard-stamina').textContent;
  for(let i=0;i<30;i++){run.skill_timers.guard_break_recovery=3-i/10;RiftHUD.update(run,true);await Promise.resolve();}
  const broken=document.getElementById('rift-guard-stamina').textContent;
  run.skill_timers.guard_break_recovery=0;RiftHUD.update(run,true);await Promise.resolve();observer.disconnect();return {messages,stamina,broken};
 });
 expect(result).toEqual({messages:['Guard broken. Wait for recovery.','Guard available.'],stamina:'Stamina: 71%',broken:'Stamina: Broken (0.1s)'});
 await expect(page.locator('#rift-guard-stamina')).toHaveAttribute('aria-live','off');
 await expect(page.locator('#rift-guard-state')).toHaveAttribute('role','status');
});
