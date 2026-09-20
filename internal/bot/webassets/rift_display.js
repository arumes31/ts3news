(function(){
  'use strict';
  const root=document.getElementById('rift-app');
  const defaults={optionalCombatText:true,shakeIntensity:0,personalRecords:true,damageNumbers:true,enemyNames:'all',healthBars:true,particles:true,lootMotion:true,hazardContrast:false,hazardPatterns:true,projectileShapes:true,textScale:1,compactHUD:false,cameraSmooth:true,fps:60,hazardLabels:true,damageMotion:true,lootSparkle:true,particleIntensity:1,motionIntensity:1,flashIntensity:1,effectIntensity:1,cleanScreenshot:false,enemyIndicators:true,skillRange:false,largeActionBar:false};
  const choices={shakeIntensity:[0,0.5,1],enemyNames:['all','boss','none'],textScale:[1,1.15,1.25],fps:[30,60],particleIntensity:[0,0.5,1],motionIntensity:[0,0.5,1],flashIntensity:[0,0.5,1],effectIntensity:[0.35,0.65,1]};
  const values={...defaults};
  try{
    const saved=JSON.parse(localStorage.getItem('riftDisplay'));
    if(saved?.version===1)for(const key of Object.keys(defaults)){
      if(choices[key]?choices[key].includes(saved[key]):typeof saved[key]==='boolean')values[key]=saved[key];
    }
  }catch(_){}
  const presets={
    balanced:{label:'Balanced',description:'Standard text, all enemy names, health bars, moving damage numbers and hazard labels. Full decoration, spell opacity and transitions, smooth camera and 60 FPS.',values:defaults},
    minimal:{label:'Minimal distractions',description:'Standard text, health bars and hazard labels. Boss names only, no damage numbers, decoration or transition fade. Soft spell effects, compact HUD, smooth camera and 60 FPS.',values:{...defaults,enemyNames:'boss',damageNumbers:false,particles:false,lootMotion:false,lootSparkle:false,particleIntensity:0,motionIntensity:0,flashIntensity:0,effectIntensity:0.35,damageMotion:false,compactHUD:true}},
    accessible:{label:'Clearer battlefield',description:'Larger text (125%), all names, health bars, static damage numbers, hazard labels, tactile fill patterns and shaped projectiles. Strong hazard outlines, no decoration, fade or camera smoothing. Soft spell effects, full HUD and 60 FPS.',values:{...defaults,textScale:1.25,hazardContrast:true,hazardPatterns:true,projectileShapes:true,particles:false,lootMotion:false,lootSparkle:false,particleIntensity:0,motionIntensity:0,flashIntensity:0,effectIntensity:0.35,damageMotion:false,cameraSmooth:false,largeActionBar:true}},
    lowPower:{label:'Lower power',description:'Standard text, health bars and hazard labels. Boss names only, no damage numbers, decoration or fade. Soft spell effects, compact HUD, direct camera and 30 FPS.',values:{...defaults,fps:30,particles:false,lootMotion:false,lootSparkle:false,particleIntensity:0,motionIntensity:0,flashIntensity:0,effectIntensity:0.35,damageMotion:false,enemyNames:'boss',damageNumbers:false,compactHUD:true,cameraSmooth:false}},
    cinematic:{label:'Cinematic effects',description:'Full decoration, moving damage numbers, spell opacity and transition fades. Standard text, boss names, health bars and hazard labels. Compact HUD, smooth camera and 60 FPS; reduced motion still takes priority.',values:{...defaults,enemyNames:'boss',compactHUD:true}}
  };
  for(const preset of Object.values(presets))preset.description+=' Screen shake is off.';
  const section=document.createElement('div');section.className='rift-display-settings';
  const heading=document.createElement('h4');heading.textContent='Battlefield display';section.append(heading);
  const controls=new Map();
  const summary=document.createElement('span');summary.id='rift-settings-summary';document.querySelector('.rift-settings > summary').append(summary);
  function refreshSummary(){
    const audio=window.RiftAudio,reduced=document.getElementById('rift-reduced').checked;
    const shake=reduced||values.motionIntensity===0?'Off':values.shakeIntensity===1?'Full':values.shakeIntensity===0.5?'Gentle':'Off';
    const mix=[['effects','Effects'],['ambience','Ambience'],['music','Music'],['voice','Voices'],['interface','Interface']].map(([key,label])=>{if(key==='interface'&&audio.interfaceMuted)return 'Interface muted ('+Math.round(audio[key]*100)+'%)';if(key==='music'&&(audio.streamerMusic||audio.musicPreset==='streamer'))return 'Music off (streamer preset)';return label+' '+Math.round(audio[key]*100)+'%';}).join(' · ');
    summary.textContent=(values.cleanScreenshot?'Clean screenshot':values.compactHUD?'Compact':'Full')+' HUD · Text '+Math.round(values.textScale*100)+'% · '+values.fps+' FPS · '+(reduced?'Reduced effects':'Motion '+(values.motionIntensity===0?'Still':values.motionIntensity===0.5?'Gentle':'Full'))+' · Shake '+shake+' — '+(audio.muted?'Sound muted':'Sound enabled')+' · '+(audio.mono?'Mono':'Stereo')+' · '+(audio.nightMode||audio.dynamicRange==='night'?'Night range':'Standard range')+' · '+mix;
  }
  window.addEventListener('riftaudiochange',refreshSummary);window.addEventListener('riftmotionchange',refreshSummary);
  function selectControl(id,label,options){
    const wrapper=document.createElement('label'),select=document.createElement('select');wrapper.textContent=label;select.id=id;wrapper.htmlFor=id;
    options.forEach(([value,text])=>{const option=document.createElement('option');option.value=value;option.textContent=text;select.append(option);});
    wrapper.append(select);section.append(wrapper);return select;
  }
  const preset=selectControl('rift-display-preset','Preset',Object.entries(presets).map(([key,p])=>[key,p.label]));
  const description=document.createElement('p');description.id='rift-preset-description';description.className='rift-fine';description.textContent=presets.balanced.description;section.append(description);
  preset.setAttribute('aria-describedby',description.id);preset.onchange=()=>description.textContent=presets[preset.value].description;
  const apply=document.createElement('button');apply.id='rift-apply-preset';apply.type='button';apply.textContent='Apply preset';section.append(apply);
  const definitions=[
    ['textScale','rift-text-scale','HUD text size',[[1,'100%'],[1.15,'115%'],[1.25,'125%']]],
    ['enemyNames','rift-enemy-names','Enemy names',[['all','All'],['boss','Bosses only'],['none','Hidden']]],
    ['fps','rift-render-rate','Rendering limit',[[60,'60 FPS'],[30,'30 FPS']]],
    ['particleIntensity','rift-particle-intensity','Background particle density',[[0,'Off'],[0.5,'Half'],[1,'Full']]],
    ['shakeIntensity','rift-shake-intensity','Screen shake',[[0,'Off'],[0.5,'Gentle'],[1,'Full']]],
    ['motionIntensity','rift-motion-intensity','Decorative motion',[[0,'Still'],[0.5,'Gentle'],[1,'Full']]],
    ['flashIntensity','rift-flash-intensity','Transition fade intensity',[[0,'Off'],[0.5,'Gentle'],[1,'Full']]],
    ['effectIntensity','rift-effect-intensity','Spell effect opacity',[[0.35,'Soft'],[0.65,'Medium'],[1,'Full']]],
    ['optionalCombatText','rift-optional-combat-text','Optional combat text (loot labels and gold popups)'],
    ['healthBars','rift-enemy-health','Enemy health bars'],['damageNumbers','rift-damage-numbers','Damage numbers'],
    ['damageMotion','rift-damage-motion','Moving damage numbers'],['hazardLabels','rift-hazard-labels','Hazard labels'],['lootSparkle','rift-loot-sparkle','Loot sparkle'],
    ['particles','rift-background-particles','Background particles'],['lootMotion','rift-loot-motion','Loot bobbing'],
    ['hazardContrast','rift-hazard-contrast','Strong hazard outlines'],['hazardPatterns','rift-hazard-patterns','Hazard fill patterns'],['projectileShapes','rift-projectile-shapes','Hostile vs friendly projectile shapes'],['cameraSmooth','rift-camera-smoothing','Smooth camera'],
    ['compactHUD','rift-compact-hud','Compact HUD'],['personalRecords','rift-personal-records','Personal records in HUD'],
    ['cleanScreenshot','rift-clean-screenshot','Clean screenshot mode (hide HUD)'],
    ['largeActionBar','rift-large-action-bar','Large-target action bar'],
    ['enemyIndicators','rift-enemy-indicators','Off-screen enemy direction indicators'],
    ['skillRange','rift-show-skill-range','Equipped skill range indicators']
  ];
  function save(){try{localStorage.setItem('riftDisplay',JSON.stringify({version:1,...values}));}catch(_){} }
  function refresh(){
    refreshSummary();
    root.classList.toggle('rift-hide-records',!values.personalRecords);
    root.style.setProperty('--rift-hud-scale',values.textScale);root.classList.toggle('rift-compact-hud',values.compactHUD);
    root.classList.toggle('rift-clean-screenshot',values.cleanScreenshot);
    root.classList.toggle('rift-large-action-bar',Boolean(values.largeActionBar));
    const screenshotBtn=document.getElementById('rift-screenshot-toggle');
    if(screenshotBtn)screenshotBtn.setAttribute('aria-pressed',String(values.cleanScreenshot));
    for(const [key,control] of controls)if(control.type==='checkbox')control.checked=values[key];else control.value=values[key];
  }
  definitions.forEach(([key,id,label,options])=>{
    let control;
    if(options)control=selectControl(id,label,options);
    else{const wrapper=document.createElement('label');wrapper.textContent=label;control=document.createElement('input');control.type='checkbox';control.id=id;wrapper.htmlFor=id;wrapper.append(control);section.append(wrapper);}
    controls.set(key,control);control.onchange=()=>{values[key]=control.type==='checkbox'?control.checked:typeof defaults[key]==='number'?Number(control.value):control.value;save();refresh();};
  });
  apply.onclick=()=>{Object.assign(values,presets[preset.value].values);save();refresh();};
  function resetDisplay(){Object.assign(values,defaults);preset.value='balanced';description.textContent=presets.balanced.description;save();refresh();}
  const reset=document.createElement('button');reset.id='rift-reset-display';reset.type='button';reset.textContent='Reset display';reset.onclick=resetDisplay;
  const masterReset=document.createElement('button');masterReset.id='rift-reset-preferences';masterReset.type='button';masterReset.textContent='Reset all preferences';
  const masterStatus=document.createElement('p');masterStatus.id='rift-preferences-status';masterStatus.className='rift-fine';masterStatus.setAttribute('role','status');
  masterReset.onclick=()=>{
    resetDisplay();
    if(window.RiftAudio?.resetMix)window.RiftAudio.resetMix();
    const reduced=document.getElementById('rift-reduced');
    if(reduced&&reduced.checked){reduced.checked=false;try{localStorage.removeItem('riftReducedMotion');}catch(_){}reduced.dispatchEvent(new Event('change'));}
    const confirmBoss=document.getElementById('rift-confirm-boss');
    if(confirmBoss&&confirmBoss.checked){confirmBoss.checked=false;try{localStorage.removeItem('riftConfirmBoss');}catch(_){}confirmBoss.dispatchEvent(new Event('change'));}
    const pauseBoss=document.getElementById('rift-pause-boss-room');
    if(pauseBoss&&pauseBoss.checked){pauseBoss.checked=false;try{localStorage.removeItem('riftPauseBossRoom');}catch(_){}pauseBoss.dispatchEvent(new Event('change'));}
    const pauseRegion=document.getElementById('rift-pause-new-region');
    if(pauseRegion&&pauseRegion.checked){pauseRegion.checked=false;try{localStorage.removeItem('riftPauseNewRegion');}catch(_){}pauseRegion.dispatchEvent(new Event('change'));}
    const fsControls=document.getElementById('rift-fullscreen-controls');
    if(fsControls&&fsControls.checked){fsControls.checked=false;try{localStorage.removeItem('riftFullscreenControls');}catch(_){}fsControls.dispatchEvent(new Event('change'));}
    const transitionDelay=document.getElementById('rift-transition-delay');
    if(transitionDelay&&transitionDelay.value!=='1.2'){transitionDelay.value='1.2';try{localStorage.removeItem('riftTransitionDelay');}catch(_){}transitionDelay.dispatchEvent(new Event('change'));}
    const collapse=document.getElementById('rift-campaign-start-collapsed');
    if(collapse&&collapse.checked){collapse.checked=false;try{const saved=JSON.parse(localStorage.getItem('riftCampaignView')||'{}');if(saved&&typeof saved==='object'){saved.startCollapsed=false;localStorage.setItem('riftCampaignView',JSON.stringify(saved));}}catch(_){}collapse.dispatchEvent(new Event('change'));}
    const readout=document.getElementById('rift-input-readout-enabled');
    if(readout&&readout.checked){readout.checked=false;try{localStorage.removeItem('riftInputReadout');}catch(_){}readout.dispatchEvent(new Event('change'));}
    const combatCaptions=document.getElementById('rift-combat-captions');
    if(combatCaptions&&combatCaptions.checked){combatCaptions.checked=false;try{localStorage.removeItem('riftCaptions');}catch(_){}combatCaptions.dispatchEvent(new Event('change'));}
    const dirCaptions=document.getElementById('rift-directional-threat-captions');
    if(dirCaptions&&dirCaptions.checked){dirCaptions.checked=false;try{localStorage.removeItem('riftDirectionalCaptions');}catch(_){}dirCaptions.dispatchEvent(new Event('change'));}
    const previewAuto=document.getElementById('rift-mission-preview-auto');
    if(previewAuto&&previewAuto.checked){previewAuto.checked=false;try{localStorage.removeItem('riftPreviewOpen');}catch(_){}previewAuto.dispatchEvent(new Event('change'));}
    masterStatus.textContent='All sound, display, and workflow preferences restored to defaults.';
    refreshSummary();
  };
  const note=document.createElement('p');note.className='rift-fine';note.textContent='Display choices are saved on this device and apply immediately. Screen shake defaults to off; reduced visual effects and Still decorative motion disable it. Player and boss vitals stay visible. Reduced visual effects overrides decorative motion. Presets change only the display settings listed here and restore personal records. Hiding HUD records keeps mission-card history and combat breakdowns available.';section.append(reset,masterReset,masterStatus,note);
  document.querySelector('.rift-settings').append(section);
  refresh();window.RiftDisplay=Object.assign(values,{reset:resetDisplay,toggleScreenshot(){values.cleanScreenshot=!values.cleanScreenshot;save();refresh();return values.cleanScreenshot;}});
})();
