(function(){
  'use strict';
  const label=document.createElement('label'),enabled=document.createElement('input');enabled.type='checkbox';enabled.id='rift-vibration';try{enabled.checked=JSON.parse(localStorage.getItem('riftVibration'))===true;}catch(_){}label.append(enabled,document.createTextNode('Controller vibration on hits'));document.getElementById('rift-gamepad-panel').append(label);
  let previous=null,actuator=null,generation=0,lastPulse=-Infinity;
  function stop(){generation++;const current=actuator;actuator=null;if(current)try{Promise.resolve(current.reset()).catch(()=>{});}catch(_){} }
  enabled.onchange=()=>{try{localStorage.setItem('riftVibration',JSON.stringify(enabled.checked));}catch(_){}stop();};
  function update(run,replay,playing){
    const next={identity:[run.id,run.level?.id,run.room].join(':'),dealt:run.stats?.damage_dealt||0,taken:run.stats?.damage_taken||0},old=previous;previous=next;
    if(!old||old.identity!==next.identity||replay||!playing||run.paused||document.hidden||!enabled.checked||!['fighting','cleared'].includes(run.status)||next.dealt<old.dealt||next.taken<old.taken){stop();return;}
    const hurt=next.taken>old.taken,hit=next.dealt>old.dealt,now=performance.now();if((!hurt&&!hit)||now-lastPulse<120)return;
    const current=window.RiftGamepad.actuator();if(!current)return;lastPulse=now;actuator=current;const token=++generation;
    try{Promise.resolve(current.playEffect('dual-rumble',{duration:hurt?110:55,startDelay:0,strongMagnitude:hurt?.45:.08,weakMagnitude:hurt?.25:.22})).then(()=>{if(token===generation)actuator=null;},()=>{if(token===generation)actuator=null;});}catch(_){if(token===generation)actuator=null;}
  }
  window.addEventListener('pagehide',stop);document.addEventListener('visibilitychange',()=>{if(document.hidden)stop();});
  window.RiftHaptics={update,stop};
})();
