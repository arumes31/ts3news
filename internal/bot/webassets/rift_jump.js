(function(){
  'use strict';
  const choices=[0,100,200,300];let milliseconds=0,until=0,identity='',jumps=0;
  try{const saved=JSON.parse(localStorage.getItem('riftJumpBuffer'));if(choices.includes(saved))milliseconds=saved;}catch(_){}
  const label=document.createElement('label'),select=document.createElement('select'),description=document.createElement('p');label.className='rift-jump-setting';select.id='rift-jump-buffer';description.id='rift-jump-buffer-description';select.setAttribute('aria-describedby',description.id);
  for(const value of choices){const option=document.createElement('option');option.value=value;option.textContent=value?value+' ms':'Off';select.append(option);}
  select.value=milliseconds;label.append(document.createTextNode('Early jump buffer'),select);description.textContent='Carry a slightly early jump press through landing or jump recovery. Off keeps normal jump timing. Holding jump still repeats when ready. Pausing clears buffered jumps.';document.querySelector('.rift-settings').append(label,description);
  select.onchange=()=>{milliseconds=Number(select.value);until=0;try{localStorage.setItem('riftJumpBuffer',JSON.stringify(milliseconds));}catch(_){}window.dispatchEvent(new Event('riftintentchange'));};
  window.RiftJump={
    press(){until=performance.now()+milliseconds;},
    reset(){until=0;},
    input(requested){return !!requested||performance.now()<until;},
    sync(run,replay){const next=[run.id,run.level?.id,run.room].join(':'),count=run.stats?.jumps||0;if(replay||identity!==next||run.paused||!['fighting','cleared'].includes(run.status)||count!==jumps)until=0;identity=next;jumps=count;}
  };
})();
