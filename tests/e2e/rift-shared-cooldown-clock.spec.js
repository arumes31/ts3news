const {test,expect}=require('@playwright/test');
test('all cooldown buttons use one snapshot without allocating timers',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');
 await expect(page.locator('#rift-start')).toBeEnabled();
 const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 const result=await page.evaluate(run=>{
  const skills=[...run.build.skills,...run.build.signatures,run.build.ultimate];
  const buttons=[...document.querySelectorAll('#rift-skills button,#rift-signatures button')];
  if(skills.some(s=>!s)||skills.length!==buttons.length)throw new Error('Fixture must contain optional, class and ultimate abilities');
  run.paused=false;run.player.mana=100;
  window.RiftHUD.update(run,true);
  const timeout=window.setTimeout,interval=window.setInterval,raf=window.requestAnimationFrame;
  let timers=0,frames=0;
  window.setTimeout=function(...args){timers++;return timeout.apply(this,args);};
  window.setInterval=function(...args){timers++;return interval.apply(this,args);};
  window.requestAnimationFrame=function(...args){frames++;return raf.apply(this,args);};
  const samples=[];
  try{
   for(const remaining of [8,4,1.2,0]){
    run.clock+=1;
    for(const skill of skills){skill.cooldown=8;run.skill_timers[skill.id]=remaining;}
    for(let i=0;i<20;i++)window.RiftHUD.update(run,true);
    samples.push({remaining,buttons:buttons.map(button=>({label:button.getAttribute('aria-label'),progress:button.querySelector('.rift-cooldown-ring').style.getPropertyValue('--cooldown-progress'),hidden:button.querySelector('.rift-cooldown-ring').hidden}))});
   }
   run.paused=true;
   for(const skill of skills)run.skill_timers[skill.id]=4;
   window.RiftHUD.update(run,false);
   samples.push({paused:true,buttons:buttons.map(button=>({label:button.getAttribute('aria-label'),progress:button.querySelector('.rift-cooldown-ring').style.getPropertyValue('--cooldown-progress')}))});
  }finally{window.setTimeout=timeout;window.setInterval=interval;window.requestAnimationFrame=raf;}
  return {timers,frames,samples,roles:buttons.map(button=>button.dataset.abilityRole)};
 },run);
 expect(result.roles).toEqual(expect.arrayContaining(['optional','builder','finisher','ultimate']));
 expect(result.timers).toBe(0);expect(result.frames).toBe(0);
 for(const sample of result.samples)for(const button of sample.buttons){
  if(sample.paused){expect(button.label).toContain('Paused');expect(button.progress).toBe('50%');continue;}
  expect(parseFloat(button.progress)).toBeCloseTo(sample.remaining/8*100,5);
  expect(button.hidden).toBe(sample.remaining===0);
  expect(button.label).toContain(sample.remaining>0?sample.remaining.toFixed(1)+' seconds cooldown':'Ready');
 }
});
