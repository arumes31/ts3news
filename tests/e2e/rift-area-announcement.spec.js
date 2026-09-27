const {test,expect}=require('@playwright/test');
test('protection and slowing announce state changes while timers remain readable',async({page})=>{
 await page.goto('/abyss/rift?scenario=terminal-defeated');await expect(page.locator('#rift-start')).toBeEnabled();
 const result=await page.evaluate(async()=>{
  const {run}=await(await fetch('/api/abyss/rift')).json();run.status='fighting';run.paused=false;run.player.hp=run.player.max_hp;run.enemies=[];run.events=[];delete run.level;delete run.practice;run.skill_timers={};RiftHUD.update(run,true);
  const messages=[];const observers=[...document.querySelectorAll('#rift-area-effects,#rift-area-announcement')].filter(node=>node.getAttribute('aria-live')!=='off').map(node=>{const observer=new MutationObserver(()=>messages.push(node.textContent));observer.observe(node,{subtree:true,childList:true,characterData:true});return observer;});
  for(let i=0;i<12;i++){run.skill_timers.connection_grace=1.2-i/10;RiftHUD.update(run,true);await Promise.resolve();}
  const protection=document.getElementById('rift-area-effects').textContent;
  run.skill_timers.connection_grace=0;run.slow_source='ice';
  for(let i=0;i<30;i++){run.skill_timers.slowed=3-i/10;RiftHUD.update(run,true);await Promise.resolve();}
  const slowing=document.getElementById('rift-area-effects').textContent;
  run.skill_timers.slowed=0;RiftHUD.update(run,true);await Promise.resolve();observers.forEach(o=>o.disconnect());return {messages,protection,slowing};
 });
 expect(result).toEqual({messages:['Connection recovered: temporarily protected','Area effect: Slowed by Ice','No active area effects'],protection:'Connection recovered: protected (0.1s)',slowing:'Area effect: Slowed by Ice (0.1s)'});
 await expect(page.locator('#rift-area-effects')).toHaveAttribute('aria-live','off');await expect(page.locator('#rift-area-announcement')).toHaveAttribute('role','status');
});
