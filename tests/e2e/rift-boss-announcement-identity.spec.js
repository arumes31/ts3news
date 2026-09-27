const {test,expect}=require('@playwright/test');
test('boss windups announce once per attack and distinguish a new boss',async({page})=>{
 await page.goto('/abyss/rift?scenario=terminal-defeated');await expect(page.locator('#rift-start')).toBeEnabled();
 const result=await page.evaluate(async()=>{
  const {run}=await(await fetch('/api/abyss/rift')).json();run.status='fighting';run.paused=false;
  const boss={...run.player,id:'boss-first',name:'First guardian',kind:'boss',hp:100,max_hp:100,attacks:0,windup:2,attack_name:'Ground Slam'};run.enemies=[boss];
  RiftHUD.update({...run,enemies:[]},true);
  const messages=[];const observer=new MutationObserver(()=>messages.push(document.getElementById('rift-announcer').textContent));observer.observe(document.getElementById('rift-announcer'),{childList:true,subtree:true,characterData:true});
  for(let i=0;i<100;i++){boss.windup=2-i/100;RiftHUD.update(run,true);await Promise.resolve();}
  const first=messages.slice();boss.id='boss-second';boss.name='Second guardian';RiftHUD.update(run,true);await Promise.resolve();
  const second=messages.slice();observer.disconnect();return {first,second};
 });
 expect(result.first).toHaveLength(1);expect(result.first[0]).toContain('First guardian');
 expect(result.second).toHaveLength(2);expect(result.second[1]).toContain('Second guardian');
});
