(function(){
 'use strict';
 const $=id=>document.getElementById(id),panel=$('rift-walkthrough');
 let step=0,dismissed=false;
 try{const saved=JSON.parse(localStorage.getItem('riftWalkthrough'));dismissed=saved?.version===1&&saved.completed===true;}catch(_){}
 const key=action=>window.RiftControls.label(action);
 const steps=[
  ['Move and jump',()=> 'Move with '+['up','left','down','right'].map(key).join(', ')+'. Jump with '+key('jump')+' to cross low cover and evade low attacks. You can start playing at any time.'],
  ['Fight and guard',()=> 'Attack with '+key('attack')+' for a three-strike combo. Guard with '+key('guard')+' while facing an attacker. Attacks from behind bypass guard. '+key('pause')+' pauses or resumes.'],
  ['Use your Abyss build',()=> 'Your Abyss subclass, equipment and selected skills come with you. Use '+key('signature0')+' to build class charges and '+key('signature1')+' to spend them. Equipped skills use '+['skill0','skill1','skill2'].map(key).join(', ')+'. An equipped ultimate uses '+key('ultimate')+'.'],
  ['Bank your rewards',()=> 'Clear each tier to reach a checkpoint. Bank & continue secures rewards before moving on; Bank & leave ends the expedition. Seamless tiers can bank and continue automatically. Defeat loses unbanked loot, while banked rewards stay safe.']
 ];
 function render(){const [title,copy]=steps[step];$('rift-walkthrough-step').textContent='Quick start · '+(step+1)+' of '+steps.length;$('rift-walkthrough-title').textContent=title;$('rift-walkthrough-copy').textContent=copy();$('rift-walkthrough-back').disabled=step===0;$('rift-walkthrough-next').textContent=step===steps.length-1?'Finish':'Next';}
 function finish(){try{localStorage.setItem('riftWalkthrough',JSON.stringify({version:1,completed:true}));}catch(_){}panel.hidden=true;$('rift-controls-open').focus({preventScroll:true});}
 $('rift-walkthrough-back').onclick=()=>{if(step>0){step--;render();}};
 $('rift-walkthrough-next').onclick=()=>{if(step===steps.length-1)finish();else{step++;render();}};
 $('rift-walkthrough-skip').onclick=finish;
 const review=document.createElement('button');review.id='rift-walkthrough-review';review.type='button';review.textContent='Review controls walkthrough';document.querySelector('.rift-settings').append(review);
 review.onclick=()=>{step=0;panel.hidden=false;render();$('rift-walkthrough-next').focus();};
 window.addEventListener('riftbindingschange',render);
 render();panel.hidden=dismissed;

 const guide=$('rift-checkpoint-guide'),learned={auto:false,manual:false};let guideMode=null,guideRun='';
 try{const saved=JSON.parse(localStorage.getItem('riftCheckpointHelp'));if(saved?.version===1){learned.auto=saved.auto===true;learned.manual=saved.manual===true;}}catch(_){}
 $('rift-checkpoint-guide-dismiss').onclick=()=>{if(guideMode){learned[guideMode]=true;try{localStorage.setItem('riftCheckpointHelp',JSON.stringify({version:1,...learned}));}catch(_){}}guideMode=null;guide.hidden=true;$('rift-controls-open').focus({preventScroll:true});};
 function update(run){
  const mode=$('rift-auto').checked?'auto':'manual';
  if(guideRun!==run.id){guideRun=run.id;guideMode=null;}
  if(guideMode!==mode)guideMode=null;
  if(run.status==='cleared'&&!learned[mode])guideMode=mode;
  guide.hidden=!guideMode||!['fighting','cleared'].includes(run.status);
  if(guide.hidden)return;
  const copy=mode==='auto'?'Seamless tiers: while playing, a cleared room banks after the countdown and continues automatically. The final tier continues to the next mission, through mission 100. Pause to stop the countdown, use Continue now to move sooner, or Bank & leave to stop. Only confirmed banking secures rewards.':'Manual checkpoints: the game waits after each clear. Bank & continue secures rewards and opens the next tier; at the final tier, Bank & finish expedition completes this mission. Bank & leave secures current rewards and ends the expedition. Only confirmed banking secures rewards.';
  if($('rift-checkpoint-guide-copy').textContent!==copy)$('rift-checkpoint-guide-copy').textContent=copy;
 }
 window.RiftOnboarding={update};
})();
