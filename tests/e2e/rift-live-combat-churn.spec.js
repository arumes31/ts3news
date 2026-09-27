const {test,expect}=require('@playwright/test');
test('continuous HUD ticks do not repeatedly mutate live regions',async({page})=>{
 await page.goto('/abyss/rift?scenario=terminal-defeated');await expect(page.locator('#rift-start')).toBeEnabled();
 const churn=await page.evaluate(async()=>{
  const {run}=await(await fetch('/api/abyss/rift')).json();run.status='fighting';run.paused=false;run.player.hp=run.player.max_hp;run.events=[];
  RiftHUD.update(run,true);
  const nodes=[...document.querySelectorAll('#rift-app [role=status],#rift-app [aria-live=polite],#rift-app [aria-live=assertive]')].filter(n=>n.getAttribute('aria-live')!=='off');
  const counts={};const observers=nodes.map((node,index)=>{const id=node.id||'anonymous-'+index;const observer=new MutationObserver(()=>counts[id]=(counts[id]||0)+1);observer.observe(node,{subtree:true,childList:true,characterData:true});return observer;});
  for(let i=0;i<100;i++){run.stats.seconds=i/10;run.player.x+=1;run.saved_at_ms=1700000000000+i*1000;RiftHUD.updateLatency(100+i);for(const id of [...run.build.skills,...run.build.signatures,run.build.ultimate].filter(Boolean).map(skill=>skill.id))run.skill_timers[id]=Math.max(0,10-i/10);RiftHUD.update(run,true);await Promise.resolve();}
  observers.forEach(o=>o.disconnect());return Object.entries(counts).filter(([,count])=>count>5);
 });
 expect(churn).toEqual([]);
});
