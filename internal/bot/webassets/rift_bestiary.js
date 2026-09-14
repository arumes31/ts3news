(function(){
  'use strict';
  const art=window.AbyssCombatArt;
  const assetURL=path=>path+(window.__ASSET_VER__?'?v='+encodeURIComponent(window.__ASSET_VER__):'');
  function profile(unit){return art.actorProfile({...unit,role:unit.kind==='boss'?'boss':unit.kind});}
  function frame(unit,pose,index){return art.actorFrame({...unit,role:unit.kind==='boss'?'boss':unit.kind},pose,index);}
  let records={},practice=false,recordKey='';
  function update(run){const next=run?.monster_records||{},isPractice=!!run?.practice||!!document.getElementById("rift-app").dataset.practice,key=JSON.stringify([next,isPractice]);if(key===recordKey)return;recordKey=key;records=next;practice=isPractice;if(render.refreshRecords)render.refreshRecords();}
  function render(roster,run){
    const list=document.getElementById('rift-monsters'),search=document.getElementById('rift-monster-search');
    const root=list.closest('details'),panel=document.getElementById('rift-monster-detail');
    const encountered=document.getElementById('rift-monster-seen');
    const tier=document.getElementById('rift-monster-tier'),element=document.getElementById('rift-monster-element'),style=document.getElementById('rift-monster-style');
    const poseSelect=document.getElementById('rift-monster-pose'),preview=document.getElementById('rift-monster-preview');
    const normalize=value=>String(value).normalize('NFD').replace(/[\u0300-\u036f]/g,'').toLowerCase().replace(/[^\p{L}\p{N}]+/gu,' ').trim();
    const attackStyle=unit=>unit.kind==='boss'?'Area attacks':unit.kind==='archer'?'Ranged':unit.kind==='treasure'?'Fleeing':'Melee';
    const elementName=unit=>unit.element||'physical';
    const numeric=value=>new Intl.NumberFormat(undefined,{maximumFractionDigits:1}).format(Number(value)||0);
    function options(select,values){
      const saved=select.value,first=select.options[0];select.replaceChildren(first);
      [...new Set(values)].sort().forEach(value=>{const option=document.createElement('option');option.value=value;option.textContent=value;select.append(option);});
      if(values.includes(saved))select.value=saved;
    }
    options(tier,roster.map(unit=>unit.tier));options(element,roster.map(elementName));options(style,roster.map(attackStyle));
    if(render.cleanup)render.cleanup();
    let selected=null,opener=null,timer=null,tick=0,built=false;
    panel.hidden=true;
    function paint(){
      if(!selected)return;
      const pose=frame(selected,poseSelect.value,window.RiftRenderer?.reduced?0:tick++);
      preview.style.backgroundImage='url("'+assetURL(pose.asset)+'")';preview.style.backgroundPosition=pose.position;preview.style.backgroundSize=pose.size;preview.style.transform=pose.transform||'';
    }
    function animate(){
      clearInterval(timer);timer=null;
      if(!selected||!root.open||document.hidden)return;
      paint();if(!window.RiftRenderer?.reduced)timer=setInterval(paint,180);
    }
    function onToggle(){if(root.open)buildCards();animate();}
    root.addEventListener('toggle',onToggle);document.addEventListener('visibilitychange',animate);
    window.addEventListener('riftmotionchange',animate);
    render.cleanup=()=>{clearInterval(timer);root.removeEventListener('toggle',onToggle);document.removeEventListener('visibilitychange',animate);window.removeEventListener('riftmotionchange',animate);};
    poseSelect.onchange=()=>{tick=0;paint();};
    function close(){window.RiftAudio.cancelPreview();selected=null;panel.hidden=true;animate();if(opener&&!opener.closest('article').hidden)opener.focus();else search.focus();}
    document.getElementById('rift-monster-close').onclick=close;
    panel.onkeydown=event=>{if(event.key==='Escape'){event.preventDefault();event.stopPropagation();close();}};
    function showRecord(){
      if(!selected)return;const record=records[selected.art_key];
      document.getElementById('rift-monster-record').textContent=practice?'Campaign records are shown in campaign mode. Practice does not add encounters or defeats.':record?'First recorded encounter: '+new Date(record.first_seen_ms).toLocaleString()+' · Defeats: '+record.defeats+'. Older fights may be missing.':'No recorded encounter. Older fights may be missing.';
    }
    function statsFor(unit){
      const values=[['Abyss tier',unit.tier],['Element',elementName(unit)],['Attack style',attackStyle(unit)],['Health',numeric(unit.max_hp)],['Damage',numeric(unit.damage)],['Armor reduction',numeric(unit.armor*100)+'%'],['Speed',numeric(unit.speed)],['Projectile',unit.kind==='archer'||unit.kind==='boss'?unit.shot||'arrow':'None']];
      if(unit.training)values.push(['Attack windup',new Intl.NumberFormat(undefined,{maximumFractionDigits:2}).format(unit.training.windup_seconds)+' s'],['Knockdown',unit.training.resists_knockdown?'Resistant':'Third basic strike can knock down'],['Interruptible',unit.training.interruptible?'Yes: third basic strike or ice':'No: resists basic-combo and ice interrupts']);
      return values;
    }
    function inspect(unit,button){
      window.RiftAudio.cancelPreview();selected=unit;opener=button;tick=0;poseSelect.value='idle';panel.hidden=false;
      const title=document.getElementById('rift-monster-title');title.textContent=unit.name;
      const stats=document.getElementById('rift-monster-stats');stats.replaceChildren();
      const values=statsFor(unit);
      values.forEach(([label,value])=>{const row=document.createElement('div'),dt=document.createElement('dt'),dd=document.createElement('dd');dt.textContent=label;dd.textContent=value;row.append(dt,dd);stats.append(row);});
      document.getElementById('rift-monster-tip').textContent=unit.kind==='boss'?'Watch the marked slam area and jump or move clear. This boss alternates slams with aimed projectiles. Use the recovery after a slam to strike.':unit.kind==='archer'?'Change lanes to evade aimed shots, then close the gap during recovery.':unit.kind==='treasure'?'Flees when you approach. Drive it toward the arena edge to stop its retreat.':'Move out of its lane during the windup, then strike during recovery. A third basic strike can knock it down.';
      const sounds=document.getElementById('rift-monster-sounds'),soundStatus=document.getElementById('rift-monster-sound-status');sounds.replaceChildren();soundStatus.textContent='';
      const cues=unit.kind==='boss'?[['roar','boss_roar'],['slam','slam'],['projectile',unit.shot||'arrow'],['defeat','boss_death']]:unit.kind==='archer'?[['projectile',unit.shot||'arrow'],['defeat','archer_death']]:[['attack',unit.kind+'_attack'],['defeat',unit.kind+'_death']];
      for(const [label,cue] of cues){const previewButton=document.createElement('button');previewButton.type='button';previewButton.textContent='Preview '+label;
        previewButton.onclick=async()=>{previewButton.disabled=true;try{const played=await window.RiftAudio.previewCue(cue);if(selected===unit&&!panel.hidden)soundStatus.textContent=played?'Previewing '+label+' for '+unit.name+'.':window.RiftAudio.muted?'Sound is muted. Unmute to preview.':'Audio is unavailable or blocked by the browser.';}finally{previewButton.disabled=false;}};
        sounds.append(previewButton);
      }
      showRecord();animate();title.focus();
    }
    const comparison=document.getElementById('rift-monster-comparison'),left=document.getElementById('rift-compare-left'),right=document.getElementById('rift-compare-right');
    for(const select of [left,right]){const saved=select.value;select.replaceChildren(new Option('Choose a creature',''));for(const unit of roster)select.append(new Option(unit.name,unit.art_key));if(roster.some(unit=>unit.art_key===saved))select.value=saved;}
    function compare(){
      comparison.replaceChildren();const a=roster.find(unit=>unit.art_key===left.value),b=roster.find(unit=>unit.art_key===right.value),message=document.getElementById('rift-compare-status');
      if(!a||!b||a.art_key===b.art_key){comparison.hidden=true;message.textContent=a&&b?'Choose two different creatures.':'Choose two creatures to compare their base stats.';return;}
      comparison.hidden=false;message.textContent='Different values are highlighted. Mission difficulty and room scaling still apply.';
      const caption=document.createElement('caption');caption.textContent=a.name+' compared with '+b.name;comparison.append(caption);
      const head=document.createElement('thead'),header=document.createElement('tr');for(const label of ['Trait',a.name,b.name]){const cell=document.createElement('th');cell.scope='col';cell.textContent=label;header.append(cell);}head.append(header);comparison.append(head);
      const body=document.createElement('tbody'),other=new Map(statsFor(b));for(const [label,value] of statsFor(a)){
        const row=document.createElement('tr'),title=document.createElement('th');title.scope='row';title.textContent=label;row.append(title);const compared=other.get(label)||'Unavailable';
        for(const text of [value,compared]){const cell=document.createElement('td');cell.textContent=text;row.append(cell);}if(value!==compared)row.className='rift-compare-different';body.append(row);
      }comparison.append(body);
    }
    left.onchange=compare;right.onchange=compare;document.getElementById('rift-compare-clear').onclick=()=>{left.value='';right.value='';compare();left.focus();};compare();
    document.getElementById('rift-monster-count').textContent=roster.length+' monsters';
    list.replaceChildren();
    function buildCards(){
      if(built)return;built=true;
      roster.forEach(unit=>{
        const item=document.createElement('article');item.setAttribute('role','listitem');item.dataset.search=normalize(unit.name+' '+unit.tier+' '+elementName(unit)+' '+attackStyle(unit));item.dataset.artKey=unit.art_key;item.dataset.tier=unit.tier;item.dataset.element=elementName(unit);item.dataset.style=attackStyle(unit);
        const sprite=document.createElement('span'),pose=frame(unit,'idle',0);sprite.className='rift-monster-art';sprite.setAttribute('aria-hidden','true');sprite.style.backgroundImage='url("'+assetURL(pose.asset)+'")';sprite.style.backgroundPosition=pose.position;sprite.style.backgroundSize=pose.size;
        const body=document.createElement('div'),name=document.createElement('strong'),detail=document.createElement('small');name.textContent=unit.name;detail.textContent=unit.tier+' · '+(unit.kind==='boss'?'Area attacks':unit.kind==='archer'?'Ranged':unit.kind==='treasure'?'Fleeing':'Melee');
        const button=document.createElement('button');button.type='button';button.textContent='Inspect';button.setAttribute('aria-label','Inspect '+unit.name);button.onclick=()=>inspect(unit,button);
        body.append(name,detail,button);item.append(sprite,body);list.append(item);
      });
      filter();
    }
    function filter(){let visible=0;for(const item of list.children){item.hidden=(encountered.checked&&!Object.hasOwn(records,item.dataset.artKey))||!item.dataset.search.includes(normalize(search.value))||(tier.value&&item.dataset.tier!==tier.value)||(element.value&&item.dataset.element!==element.value)||(style.value&&item.dataset.style!==style.value);if(!item.hidden)visible++;}const empty=document.getElementById('rift-monsters-empty');empty.hidden=visible>0;empty.textContent=encountered.checked?Object.keys(records).length?'No recorded creatures match these filters. Clear filters to see the full roster.':'No recorded encounters yet. Start a campaign expedition to record monsters. Older fights may be missing.':'No matching monsters. Clear filters or try another name.';document.getElementById('rift-monster-matches').textContent=visible+' of '+roster.length+' monsters';}
    encountered.onchange=filter;
    render.refreshRecords=()=>{encountered.disabled=practice;if(practice)encountered.checked=false;showRecord();if(built)filter();};
    update(run);render.refreshRecords();
    search.oninput=filter;[tier,element,style].forEach(select=>select.onchange=filter);
    document.getElementById('rift-monster-clear').onclick=()=>{search.value='';tier.value='';element.value='';style.value='';encountered.checked=false;filter();search.focus();};onToggle();
  }
  window.RiftBestiary={profile,frame,render,update,assetURL,assets:art.atlasAssets};
})();
