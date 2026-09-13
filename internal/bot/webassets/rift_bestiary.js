(function(){
  'use strict';
  const art=window.AbyssCombatArt;
  const assetURL=path=>path+(window.__ASSET_VER__?'?v='+encodeURIComponent(window.__ASSET_VER__):'');
  function profile(unit){return art.actorProfile({...unit,role:unit.kind==='boss'?'boss':unit.kind});}
  function frame(unit,pose,index){return art.actorFrame({...unit,role:unit.kind==='boss'?'boss':unit.kind},pose,index);}
  function render(roster){
    const list=document.getElementById('rift-monsters'),search=document.getElementById('rift-monster-search');
    document.getElementById('rift-monster-count').textContent=roster.length+' monsters';
    list.replaceChildren();
    roster.forEach(unit=>{
      const item=document.createElement('article');item.setAttribute('role','listitem');item.dataset.search=(unit.name+' '+unit.tier).toLowerCase();item.dataset.artKey=unit.art_key;
      const sprite=document.createElement('span'),pose=frame(unit,'idle',0);sprite.className='rift-monster-art';sprite.setAttribute('aria-hidden','true');sprite.style.backgroundImage='url("'+assetURL(pose.asset)+'")';sprite.style.backgroundPosition=pose.position;sprite.style.backgroundSize=pose.size;
      const body=document.createElement('div'),name=document.createElement('strong'),detail=document.createElement('small');name.textContent=unit.name;detail.textContent=unit.tier+' · '+(unit.kind==='boss'?'Area attacks':unit.kind==='archer'?'Ranged':unit.kind==='treasure'?'Fleeing':'Melee');
      body.append(name,detail);item.append(sprite,body);list.append(item);
    });
    search.oninput=()=>{let visible=0;for(const item of list.children){item.hidden=!item.dataset.search.includes(search.value.trim().toLowerCase());if(!item.hidden)visible++;}document.getElementById('rift-monsters-empty').hidden=visible>0;};search.oninput();
  }
  window.RiftBestiary={profile,frame,render,assetURL,assets:art.atlasAssets};
})();
