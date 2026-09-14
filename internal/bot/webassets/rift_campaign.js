(function(){
  'use strict';
  const $=id=>document.getElementById(id);
  const key='riftCampaignView';
  const defaults={search:'',region:'all',completion:'all',difficulty:'all',favoritesOnly:false,compact:false,selected:1,favorites:[],scrollTop:0,sort:'mission'};
  let view={...defaults},levels=[],completed=new Set(),history={},active=false,selected=1,initialized=false,overview=false,expandedRegion=null,lastAttempt=null;
  try{const saved=JSON.parse(localStorage.getItem(key));if(saved&&typeof saved==='object')view={...view,...saved};}catch(_){}
  view.search=typeof view.search==='string'?view.search.slice(0,80):'';
  view.favorites=Array.isArray(view.favorites)?view.favorites.filter(n=>Number.isInteger(n)&&n>=1&&n<=100).slice(0,100):[];
  view.favoritesOnly=view.favoritesOnly===true;view.compact=view.compact===true;
  if(typeof view.scrollTop!=='number'||!Number.isFinite(view.scrollTop)||view.scrollTop<0)view.scrollTop=0;
  if(!['all','complete','unfinished'].includes(view.completion))view.completion='all';
  if(!['all','Wayfarer','Veteran','Champion','Mythic'].includes(view.difficulty))view.difficulty='all';
  if(!['all',...Array.from({length:10},(_,i)=>String(i))].includes(view.region))view.region='all';
  if(!['mission','best','recent'].includes(view.sort))view.sort='mission';
  const outcomeNames={active:'In progress',completed:'Finished',defeated:'Defeated',exited:'Left early',expired:'Expired'};
  const record=id=>history[id];
  const best=id=>Number.isFinite(record(id)?.best_seconds)&&record(id).best_seconds>0?record(id).best_seconds:Infinity;
  const recent=id=>Number.isFinite(record(id)?.last_started_ms)?record(id).last_started_ms:0;
  const normalize=value=>String(value).normalize('NFD').replace(/[\u0300-\u036f]/g,'').toLowerCase();
  function save(){try{localStorage.setItem(key,JSON.stringify(view));}catch(_){} }
  function choose(id){if(!active)$('rift-levels').querySelector('[data-level="'+id+'"]').click();}
  function apply(){
    const query=normalize(view.search.trim()),favorites=new Set(view.favorites);let count=0;
    $('rift-levels').querySelectorAll('[data-level]').forEach(button=>{
      const id=Number(button.dataset.level),level=levels.find(l=>l.id===id);
      const matches=normalize(id+' '+String(id).padStart(3,'0')+' '+level.name+' '+level.tactic).includes(query)
        &&(view.region==='all'||String(level.region)===view.region)
        &&(view.difficulty==='all'||level.difficulty===view.difficulty)
        &&(view.completion==='all'||completed.has(id)===(view.completion==='complete'))
        &&(!view.favoritesOnly||favorites.has(id));
      button.hidden=!matches;button.classList.toggle('favorite',favorites.has(id));if(matches)count++;
      const entry=record(id),note=button.querySelector('.rift-mission-history');
      note.hidden=!entry&&!completed.has(id);
      note.textContent=entry?entry.attempts+' recorded '+(entry.attempts===1?'attempt':'attempts')+' · '+(outcomeNames[entry.last_outcome]||'Outcome unavailable')+(Number.isFinite(best(id))?' · Best '+best(id).toFixed(1)+'s':''): 'Completed before attempt tracking';
    });
    const grid=$('rift-levels'),cards=Array.from(grid.children),ordered=[...cards].sort((a,b)=>{
      const left=Number(a.dataset.level),right=Number(b.dataset.level);
      return (view.sort==='best'?best(left)-best(right):view.sort==='recent'?recent(right)-recent(left):0)||left-right;
    });
    if(ordered.some((card,index)=>card!==cards[index])){
      const focused=document.activeElement,scroll=grid.scrollTop;
      grid.append(...ordered);if(grid.contains(focused))focused.focus({preventScroll:true});grid.scrollTop=scroll;
    }
    lastAttempt=levels.filter(level=>recent(level.id)>0).sort((a,b)=>recent(b.id)-recent(a.id)||a.id-b.id)[0]||null;
    $('rift-last-attempt').hidden=!lastAttempt;
    $('rift-last-attempt').textContent=lastAttempt?'Last attempted: Mission '+lastAttempt.id+' · '+lastAttempt.name:'';
    $('rift-last-attempt').disabled=active;
    $('rift-levels').classList.toggle('compact',view.compact);
    $('rift-filter-count').textContent=count+' of '+levels.length+' missions';
    $('rift-filter-empty').hidden=count>0;
    $('rift-favorite').setAttribute('aria-pressed',String(favorites.has(selected)));
    $('rift-favorite').textContent=favorites.has(selected)?'★ Saved favorite':'☆ Favorite mission';
    const selectedLevel=levels.find(level=>level.id===selected);
    $('rift-favorite').setAttribute('aria-label',(favorites.has(selected)?'Remove favorite':'Favorite mission')+' '+selected+': '+selectedLevel.name);
    $('rift-previous-mission').disabled=active||selected<=1;
    $('rift-next-mission').disabled=active||selected>=levels.length;
    $('rift-unfinished').disabled=active||levels.every(l=>completed.has(l.id));
    const selectedButton=$('rift-levels').querySelector('[data-level="'+selected+'"]');
    $('rift-show-selected').hidden=!selectedButton?.hidden;
    $('rift-selected-mission').hidden=!selectedButton?.hidden&&!overview;
    $('rift-selected-mission').textContent=(active?'Active expedition: ':'Selected mission: ')+selected+' · '+selectedLevel.name+' — '+selectedLevel.difficulty+'. '+selectedLevel.tactic;
    $('rift-region-overview').hidden=!overview;
    $('rift-levels').hidden=overview;$('rift-grid-navigation').hidden=overview;
    $('rift-overview-toggle').textContent=overview?'Show mission cards':'Region overview';
    $('rift-overview-toggle').setAttribute('aria-expanded',String(overview));
    $('rift-region-overview').querySelectorAll('[data-overview-region]').forEach(section=>{
      const region=Number(section.dataset.overviewRegion),regional=levels.filter(level=>level.region===region);
      section.querySelector('summary').textContent=regional[0].region_name+' · '+regional.filter(level=>completed.has(level.id)).length+'/'+regional.length+' completed';
      section.querySelector('p').textContent='Missions '+regional[0].id+'–'+regional[regional.length-1].id+'. '+(selectedLevel.region===region?(active?'Active expedition':'Selected mission')+': '+selected+'.':'');
      if(expandedRegion!==selectedLevel.region)section.querySelector('details').open=selectedLevel.region===region;
    });
    expandedRegion=selectedLevel.region;
    $('rift-region-progress').textContent=Array.from({length:10},(_,region)=>{
      const regional=levels.filter(l=>l.region===region);return regional[0].region_name+': '+regional.filter(l=>completed.has(l.id)).length+'/'+regional.length;
    }).join(' · ');
  }
  function reflect(){
    $('rift-mission-search').value=view.search;$('rift-region').value=view.region;
    $('rift-completion').value=view.completion;$('rift-difficulty').value=view.difficulty;
    $('rift-favorites-only').checked=view.favoritesOnly;$('rift-compact').checked=view.compact;
    if($('rift-mission-sort'))$('rift-mission-sort').value=view.sort;
  }
  function reset(){view={...view,search:'',region:'all',completion:'all',difficulty:'all',favoritesOnly:false};reflect();save();apply();}
  function init(catalog){
    levels=catalog;if(!levels.length)return;
    reflect();
    if(!initialized){
      initialized=true;
      const grid=$('rift-levels'),hint=document.createElement('p');
      grid.querySelectorAll('[data-level]').forEach(button=>{const note=document.createElement('span');note.className='rift-mission-history';note.id='rift-history-'+button.dataset.level;button.append(note);button.setAttribute('aria-describedby',note.id);});
      const sortLabel=document.createElement('label'),sort=document.createElement('select'),last=document.createElement('button');
      sortLabel.textContent='Sort missions ';sort.id='rift-mission-sort';
      for(const [value,name] of [['mission','Mission number'],['best','Best clear time'],['recent','Recently played']]){const option=document.createElement('option');option.value=value;option.textContent=name;sort.append(option);}
      sort.value=view.sort;sortLabel.append(sort);last.id='rift-last-attempt';last.type='button';last.hidden=true;grid.before(sortLabel,last);
      sort.addEventListener('change',()=>{view.sort=sort.value;save();apply();});
      last.addEventListener('click',()=>{if(lastAttempt&&!active)choose(lastAttempt.id);});
      const campaign=$('rift-campaign');
      const rememberScroll=()=>{if(campaign.open&&!grid.hidden)view.scrollTop=grid.scrollTop;};
      const restoreScroll=()=>requestAnimationFrame(()=>{if(campaign.open&&!grid.hidden)grid.scrollTop=view.scrollTop;});
      grid.addEventListener('scroll',rememberScroll,{passive:true});
      campaign.addEventListener('toggle',()=>{if(campaign.open)restoreScroll();else save();});
      window.addEventListener('pagehide',()=>{rememberScroll();save();});
      restoreScroll();
      hint.id='rift-grid-navigation';hint.className='rift-muted';
      hint.textContent='Mission cards: use arrow keys to browse, Home or End for the first or last result, and Enter to select.';
      grid.before(hint);grid.setAttribute('aria-describedby',hint.id);
      const toggle=document.createElement('button'),regions=document.createElement('nav'),pinned=document.createElement('p');
      toggle.id='rift-overview-toggle';toggle.type='button';toggle.setAttribute('aria-controls','rift-region-overview');
      regions.id='rift-region-overview';regions.setAttribute('aria-label','Campaign regions');regions.hidden=true;
      pinned.id='rift-selected-mission';pinned.hidden=true;pinned.setAttribute('role','status');
      hint.before(toggle,pinned,regions);
      for(const region of [...new Set(levels.map(level=>level.region))]){
        const regional=levels.filter(level=>level.region===region),section=document.createElement('section'),details=document.createElement('details'),summary=document.createElement('summary'),description=document.createElement('p'),browse=document.createElement('button');
        section.dataset.overviewRegion=region;section.setAttribute('aria-label',regional[0].region_name);
        browse.type='button';browse.textContent='Browse '+regional[0].region_name;
        browse.addEventListener('click',()=>{view.region=String(region);overview=false;reflect();save();apply();$('rift-region').focus();});
        details.append(summary,description,browse);section.append(details);regions.append(section);
      }
      toggle.addEventListener('click',()=>{rememberScroll();overview=!overview;if(overview)expandedRegion=null;apply();if(!overview)restoreScroll();});
      grid.addEventListener('keydown',event=>{
        if(active||event.altKey||event.ctrlKey||event.metaKey||event.shiftKey)return;
        const cards=Array.from(grid.querySelectorAll('[data-level]')).filter(button=>!button.hidden&&!button.disabled);
        const index=cards.indexOf(event.target);if(index<0)return;
        let target;
        if(event.key==='Home')target=cards[0];
        else if(event.key==='End')target=cards[cards.length-1];
        else if(event.key==='ArrowLeft')target=cards[Math.max(0,index-1)];
        else if(event.key==='ArrowRight')target=cards[Math.min(cards.length-1,index+1)];
        else if(event.key==='ArrowUp'||event.key==='ArrowDown'){
          const origin=event.target.getBoundingClientRect(),direction=event.key==='ArrowDown'?1:-1;
          const rows=cards.map(button=>({button,rect:button.getBoundingClientRect()})).filter(({rect})=>(rect.top-origin.top)*direction>1);
          rows.sort((a,b)=>Math.abs(a.rect.top-origin.top)-Math.abs(b.rect.top-origin.top)||Math.abs(a.rect.left-origin.left)-Math.abs(b.rect.left-origin.left));
          target=rows[0]?.button||event.target;
        }else return;
        event.preventDefault();target?.focus();
      });
      for(const [id,field,event] of [['rift-mission-search','search','input'],['rift-region','region','change'],['rift-completion','completion','change'],['rift-difficulty','difficulty','change'],['rift-favorites-only','favoritesOnly','change'],['rift-compact','compact','change']]){
        $(id).addEventListener(event,()=>{view[field]=$(id).type==='checkbox'?$(id).checked:$(id).value;save();apply();});
      }
      $('rift-clear-filters').addEventListener('click',reset);
      $('rift-favorite').addEventListener('click',()=>{view.favorites=view.favorites.includes(selected)?view.favorites.filter(id=>id!==selected):[...view.favorites,selected];save();apply();});
      $('rift-unfinished').addEventListener('click',()=>{const target=levels.find(l=>l.id>selected&&!completed.has(l.id))||levels.find(l=>!completed.has(l.id));if(target)choose(target.id);});
      $('rift-previous-mission').addEventListener('click',()=>choose(selected-1));
      $('rift-next-mission').addEventListener('click',()=>choose(selected+1));
      $('rift-show-selected').addEventListener('click',()=>{reset();$('rift-levels').querySelector('[data-level="'+selected+'"]').scrollIntoView({block:'nearest'});});
    }
  }
  window.RiftCampaignTools={init,preferred:()=>Number.isInteger(view.selected)&&view.selected>=1&&view.selected<=100?view.selected:1,
    showRegion(region){if(!levels.some(level=>level.region===region))return;overview=true;apply();$('rift-campaign').open=true;const section=$('rift-region-overview').querySelector('[data-overview-region="'+region+'"]');section.querySelector('details').open=true;section.querySelector('summary').focus();section.scrollIntoView({block:'center'});},
    update(run,id){if(!levels.length)return;selected=id;history=run?.mission_history||{};active=!!run&&['fighting','cleared'].includes(run.status);completed=new Set(run?.completed_levels||[]);view.selected=id;save();apply();}};
})();
