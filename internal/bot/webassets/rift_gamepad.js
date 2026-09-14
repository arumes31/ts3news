(function(){
  'use strict';
  const $=id=>document.getElementById(id),settings={deadzone:.18,horizontal:1,vertical:1};
  const limits={deadzone:[0,.8,.02],horizontal:[.25,2,.05],vertical:[.25,2,.05]};
  try{const saved=JSON.parse(localStorage.getItem('riftGamepad'));if(saved?.version===1)for(const key of Object.keys(settings)){const value=saved[key],[min,max]=limits[key];if(typeof value==='number'&&Number.isFinite(value)&&value>=min&&value<=max)settings[key]=value;}}catch(_){}
  const panel=document.createElement('details');panel.id='rift-gamepad-panel';
  const summary=document.createElement('summary');summary.textContent='Controller setup & test';panel.append(summary);
  const status=document.createElement('p');status.id='rift-gamepad-status';status.setAttribute('role','status');panel.append(status);
  const hint=document.createElement('p');hint.textContent='Press a controller button to connect. Standard mapping: left stick / D-pad move; South jumps, West attacks, East guards, North casts the selected skill. LB/RB use class abilities; LT/RT select skills; View uses the ultimate; Menu pauses or resumes. While paused, D-pad navigates and South activates.';panel.append(hint);
  const output=document.createElement('output');output.id='rift-gamepad-test';output.setAttribute('aria-label','Controller test readings');output.setAttribute('aria-live','off');panel.append(output);
  for(const [key,title] of [['deadzone','Stick dead zone'],['horizontal','Horizontal stick sensitivity'],['vertical','Vertical stick sensitivity']]){
    const label=document.createElement('label'),input=document.createElement('input'),value=document.createElement('span');input.type='range';input.id='rift-gamepad-'+key;[input.min,input.max,input.step]=limits[key];input.value=settings[key];value.textContent=String(settings[key]);label.append(document.createTextNode(title),input,value);panel.append(label);
    input.oninput=()=>{settings[key]=Number(input.value);value.textContent=input.value;reset();try{localStorage.setItem('riftGamepad',JSON.stringify({version:1,...settings}));}catch(_){}};
  }
  document.querySelector('.rift-settings').append(panel);
  let callbacks=null,identity='',blocked=true,previous=[],held=new Set(),taps=new Set(),x=0,y=0,selected=0,active=false,lastX=0,lastY=0;
  const names={attack:'West',jump:'South',guard:'East',signature0:'LB',signature1:'RB',ultimate:'View',pause:'Menu'};
  const put=(node,text)=>{if(node.textContent!==text)node.textContent=text;};
  function device(value){if(active===value)return;active=value;window.RiftControls?.prompts();}
  function reset(){blocked=true;held.clear();taps.clear();x=y=0;}
  function disconnect(message){const wasConnected=!!identity;identity='';reset();previous=[];device(false);put(status,message);put(output,'No controller input');if(wasConnected)callbacks?.disconnect();}
  function axis(value,sensitivity){if(!Number.isFinite(value))return 0;const magnitude=Math.min(1,Math.abs(value));return magnitude<=settings.deadzone?0:Math.sign(value)*Math.min(1,(magnitude-settings.deadzone)/(1-settings.deadzone)*sensitivity);}
  function navigate(button){
    const elements=[...$('rift-app').querySelectorAll('button:not(:disabled),summary,input:not(:disabled),select:not(:disabled),a[href]')].filter(node=>node.getClientRects().length&&!node.closest('[hidden],dialog:not([open])'));
    if(!elements.length)return;const current=document.activeElement,index=elements.indexOf(current);
    if((button===14||button===15)&&current.matches('select,input[type=range]')){
      const direction=button===14?-1:1;
      if(current.matches('select'))current.selectedIndex=Math.max(0,Math.min(current.options.length-1,current.selectedIndex+direction));
      else current.value=String(Math.max(Number(current.min),Math.min(Number(current.max),Number(current.value)+direction*Number(current.step||1))));
      current.dispatchEvent(new Event('input',{bubbles:true}));current.dispatchEvent(new Event('change',{bubbles:true}));return;
    }
    if(button===0){if(index>=0&&current.matches('button,summary,a,input[type=checkbox]'))current.click();return;}
    const direction=button===12||button===14?-1:1;elements[(index+direction+elements.length)%elements.length].focus();
  }
  function poll(){
    if(!callbacks)return;
    let pads;try{pads=navigator.getGamepads?.()||[];}catch(_){disconnect('Controller access unavailable. Keyboard and touch remain available.');return;}
    const connected=[...pads].filter(p=>p?.connected),pad=connected.find(p=>String(p.index)+':'+p.id===identity)||connected.find(p=>p.mapping==='standard');
    if(!pad){disconnect(connected.length?'This controller has no standard mapping. Use keyboard or touch.':'No controller connected. Press a controller button.');return;}
    if(pad.mapping!=='standard'){disconnect('This controller has no standard mapping. Use keyboard or touch.');return;}
    const next=String(pad.index)+':'+pad.id;
    if(identity&&identity!==next){disconnect('Controller changed. Release controls before resuming.');return;}
    if(!identity){identity=next;reset();previous=[];}
    const buttons=Array.from({length:17},(_,i)=>pad.buttons[i]?.pressed===true||pad.buttons[i]?.value>.5);
    const sx=axis(pad.axes[0],settings.horizontal),sy=axis(pad.axes[1],settings.vertical),neutral=!buttons.some(Boolean)&&sx===0&&sy===0;
    put(status,'Connected: '+pad.id+(blocked?' · Release controls to arm.':''));
    if(panel.open)put(output,'X '+sx.toFixed(2)+' · Y '+sy.toFixed(2)+' · Buttons '+(buttons.map((down,i)=>down?i:null).filter(i=>i!==null).join(', ')||'none'));
    if(blocked){previous=buttons;if(neutral)blocked=false;return;}
    const edges=buttons.map((down,i)=>down&&!previous[i]);previous=buttons;
    if(document.hidden||window.RiftControls.opened){reset();return;}
    if((sx||sy)&&(sx!==lastX||sy!==lastY)||edges.some(Boolean))device(true);lastX=sx;lastY=sy;
    if(edges[9]){reset();callbacks.togglePause();return;}
    if(!callbacks.playing()){
      held.clear();taps.clear();x=y=0;for(const i of [12,13,14,15,0])if(edges[i])navigate(i);return;
    }
    const count=callbacks.skillCount();selected=count?Math.min(selected,count-1):0;
    if(count&&(edges[6]||edges[7])){selected=(selected+(edges[7]?1:-1)+count)%count;window.RiftControls.prompts();}
    x=buttons[14]||buttons[15]?Number(buttons[15])-Number(buttons[14]):sx;
    y=buttons[12]||buttons[13]?Number(buttons[13])-Number(buttons[12]):sy;
    held.clear();for(const [i,action] of [[0,'jump'],[1,'guard'],[2,'attack'],[3,'skill'+selected],[4,'signature0'],[5,'signature1'],[8,'ultimate']]){
      if(buttons[i])held.add(action);if(edges[i]){taps.add(action);if(action==='guard')callbacks.guard();}
    }
  }
  window.addEventListener('gamepaddisconnected',event=>{if(identity.startsWith(String(event.gamepad.index)+':'))disconnect('Controller disconnected. Expedition paused.');});
  window.addEventListener('keydown',()=>device(false),true);window.addEventListener('pointerdown',()=>device(false),true);
  window.RiftGamepad={init(value){callbacks=value;function frame(){poll();requestAnimationFrame(frame);}frame();},reset,consume(){const actions=new Set([...held,...taps]);taps.clear();return {x,y,actions};},get active(){return active;},label(action){return names[action]||(action.startsWith('skill')?(Number(action.slice(5))===selected?'North':'LT / RT'):null);}};
})();
