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
})();
