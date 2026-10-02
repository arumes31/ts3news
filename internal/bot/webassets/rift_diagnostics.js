(function(){
  'use strict';
  const number=(value,min,max)=>typeof value==='number'&&Number.isFinite(value)&&value>=min&&value<=max?Math.round(value*100)/100:null;
  const integer=(value,min,max)=>Number.isInteger(value)&&value>=min&&value<=max?value:null;
  const boolean=value=>typeof value==='boolean'?value:null;
  const choice=(value,allowed)=>allowed.includes(value)?value:null;
  let state=null,prepared='';
  // Copy only technical scalars. Never retain a run, build, request or error object.
  function update(run){
    state=run?{schema:integer(run.schema,1,100),status:choice(run.status,['fighting','cleared','complete','defeated','banked','expired']),mission:integer(run.level?.id,1,100000),tier:integer(run.room,0,2)===null?null:run.room+1,paused:boolean(run.paused)}:null;
  }
  function summary(source,key){
    if(!Array.isArray(source?.samples))return null;
    const samples=source.samples.slice(-120).map(sample=>number(sample?.[key],0,60000)).filter(value=>value!==null).sort((a,b)=>a-b);
    if(!samples.length)return null;
    return {samples:samples.length,average:Math.round(samples.reduce((sum,value)=>sum+value,0)/samples.length*100)/100,p95:samples[Math.ceil(samples.length*.95)-1],max:samples[samples.length-1]};
  }
  function report(){
    const renderer=window.RiftRenderer,audio=window.RiftAudio,display=window.RiftDisplay,settings={};
    const options={fps:[30,60],textScale:[1,1.15,1.25],enemyNames:['all','boss','none'],effectIntensity:[.35,.65,1]};
    for(const key of ['shakeIntensity','particleIntensity','motionIntensity','flashIntensity','lootBeamIntensity'])options[key]=[0,.5,1];
    for(const [key,allowed] of Object.entries(options))settings[key]=choice(display?.[key],allowed);
    for(const key of ['damageNumbers','healthBars','particles','adaptiveParticles','lootMotion','hazardContrast','hazardPatterns','projectileShapes','compactHUD','cameraSmooth','hazardLabels','damageMotion','lootSparkle','cleanScreenshot','layoutGrid','minimap','enemyIndicators','skillRange','largeActionBar'])settings[key]=boolean(display?.[key]);
    const sound={muted:boolean(audio?.muted),blocked:typeof audio?.isBlocked==='function'?boolean(audio.isBlocked()):null,mono:boolean(audio?.mono)};
    for(const key of ['effects','ambience','music','voice','interface'])sound[key]=number(audio?.[key],0,1);
    return {
      format:'rift-brawl-diagnostics-v1',
      environment:{viewport_width:integer(innerWidth,1,20000),viewport_height:integer(innerHeight,1,20000),pixel_ratio:number(devicePixelRatio,.1,10),online:boolean(navigator.onLine),visible:document.visibilityState==='visible',fullscreen:!!document.fullscreenElement,reduced_motion:boolean(renderer?.reduced)},
      display:settings,audio:sound,last_confirmed_state:state,
      measurements:{frame_interval_ms:summary(renderer?.frameDiagnostics,'interval'),render_ms:summary(renderer?.frameDiagnostics,'render'),input_total_ms:summary(window.RiftInputDiagnostics,'total'),input_queue_ms:summary(window.RiftInputDiagnostics,'queue'),input_request_ms:summary(window.RiftInputDiagnostics,'request'),max_response_bytes:integer(window.RiftPayloadDiagnostics?.maxResponseBytes,0,1000000000)},
      notes:'Timing summaries cover at most 120 recent debug samples, not a full-session benchmark. Null means unavailable or invalid. Expedition state is the last confirmed snapshot and may be stale after a connection failure. No account, character, reward, URL, storage or log data is included. Nothing is uploaded.'
    };
  }
  const section=document.createElement('div');section.id='rift-diagnostics';section.className='rift-display-settings';
  const heading=document.createElement('h4');heading.textContent='Diagnostic report';
  const description=document.createElement('p');description.className='rift-fine';description.textContent='Prepare a technical summary to review and share when reporting a problem. Includes display and sound settings, viewport, expedition status and available timing summaries. Excludes account details, names, rewards, URLs, logs and browser storage. Nothing is uploaded.';
  const prepare=document.createElement('button');prepare.type='button';prepare.id='rift-prepare-diagnostics';prepare.textContent='Prepare diagnostic preview';
  const label=document.createElement('label');label.htmlFor='rift-diagnostic-preview';label.textContent='Diagnostic JSON preview';
  const preview=document.createElement('textarea');preview.id=label.htmlFor;preview.readOnly=true;preview.rows=10;preview.spellcheck=false;preview.style.cssText='display:block;width:100%;max-width:100%;box-sizing:border-box';
  const download=document.createElement('button');download.type='button';download.id='rift-download-diagnostics';download.textContent='Download diagnostic JSON';download.disabled=true;
  const status=document.createElement('p');status.id='rift-diagnostic-status';status.className='rift-fine';status.setAttribute('role','status');
  prepare.onclick=()=>{try{prepared=JSON.stringify(report(),null,2);preview.value=prepared;download.disabled=false;status.textContent='Preview ready. Review it before downloading or sharing.';}catch(_){prepared='';preview.value='';download.disabled=true;status.textContent='Could not prepare the report. Try again.';}};
  download.onclick=()=>{
    if(!prepared)return;
    let url=null,link=null;
    try{url=URL.createObjectURL(new Blob([prepared],{type:'application/json'}));link=document.createElement('a');link.href=url;link.download='rift-brawl-diagnostics.json';document.body.append(link);link.click();status.textContent='Diagnostic download prepared. Nothing was uploaded.';}
    catch(_){status.textContent='Download unavailable. You can select and copy the preview instead.';}
    finally{link?.remove();if(url)setTimeout(()=>URL.revokeObjectURL(url),1000);}
  };
  section.append(heading,description,prepare,label,preview,download,status);document.querySelector('.rift-settings').append(section);
  window.RiftDiagnostics={update};
})();
