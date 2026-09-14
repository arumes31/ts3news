(function(){
  'use strict';
  const $=id=>document.getElementById(id),key='riftLoadoutPresets';
  let presets=[],skills=[],blocked=()=>true;
  try{const saved=JSON.parse(localStorage.getItem(key));if(Array.isArray(saved))presets=saved.filter(p=>p&&typeof p.id==='string'&&typeof p.name==='string'&&Array.isArray(p.skills)&&p.skills.length<=3&&p.skills.every(id=>typeof id==='string'&&id.length<=120)).slice(0,10).map(p=>({...p,name:p.name.slice(0,40)}));}catch(_){}
  const section=document.createElement('div');section.id='rift-loadout-presets';section.hidden=true;section.setAttribute('role','group');section.setAttribute('aria-label','Skill presets');
  section.innerHTML='<h4>Skill presets</h4><label>Saved preset <select id="rift-preset-list"><option value="">Choose a preset</option></select></label><label>Preset name <input id="rift-loadout-name" maxlength="40" placeholder="e.g. Guardian hunter"></label><div><button type="button" id="rift-save-loadout">Save current skills</button><button type="button" id="rift-rename-loadout">Rename</button><button type="button" id="rift-delete-loadout">Delete preset</button></div><p id="rift-loadout-review"></p><button type="button" id="rift-apply-loadout">Apply reviewed skills</button><p id="rift-loadout-status" role="status"></p>';
  $('rift-loadout').after(section);
  const slots=()=>Array.from($('rift-loadout').querySelectorAll('select'));
  const chosen=()=>presets.find(p=>p.id===$('rift-preset-list').value);
  const name=id=>id?(skills.find(skill=>skill.id===id)?.name||'Unavailable skill: '+id):'None';
  const invalid=p=>p.skills.some(id=>id&&!skills.some(skill=>skill.id===id))||new Set(p.skills.filter(Boolean)).size!==p.skills.filter(Boolean).length||p.skills.slice(slots().length).some(Boolean);
  function write(next){try{localStorage.setItem(key,JSON.stringify(next));presets=next;return true;}catch(_){$('rift-loadout-status').textContent='Could not save presets in this browser.';return false;}}
  function options(id=''){$('rift-preset-list').replaceChildren(new Option('Choose a preset',''),...presets.map(p=>new Option(p.name,p.id)));$('rift-preset-list').value=id;refresh();}
  function refresh(){
    const p=chosen(),locked=blocked();
    $('rift-save-loadout').disabled=locked||presets.length>=10;
    $('rift-rename-loadout').disabled=locked||!p;$('rift-delete-loadout').disabled=locked||!p;
    $('rift-apply-loadout').disabled=locked||!p||invalid(p);
    $('rift-loadout-review').textContent=p?slots().map((slot,i)=>'Slot '+(i+1)+': '+name(slot.value)+' → '+name(p.skills[i]||'')).join(' · ')+(invalid(p)?' Cannot apply: unavailable or duplicate skills.':' Applying changes only these expedition skill slots.'):'Save up to 10 presets in this browser. Empty slots keep basic attacks and class abilities available.';
  }
  $('rift-preset-list').addEventListener('change',()=>{$('rift-loadout-name').value=chosen()?.name||'';refresh();});
  $('rift-loadout').addEventListener('change',refresh);
  $('rift-save-loadout').onclick=()=>{if(blocked()||presets.length>=10)return;const title=$('rift-loadout-name').value.trim();if(!title){$('rift-loadout-status').textContent='Enter a preset name.';return;}const p={id:crypto.randomUUID(),name:title,skills:slots().map(slot=>slot.value)};if(invalid(p)){ $('rift-loadout-status').textContent='Choose each owned skill only once.';return;}if(write([...presets,p])){options(p.id);$('rift-loadout-status').textContent='Preset saved locally.';}};
  $('rift-rename-loadout').onclick=()=>{const p=chosen(),title=$('rift-loadout-name').value.trim();if(blocked()||!p)return;if(!title){$('rift-loadout-status').textContent='Enter a preset name.';return;}if(write(presets.map(item=>item.id===p.id?{...item,name:title}:item))){options(p.id);$('rift-loadout-status').textContent='Preset renamed.';}};
  $('rift-delete-loadout').onclick=()=>{const p=chosen();if(blocked()||!p)return;if(write(presets.filter(item=>item.id!==p.id))){options();$('rift-loadout-status').textContent='Preset deleted. Current skills and inventory are unchanged.';}};
  $('rift-apply-loadout').onclick=()=>{const p=chosen();if(blocked()||!p||invalid(p))return;slots().forEach((slot,i)=>{slot.value=p.skills[i]||'';});refresh();$('rift-loadout-status').textContent='Preset applied to the next expedition.';};
  window.RiftLoadouts={init(build,isBlocked){skills=build.skills;blocked=isBlocked;section.hidden=false;options();},refresh};
})();
