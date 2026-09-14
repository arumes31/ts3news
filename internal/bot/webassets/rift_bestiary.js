(function(){
  'use strict';
  const art=window.AbyssCombatArt;
  const assetURL=path=>path+(window.__ASSET_VER__?'?v='+encodeURIComponent(window.__ASSET_VER__):'');
  function profile(unit){return art.actorProfile({...unit,role:unit.kind==='boss'?'boss':unit.kind});}
  function frame(unit,pose,index){return art.actorFrame({...unit,role:unit.kind==='boss'?'boss':unit.kind},pose,index);}
  function render(roster){
    const list=document.getElementById('rift-monsters'),search=document.getElementById('rift-monster-search');
    const root=list.closest('details'),panel=document.getElementById('rift-monster-detail');
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
    let selected=null,opener=null,timer=null,tick=0;
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
    root.addEventListener('toggle',animate);document.addEventListener('visibilitychange',animate);
    window.addEventListener('riftmotionchange',animate);
    render.cleanup=()=>{clearInterval(timer);root.removeEventListener('toggle',animate);document.removeEventListener('visibilitychange',animate);window.removeEventListener('riftmotionchange',animate);};
    poseSelect.onchange=()=>{tick=0;paint();};
    function close(){selected=null;panel.hidden=true;animate();if(opener&&!opener.closest('article').hidden)opener.focus();else search.focus();}
    document.getElementById('rift-monster-close').onclick=close;
    panel.onkeydown=event=>{if(event.key==='Escape'){event.preventDefault();event.stopPropagation();close();}};
    function inspect(unit,button){
      selected=unit;opener=button;tick=0;poseSelect.value='idle';panel.hidden=false;
      const title=document.getElementById('rift-monster-title');title.textContent=unit.name;
      const stats=document.getElementById('rift-monster-stats');stats.replaceChildren();
      const values=[['Tier',unit.tier],['Element',elementName(unit)],['Attack style',attackStyle(unit)],['Health',numeric(unit.max_hp)],['Damage',numeric(unit.damage)],['Armor reduction',numeric(unit.armor*100)+'%'],['Speed',numeric(unit.speed)],['Projectile',unit.kind==='archer'||unit.kind==='boss'?unit.shot||'arrow':'None']];
      values.forEach(([label,value])=>{const row=document.createElement('div'),dt=document.createElement('dt'),dd=document.createElement('dd');dt.textContent=label;dd.textContent=value;row.append(dt,dd);stats.append(row);});
      document.getElementById('rift-monster-tip').textContent=unit.kind==='boss'?'Watch the marked slam area and jump or move clear. This boss alternates slams with aimed projectiles. Use the recovery after a slam to strike.':unit.kind==='archer'?'Change lanes to evade aimed shots, then close the gap during recovery.':unit.kind==='treasure'?'Flees when you approach. Drive it toward the arena edge to stop its retreat.':'Move out of its lane during the windup, then strike during recovery. A third basic strike can knock it down.';
      animate();title.focus();
    }
    document.getElementById('rift-monster-count').textContent=roster.length+' monsters';
    list.replaceChildren();
    roster.forEach(unit=>{
      const item=document.createElement('article');item.setAttribute('role','listitem');item.dataset.search=normalize(unit.name+' '+unit.tier+' '+elementName(unit)+' '+attackStyle(unit));item.dataset.artKey=unit.art_key;item.dataset.tier=unit.tier;item.dataset.element=elementName(unit);item.dataset.style=attackStyle(unit);
      const sprite=document.createElement('span'),pose=frame(unit,'idle',0);sprite.className='rift-monster-art';sprite.setAttribute('aria-hidden','true');sprite.style.backgroundImage='url("'+assetURL(pose.asset)+'")';sprite.style.backgroundPosition=pose.position;sprite.style.backgroundSize=pose.size;
      const body=document.createElement('div'),name=document.createElement('strong'),detail=document.createElement('small');name.textContent=unit.name;detail.textContent=unit.tier+' · '+(unit.kind==='boss'?'Area attacks':unit.kind==='archer'?'Ranged':unit.kind==='treasure'?'Fleeing':'Melee');
      const button=document.createElement('button');button.type='button';button.textContent='Inspect';button.setAttribute('aria-label','Inspect '+unit.name);button.onclick=()=>inspect(unit,button);
      body.append(name,detail,button);item.append(sprite,body);list.append(item);
    });
    function filter(){let visible=0;for(const item of list.children){item.hidden=!item.dataset.search.includes(normalize(search.value))||(tier.value&&item.dataset.tier!==tier.value)||(element.value&&item.dataset.element!==element.value)||(style.value&&item.dataset.style!==style.value);if(!item.hidden)visible++;}document.getElementById('rift-monsters-empty').hidden=visible>0;document.getElementById('rift-monster-matches').textContent=visible+' of '+roster.length+' monsters';}
    search.oninput=filter;[tier,element,style].forEach(select=>select.onchange=filter);
    document.getElementById('rift-monster-clear').onclick=()=>{search.value='';tier.value='';element.value='';style.value='';filter();search.focus();};filter();
  }
  window.RiftBestiary={profile,frame,render,assetURL,assets:art.atlasAssets};
})();
