const {test,expect}=require('@playwright/test');
test('caption repetition suppression survives more cue kinds than the visible list holds',async({page})=>{
 await page.goto('/abyss/rift?scenario=terminal-defeated');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('.rift-settings > summary').click();await page.locator('#rift-combat-captions').check();
 const messages=await page.evaluate(async()=>{
  const {run}=await(await fetch('/api/abyss/rift')).json();run.status='fighting';run.paused=false;run.player.hp=run.player.max_hp;run.events=[];run.counter=0;run.enemies=[];RiftFeedback.update(run,true,true);
  const messages=[],node=document.querySelector('#rift-captions [role=status]');const observer=new MutationObserver(()=>messages.push(node.textContent));observer.observe(node,{subtree:true,childList:true,characterData:true});
  const kinds=['block','dodge','cooldown_rejection','empty_mana'];
  for(let i=1;i<=100;i++){run.counter=i;run.events=[{id:i,kind:kinds[(i-1)%kinds.length]}];RiftFeedback.update(run,false,true);await Promise.resolve();}
  observer.disconnect();window.captionTestRun=run;return messages;
 });
 expect(messages).toEqual(['Attack guarded.','Dodged!.','Ability on cooldown.','Not enough mana.']);
 await expect(page.locator('#rift-captions li')).toHaveCount(3);
 await page.waitForTimeout(4100);
 await page.evaluate(()=>{const run=window.captionTestRun;run.counter++;run.events=[{id:run.counter,kind:'block'}];RiftFeedback.update(run,false,true);});
 await expect(page.locator('#rift-captions [role=status]')).toHaveText('Attack guarded.');
});
