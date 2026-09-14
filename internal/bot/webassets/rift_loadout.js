(function(){
  'use strict';
  const $=id=>document.getElementById(id),key='riftLoadoutPresets';
  let presets=[],skills=[],blocked=()=>true;
  try{const saved=JSON.parse(localStorage.getItem(key));if(Array.isArray(saved))presets=saved.filter(p=>p&&typeof p.id==='string'&&typeof p.name==='string'&&Array.isArray(p.skills)&&p.skills.length<=3&&p.skills.every(id=>typeof id==='string'&&id.length<=120)).slice(0,10).map(p=>({...p,name:p.name.slice(0,40)}));}catch(_){}
  const section=document.createElement('div');section.id='rift-loadout-presets';section.hidden=true;section.setAttribute('role','group');section.setAttribute('aria-label','Skill presets');
  section.innerHTML='<h4>Skill presets</h4><label>Saved preset <select id="rift-preset-list"><option value="">Choose a preset</option></select></label><label>Preset name <input id="rift-loadout-name" maxlength="40" placeholder="e.g. Guardian hunter"></label><div><button type="button" id="rift-save-loadout">Save current skills</button><button type="button" id="rift-rename-loadout">Rename</button><button type="button" id="rift-delete-loadout">Delete preset</button></div><p id="rift-loadout-review"></p><button type="button" id="rift-apply-loadout">Apply reviewed skills</button><p id="rift-loadout-status" role="status"></p>';
  $('rift-loadout').after(section);
  const reorder=document.createElement('div');reorder.id='rift-loadout-order';reorder.setAttribute('role','group');reorder.setAttribute('aria-label','Reorder expedition skills');$('rift-loadout').after(reorder);
  const empty=document.createElement('p');empty.id='rift-empty-loadout';empty.hidden=true;empty.textContent='No optional skills selected. Basic attacks, jumping and guarding remain available. Learned class abilities are separate from these slots.';$('rift-loadout').after(empty);
  const transfer=document.createElement('details');transfer.innerHTML='<summary>Import or export a preset</summary><label>Preset JSON <textarea id="rift-preset-json" maxlength="2000" rows="5" spellcheck="false"></textarea></label><button type="button" id="rift-export-loadout">Export selected preset</button><button type="button" id="rift-import-loadout">Import for review</button><p>Copy the JSON to transfer a named preset. Import saves it locally for review; it does not equip skills.</p>';section.append(transfer);
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
    $('rift-import-loadout').disabled=locked||presets.length>=10;$('rift-export-loadout').disabled=!p;
    empty.hidden=slots().some(slot=>slot.value);
    reorder.querySelectorAll('button').forEach(button=>button.disabled=locked);
    $('rift-loadout-review').textContent=p?slots().map((slot,i)=>'Slot '+(i+1)+': '+name(slot.value)+' → '+name(p.skills[i]||'')).join(' · ')+(invalid(p)?' Cannot apply: unavailable or duplicate skills.':' Applying changes only these expedition skill slots.'):'Save up to 10 presets in this browser. Empty slots keep basic attacks and class abilities available.';
  }
  $('rift-preset-list').addEventListener('change',()=>{$('rift-loadout-name').value=chosen()?.name||'';refresh();});
  $('rift-loadout').addEventListener('change',refresh);
  $('rift-save-loadout').onclick=()=>{if(blocked()||presets.length>=10)return;const title=$('rift-loadout-name').value.trim();if(!title){$('rift-loadout-status').textContent='Enter a preset name.';return;}const p={id:crypto.randomUUID(),name:title,skills:slots().map(slot=>slot.value)};if(invalid(p)){ $('rift-loadout-status').textContent='Choose each owned skill only once.';return;}if(write([...presets,p])){options(p.id);$('rift-loadout-status').textContent='Preset saved locally.';}};
  $('rift-rename-loadout').onclick=()=>{const p=chosen(),title=$('rift-loadout-name').value.trim();if(blocked()||!p)return;if(!title){$('rift-loadout-status').textContent='Enter a preset name.';return;}if(write(presets.map(item=>item.id===p.id?{...item,name:title}:item))){options(p.id);$('rift-loadout-status').textContent='Preset renamed.';}};
  $('rift-delete-loadout').onclick=()=>{const p=chosen();if(blocked()||!p)return;if(write(presets.filter(item=>item.id!==p.id))){options();$('rift-loadout-status').textContent='Preset deleted. Current skills and inventory are unchanged.';}};
  $('rift-apply-loadout').onclick=()=>{const p=chosen();if(blocked()||!p||invalid(p))return;slots().forEach((slot,i)=>{slot.value=p.skills[i]||'';});refresh();$('rift-loadout-status').textContent='Preset applied to the next expedition.';};
  $('rift-export-loadout').onclick=()=>{const p=chosen();if(!p)return;$('rift-preset-json').value=JSON.stringify({version:1,name:p.name,skills:p.skills},null,2);$('rift-preset-json').focus();$('rift-preset-json').select();$('rift-loadout-status').textContent='Preset JSON ready to copy.';};
  $('rift-import-loadout').onclick=()=>{
    if(blocked()||presets.length>=10)return;
    try{
      const source=$('rift-preset-json').value;if(source.length>2000)throw new Error('Preset JSON is too long.');
      const p=JSON.parse(source);
      if(!p||p.version!==1||typeof p.name!=='string'||!p.name.trim()||p.name.length>40||!Array.isArray(p.skills)||p.skills.length>3||!p.skills.every(id=>typeof id==='string'&&id.length<=120))throw new Error('Use a version 1 preset with a name and up to three skill IDs.');
      if(invalid(p))throw new Error('Cannot import: every skill must be owned, fit the available slots and appear only once.');
      const imported={id:crypto.randomUUID(),name:p.name.trim(),skills:p.skills};
      if(write([...presets,imported])){options(imported.id);$('rift-loadout-name').value=imported.name;$('rift-loadout-status').textContent='Preset imported. Review the changes before applying.';}
    }catch(error){$('rift-loadout-status').textContent=error instanceof SyntaxError?'Preset JSON is invalid.':error.message;}
  };
  window.RiftLoadouts={init(build,isBlocked){
    skills=build.skills;blocked=isBlocked;section.hidden=false;reorder.replaceChildren();
    for(let index=0;index<slots().length-1;index++){
      const button=document.createElement('button');button.type='button';button.textContent='Swap slots '+(index+1)+' and '+(index+2);
      button.onclick=()=>{if(blocked())return;const current=slots(),value=current[index].value;current[index].value=current[index+1].value;current[index+1].value=value;refresh();$('rift-loadout-status').textContent='Swapped skill slots '+(index+1)+' and '+(index+2)+'.';};
      reorder.append(button);
    }
    options();
  },refresh};
})();
