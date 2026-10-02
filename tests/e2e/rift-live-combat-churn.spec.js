const {test,expect}=require('@playwright/test');
for(const subclass of ['vanguard','berserker','marksman','beastmaster','elementalist','chronomancer','oracle','geomancer','bloodblade','voidwalker','runesmith','alchemist'])test(subclass+' continuous HUD ticks do not repeatedly mutate live regions',async({page})=>{
 await page.goto('/abyss/rift?scenario=terminal-defeated&subclass='+subclass);await expect(page.locator('#rift-start')).toBeEnabled();
 const result=await page.evaluate(async()=>{
  const {run}=await(await fetch('/api/abyss/rift')).json();run.status='fighting';run.paused=false;run.player.hp=run.player.max_hp;run.events=[];run.resource=2;run.build.relic=true;if(run.enemies[0]){run.enemies[0].hp=run.enemies[0].max_hp;run.marked=run.enemies[0].id;}
  RiftHUD.update(run,true);
  const nodes=[...document.querySelectorAll('#rift-app [role=status],#rift-app [aria-live=polite],#rift-app [aria-live=assertive]')].filter(n=>n.getAttribute('aria-live')!=='off');
  const counts={};const observers=nodes.map((node,index)=>{const id=node.id||'anonymous-'+index;const observer=new MutationObserver(()=>counts[id]=(counts[id]||0)+1);observer.observe(node,{subtree:true,childList:true,characterData:true});return observer;});
  for(let i=0;i<100;i++){run.stats.seconds=i/10;run.player.x+=1;run.player.guard_stamina=100-i;run.skill_timers.guard_break_recovery=Math.max(0,3-i/10);run.skill_timers.connection_grace=Math.max(0,1.2-i/10);run.skill_timers.slowed=Math.max(0,4-i/10);run.saved_at_ms=1700000000000+i*1000;RiftHUD.updateLatency(100+i);for(const id of [...run.build.skills,...run.build.signatures,run.build.ultimate].filter(Boolean).map(skill=>skill.id))run.skill_timers[id]=Math.max(0,10-i/10);RiftHUD.update(run,true);await Promise.resolve();}
  observers.forEach(o=>o.disconnect());return {subclass:run.build.class,churn:Object.entries(counts).filter(([,count])=>count>5)};
 });
 expect(result.subclass).toBe(subclass);expect(result.churn).toEqual([]);
});
