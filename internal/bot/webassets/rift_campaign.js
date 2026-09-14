(function(){
  'use strict';
  const $=id=>document.getElementById(id);
  const key='riftCampaignView';
  const defaults={search:'',region:'all',completion:'all',difficulty:'all',favoritesOnly:false,compact:false,selected:1,favorites:[]};
  let view={...defaults},levels=[],completed=new Set(),active=false,selected=1,initialized=false;
  try{const saved=JSON.parse(localStorage.getItem(key));if(saved&&typeof saved==='object')view={...view,...saved};}catch(_){}
  view.search=typeof view.search==='string'?view.search.slice(0,80):'';
  view.favorites=Array.isArray(view.favorites)?view.favorites.filter(n=>Number.isInteger(n)&&n>=1&&n<=100).slice(0,100):[];
  view.favoritesOnly=view.favoritesOnly===true;view.compact=view.compact===true;
  if(!['all','complete','unfinished'].includes(view.completion))view.completion='all';
  if(!['all','Wayfarer','Veteran','Champion','Mythic'].includes(view.difficulty))view.difficulty='all';
  if(!['all',...Array.from({length:10},(_,i)=>String(i))].includes(view.region))view.region='all';
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
    });
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
    $('rift-region-progress').textContent=Array.from({length:10},(_,region)=>{
      const regional=levels.filter(l=>l.region===region);return regional[0].region_name+': '+regional.filter(l=>completed.has(l.id)).length+'/'+regional.length;
    }).join(' · ');
  }
  function reflect(){
    $('rift-mission-search').value=view.search;$('rift-region').value=view.region;
    $('rift-completion').value=view.completion;$('rift-difficulty').value=view.difficulty;
    $('rift-favorites-only').checked=view.favoritesOnly;$('rift-compact').checked=view.compact;
  }
  function reset(){view={...view,search:'',region:'all',completion:'all',difficulty:'all',favoritesOnly:false};reflect();save();apply();}
  function init(catalog){
    levels=catalog;if(!levels.length)return;
    reflect();
    if(!initialized){
      initialized=true;
      const grid=$('rift-levels'),hint=document.createElement('p');
      hint.id='rift-grid-navigation';hint.className='rift-muted';
      hint.textContent='Mission cards: use arrow keys to browse, Home or End for the first or last result, and Enter to select.';
      grid.before(hint);grid.setAttribute('aria-describedby',hint.id);
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
    update(run,id){if(!levels.length)return;selected=id;active=!!run&&['fighting','cleared'].includes(run.status);completed=new Set(run?.completed_levels||[]);view.selected=id;save();apply();}};
})();
