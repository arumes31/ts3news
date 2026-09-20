(function(){
  'use strict';
  const audio=window.RiftAudio,parent=document.querySelector('.rift-settings'),settings=document.createElement('div');settings.className='rift-audio-mix';parent.insertBefore(settings,parent.children[1]);
  const heading=document.createElement('h4');heading.textContent='Audio mix';settings.append(heading);
  const inputs=new Map(),channels=[['effects','Combat effects'],['ambience','Area ambience'],['music','Region music'],['voice','Creature voices'],['interface','Interface sounds']];
  const status=document.createElement('p');status.id='rift-audio-preview-status';status.setAttribute('role','status');
  for(const [key,name] of channels){
    let input=document.getElementById('rift-'+key+'-volume');
    if(!input){const label=document.createElement('label');label.textContent=name;input=document.createElement('input');input.id='rift-'+key+'-volume';label.htmlFor=input.id;input.type='range';input.min=0;input.max=100;label.append(input);settings.append(label);input.addEventListener('input',()=>audio.set(key,Number(input.value)/100));}
    inputs.set(key,input);input.value=Math.round(audio[key]*100);
    const parentLabel=input.closest('label');if(parentLabel)parentLabel.htmlFor=input.id;
    settings.append(input.closest('label'));
    const button=document.createElement('button');button.type='button';button.dataset.audioPreview=key;button.textContent='Preview '+name.toLowerCase();
    button.onclick=async()=>{const played=await audio.preview(key);status.textContent=played?'Previewing '+name.toLowerCase()+'.':audio.muted?'Sound is muted. Unmute to preview.':key==='interface'&&audio.interfaceMuted?'Interface sounds are muted. Unmute this channel to preview.':'Audio is unavailable or blocked by the browser.';};input.closest('label').after(button);
  }
  const steadyLabel=document.createElement('label'),steady=document.createElement('input');steady.id='rift-steady-ambience';steadyLabel.htmlFor=steady.id;steady.type='checkbox';steady.checked=audio.steadyAmbience;steadyLabel.append(steady,document.createTextNode('Steady ambience (no periodic chirps or low tones)'));settings.append(steadyLabel);steady.onchange=()=>audio.set('steadyAmbience',steady.checked);
  const masterMuteLabel=document.createElement('label'),masterMute=document.createElement('input');masterMute.id='rift-master-muted';masterMuteLabel.htmlFor=masterMute.id;masterMute.type='checkbox';masterMute.checked=audio.muted;masterMuteLabel.append(masterMute,document.createTextNode('Mute all sound (master mute, keep saved channel volumes)'));settings.append(masterMuteLabel);masterMute.onchange=()=>audio.set('muted',masterMute.checked);
  const muteLabel=document.createElement('label'),interfaceMute=document.createElement('input');interfaceMute.id='rift-interface-muted';muteLabel.htmlFor=interfaceMute.id;interfaceMute.type='checkbox';interfaceMute.checked=audio.interfaceMuted;muteLabel.append(interfaceMute,document.createTextNode('Mute interface sounds (keep saved volume)'));settings.append(muteLabel);interfaceMute.onchange=()=>audio.set('interfaceMuted',interfaceMute.checked);
  const label=document.createElement('label'),mono=document.createElement('input');mono.id='rift-mono-audio';label.htmlFor=mono.id;mono.type='checkbox';mono.checked=audio.mono;label.append(mono,document.createTextNode('Mono audio (center directional cues)'));settings.append(label);mono.onchange=()=>audio.set('mono',mono.checked);
  const nightLabel=document.createElement('label'),night=document.createElement('input');night.id='rift-night-mode';nightLabel.htmlFor=night.id;night.type='checkbox';night.checked=audio.nightMode||audio.dynamicRange==='night';nightLabel.append(night,document.createTextNode('Night dynamic range (compress loud spikes, boost quiet details)'));settings.append(nightLabel);night.onchange=()=>audio.set('nightMode',night.checked);
  function syncAudioInputs(){for(const [key,input] of inputs)input.value=Math.round(audio[key]*100);masterMute.checked=audio.muted;mono.checked=audio.mono;interfaceMute.checked=audio.interfaceMuted;steady.checked=audio.steadyAmbience;night.checked=audio.nightMode||audio.dynamicRange==='night';}
  window.addEventListener('riftaudiochange',syncAudioInputs);
  const reset=document.createElement('button');reset.id='rift-reset-audio';reset.type='button';reset.textContent='Reset audio mix';reset.onclick=()=>{audio.resetMix();syncAudioInputs();status.textContent='Channel levels restored. Master mute is unchanged.';};settings.append(reset,status);
  const note=document.createElement('p');note.className='rift-fine';note.textContent='All cues and region music are procedural. Mix choices stay on this device. Previews never resume your expedition.';settings.append(note);
})();
