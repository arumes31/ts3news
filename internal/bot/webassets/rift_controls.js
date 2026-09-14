(function(){
  'use strict';
  const $=id=>document.getElementById(id);
  const definitions=[['up','Move up',['KeyW','ArrowUp']],['left','Move left',['KeyA','ArrowLeft']],['down','Move down',['KeyS','ArrowDown']],['right','Move right',['KeyD','ArrowRight']],['attack','Attack',['KeyJ']],['jump','Jump',['Space','KeyK']],['guard','Guard',['KeyL']],['signature0','Class builder',['KeyQ']],['signature1','Class finisher',['KeyE']],['skill0','Equipped skill 1',['Digit1']],['skill1','Equipped skill 2',['Digit2']],['skill2','Equipped skill 3',['Digit3']],['ultimate','Ultimate',['KeyR']],['pause','Pause / resume',['Escape']]];
  const defaults=Object.fromEntries(definitions.map(([id,,keys])=>[id,keys]));
  const allowed=code=>/^(Key[A-Z]|Digit[0-9]|Numpad[0-9]|Arrow(Up|Down|Left|Right)|Space|Enter|Backspace|Delete|Home|End|PageUp|PageDown|Insert|Comma|Period|Slash|Semicolon|Quote|BracketLeft|BracketRight|Backslash|Minus|Equal|Backquote)$/.test(code);
  const keyName=code=>code.replace(/^Key/,'').replace(/^Digit/,'');
  function sanitize(saved){
    const next={};
    for(const [id,,fallback] of definitions){const codes=saved?.[id];next[id]=Array.isArray(codes)&&codes.length>0&&codes.length<=2&&new Set(codes).size===codes.length&&codes.every(code=>typeof code==='string'&&(allowed(code)||id==='pause'&&code==='Escape'))?[...codes]:[...fallback];}
    // Restore colliding actions together; a fallback may free or collide with
    // another saved key, so converge on the conflict-free default map.
    for(let pass=0;pass<=definitions.length;pass++){
      const owners=new Map(),conflicts=new Set();
      for(const [id,codes] of Object.entries(next))for(const code of codes){if(owners.has(code)){conflicts.add(id);conflicts.add(owners.get(code));}else owners.set(code,id);}
      if(!conflicts.size)return next;
      for(const id of conflicts)next[id]=[...defaults[id]];
    }
    return Object.fromEntries(definitions.map(([id,,keys])=>[id,[...keys]]));
  }
  let bindings=sanitize(null),capture=null,toggleGuard=false;
  const pointer={attack:-1,guard:-1};
  try{
    const saved=JSON.parse(localStorage.getItem('riftBindings'));
    if(saved?.version===1){
      bindings=sanitize(saved.bindings);toggleGuard=saved.toggleGuard===true;
      for(const id of ['attack','guard'])if([-1,0,1,2].includes(saved.pointer?.[id]))pointer[id]=saved.pointer[id];
      if(pointer.attack!==-1&&pointer.attack===pointer.guard)pointer.attack=pointer.guard=-1;
    }
  }catch(_){}
  const dialog=$('rift-controls-dialog'),list=$('rift-key-bindings'),buttons=new Map();
  const label=id=>(!dialog.open&&window.RiftGamepad?.active&&window.RiftGamepad.label(id))||(bindings[id]||[]).map(keyName).join(' / ');
  const status=message=>$('rift-binding-status').textContent=message;
  function prompts(){
    document.querySelectorAll('[data-bind]').forEach(node=>{const kbd=node.querySelector('kbd'),text=label(node.dataset.bind);if(kbd&&kbd.textContent!==text)kbd.textContent=text;});
    const movement=$('rift-movement-keys'),text=window.RiftGamepad?.active?'Stick / D-pad':['up','left','down','right'].map(id=>keyName(bindings[id][0])).join(' ');if(movement&&movement.textContent!==text)movement.textContent=text;
    const descriptionText='Battlefield. '+description();if($('rift-canvas').getAttribute('aria-label')!==descriptionText)$('rift-canvas').setAttribute('aria-label',descriptionText);
  }
  function refresh(){for(const [id,button] of buttons){button.textContent=capture===id?'Press a key…':label(id);button.setAttribute('aria-pressed',String(capture===id));}prompts();}
  function save(){try{localStorage.setItem('riftBindings',JSON.stringify({version:1,bindings,pointer,toggleGuard}));}catch(_){}refresh();window.dispatchEvent(new Event('riftbindingschange'));}
  for(const id of ['attack','guard']){
    const select=$('rift-mouse-'+id);select.value=pointer[id];
    select.onchange=()=>{
      const button=Number(select.value),other=id==='attack'?'guard':'attack';
      if(button!==-1&&pointer[other]===button){select.value=pointer[id];status('That mouse button is already assigned to '+other+'.');return;}
      pointer[id]=button;save();status('Mouse '+id+' updated.');
    };
  }
  $('rift-toggle-guard').checked=toggleGuard;$('rift-toggle-guard').onchange=()=>{toggleGuard=$('rift-toggle-guard').checked;save();status(toggleGuard?'Guard toggles on each press. Pausing clears it.':'Hold the guard control to defend.');};
  for(const [id,name] of definitions){
    const row=document.createElement('div'),title=document.createElement('span'),button=document.createElement('button');title.textContent=name;button.type='button';button.dataset.remap=id;button.setAttribute('aria-label','Rebind '+name);button.setAttribute('aria-describedby','rift-binding-status');
    button.onclick=()=>{capture=id;refresh();status('Press a physical key for '+name+'. Escape cancels.');};row.append(title,button);list.append(row);buttons.set(id,button);
  }
  function cancelCapture(){capture=null;refresh();status('Binding unchanged.');}
  dialog.addEventListener('keydown',event=>{
    event.stopPropagation();
    if(!capture&&event.code==='Tab'){
      const focusable=[...dialog.querySelectorAll('button:not(:disabled),select,input')],first=focusable[0],last=focusable[focusable.length-1];
      if(event.shiftKey&&document.activeElement===first){event.preventDefault();last.focus();}
      else if(!event.shiftKey&&document.activeElement===last){event.preventDefault();first.focus();}
    }
    if(!capture)return;
    if(event.isComposing||event.repeat){event.preventDefault();return;}
    if(event.code==='Escape'){event.preventDefault();cancelCapture();return;}
    event.preventDefault();
    if(event.ctrlKey||event.altKey||event.metaKey||event.shiftKey||!allowed(event.code)){status('Choose an unmodified letter, number, arrow or navigation key. Escape cancels.');return;}
    const conflict=definitions.find(([id])=>id!==capture&&bindings[id].includes(event.code));
    if(conflict){status(keyName(event.code)+' is assigned to '+conflict[1]+'. Choose another key.');return;}
    const id=capture;bindings[id]=[event.code];capture=null;save();status(definitions.find(([name])=>name===id)[1]+' uses physical '+keyName(event.code)+'.');
  });
  dialog.addEventListener('cancel',event=>{if(capture){event.preventDefault();cancelCapture();}});
  dialog.addEventListener('close',()=>{capture=null;refresh();$('rift-controls-open').focus();});
  $('rift-controls-close').onclick=()=>dialog.close();
  const presets={
    standard:{keys:defaults,description:'WASD or arrows move. J attacks, Space jumps, L guards, Q/E use class abilities, 1–3 use skills and R uses the ultimate.'},
    leftHanded:{keys:{up:['ArrowUp'],left:['ArrowLeft'],down:['ArrowDown'],right:['ArrowRight'],attack:['KeyA'],jump:['Space'],guard:['KeyS'],signature0:['KeyQ'],signature1:['KeyW'],skill0:['Digit1'],skill1:['Digit2'],skill2:['Digit3'],ultimate:['KeyE'],pause:['Escape']},description:'Arrows move with the right hand. A attacks, Space jumps, S guards, Q/W use class abilities, 1–3 use skills and E uses the ultimate.'},
    oneHanded:{keys:{up:['KeyW'],left:['KeyA'],down:['KeyS'],right:['KeyD'],attack:['KeyF'],jump:['Space'],guard:['KeyC'],signature0:['KeyQ'],signature1:['KeyE'],skill0:['Digit1'],skill1:['Digit2'],skill2:['Digit3'],ultimate:['KeyR'],pause:['Escape']},description:'Compact left-hand layout: WASD moves, F attacks, Space jumps, C guards, Q/E use class abilities, 1–3 use skills and R uses the ultimate.'}
  };
  const preset=$('rift-key-preset');preset.onchange=()=>{$('rift-key-preset-description').textContent=presets[preset.value].description;};preset.onchange();
  $('rift-apply-keys').onclick=()=>{capture=null;bindings=sanitize(presets[preset.value].keys);save();status('Layout applied. The expedition remains paused.');};
  $('rift-reset-keys').onclick=()=>{capture=null;bindings=sanitize(null);preset.value='standard';preset.onchange();save();status('Default keys restored.');};
  function description(){return (window.RiftGamepad?.active?'Stick / D-pad':['up','left','down','right'].map(id=>keyName(bindings[id][0])).join('/'))+' moves · '+label('jump')+' jumps · '+label('attack')+' attacks · '+label('guard')+' guards · '+label('signature0')+'/'+label('signature1')+' class abilities · '+['skill0','skill1','skill2'].map(id=>label(id)).join('/')+' skills · '+label('ultimate')+' ultimate · '+label('pause')+' pauses. Escape always pauses.';}
  window.RiftControls={codes:id=>bindings[id]||[],action:code=>definitions.find(([id])=>bindings[id].includes(code))?.[0],pointer:button=>['attack','guard'].find(id=>pointer[id]===button),get toggleGuard(){return toggleGuard;},label,description,prompts,get opened(){return dialog.open;},open(){capture=null;refresh();status('Choose an action to rebind. Changes are saved on this device.');dialog.showModal();}};
  refresh();
})();
