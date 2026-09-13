/* Shared shopping arithmetic and local planning. Purchases remain server-owned. */
(function (global) {
  'use strict';
  function integer(value, fallback) {
    if (typeof value === 'string' && !value.trim() || value === null || typeof value === 'boolean' || !['string','number'].includes(typeof value)) return fallback === undefined ? 0 : fallback;
    var n = Number(value); return Number.isSafeInteger(n) && n >= 0 ? n : fallback === undefined ? 0 : fallback;
  }
  function normalize(text) { return String(text == null ? '' : text).normalize('NFD').replace(/[\u0300-\u036f]/g, '').toLocaleLowerCase().replace(/\s+/g, ' ').trim(); }
  function search(text, query) {
    text = normalize(text); var terms = String(query || '').match(/-?"[^"]+"|-?\S+/g) || [];
    return terms.every(function (term) { var excluded = term[0] === '-'; var word = normalize((excluded ? term.slice(1) : term).replace(/^"|"$/g, '')); return !word || (text.includes(word) !== excluded); });
  }
  function budget(balance, reserve, limit) {
    balance = integer(balance); reserve = integer(reserve);
    var available = Math.max(0, balance - reserve), cap = integer(limit, null);
    return {available: cap === null ? available : Math.min(available, cap), reserved: Math.min(balance, reserve)};
  }
  function buffCost(owned, amount) {
    owned = integer(owned, null); amount = integer(amount, null);
    if (owned === null || !amount || !Number.isSafeInteger(owned + amount)) return null;
    var rising = BigInt(Math.min(amount, Math.max(999 - owned, 0))), count = BigInt(amount);
    var total = rising * (2n * BigInt(owned) + rising + 1n) / 2n * 1000000n + (count - rising) * 1000000000n;
    return total <= BigInt(Number.MAX_SAFE_INTEGER) ? Number(total) : null;
  }
  function maxBuff(owned, amount) {
    owned = integer(owned, null); amount = integer(amount);
    if (owned === null) return 0;
    var low = 0, high = Math.min(Math.floor(amount / 1000000), Number.MAX_SAFE_INTEGER - owned);
    while (low < high) { var middle = low + Math.ceil((high - low) / 2), cost = buffCost(owned, middle); if (cost !== null && cost <= amount) low = middle; else high = middle - 1; }
    return low;
  }
  function percentageAmount(balance, percent, step) {
    balance = integer(balance); percent = Math.min(100, integer(percent)); step = integer(step);
    return step > 0 ? Number(BigInt(balance) * BigInt(percent) / 100n / BigInt(step) * BigInt(step)) : 0;
  }
  function plan(lines, wallet, reserves) {
    wallet = wallet || {}; reserves = reserves || {}; var total = {gold: 0n, tokens: 0n}, valid = Array.isArray(lines);
    [wallet,reserves].forEach(function(values){['gold','tokens'].forEach(function(currency){if(values[currency]!==undefined&&integer(values[currency],null)===null)valid=false;});});
    (Array.isArray(lines) ? lines : []).forEach(function (line) {
      var price = integer(line && line.price, null), quantity = integer(line && line.quantity, null), currency = line && line.currency;
      if (price === null || !quantity || !Object.hasOwn(total, currency)) { valid = false; return; }
      total[currency] += BigInt(price) * BigInt(quantity); if (total[currency] > BigInt(Number.MAX_SAFE_INTEGER)) valid = false;
    });
    var gold = Number(total.gold), tokens = Number(total.tokens), availableGold = budget(wallet.gold, reserves.gold).available, availableTokens = budget(wallet.tokens, reserves.tokens).available;
    return {gold: gold, tokens: tokens, goldLeft: Math.max(0, availableGold-gold), tokenLeft: Math.max(0, availableTokens-tokens), goldShortfall: Math.max(0,gold-availableGold), tokenShortfall: Math.max(0,tokens-availableTokens), valid: valid};
  }
  function demand(base, current) { base = integer(base); current = integer(current); return {saving: Math.max(0,base-current), premium: Math.max(0,current-base), percent: base > 0 ? (current-base)/base*100 : 0}; }
  function stats(item) { var values = Object.create(null); (item && (item.stat_details || item.stats) || []).forEach(function (s) { var n=Number(s.value); if (Number.isFinite(n)) values[String(s.code || s.label)] = n; }); return values; }
  function compare(a,b) { var before=stats(a),after=stats(b); return Array.from(new Set(Object.keys(before).concat(Object.keys(after)))).sort().map(function(code){return {code:code,before:before[code]||0,after:after[code]||0,delta:(after[code]||0)-(before[code]||0)};}); }
  var api=global.ShopWorkbench={model:{integer:integer,normalize:normalize,search:search,budget:budget,buffCost:buffCost,maxBuff:maxBuff,percentageAmount:percentageAmount,plan:plan,demand:demand,compare:compare}};
  if (typeof document === 'undefined') return;
  function el(tag,text,cls) { var node=document.createElement(tag); if(text!=null)node.textContent=text; if(cls)node.className=cls; return node; }
  function button(text,fn,cls) { var node=el('button',text,cls||'ghost'); node.type='button'; node.addEventListener('click',fn); return node; }
  function fmt(n) { if(!Number.isFinite(n))return 'Unavailable';if(n&&Math.abs(n)<0.0001)return n.toPrecision(4);return new Intl.NumberFormat(document.documentElement.lang||'en',{maximumFractionDigits:6}).format(n); }
  function read(key,fallback) { try { var value=JSON.parse(localStorage.getItem(key)||'null'); return value==null?fallback:value; } catch(_){return fallback;} }
  function write(key,value) { try{localStorage.setItem(key,JSON.stringify(value));return true;}catch(_){return false;} }
  function control(host,id,label,type,choices) {
    var wrap=el('label'),name=el('span',label),input=el(type==='select'?'select':'input'); input.id=id;
    if(type==='select')(choices||[]).forEach(function(pair){var option=el('option',pair[1]);option.value=pair[0];input.appendChild(option);});
    else {input.type=type||'text';if(type==='number'){input.min='0';input.step='1';input.max=String(Number.MAX_SAFE_INTEGER);input.inputMode='numeric';}}
    if(id)wrap.htmlFor=id;wrap.append(name,input);host.appendChild(wrap);return input;
  }
  function section(host,title,id) {var node=el('details',null,'sw-section');if(id)node.id=id;node.appendChild(el('summary',title));var body=el('div',null,'sw-section-body');node.appendChild(body);host.appendChild(node);return body;}
  function prefs(value) { value=value&&typeof value==='object'?value:{};return {density:value.density==='compact'?'compact':'comfortable',contrast:value.contrast==='high'?'high':'normal',text:value.text==='large'?'large':'normal',annotations:value.annotations!==false}; }
  async function copy(text,status) {try{await navigator.clipboard.writeText(text);status.textContent='Copied.';}catch(_){status.textContent='Copy unavailable. Select and copy the text below.';var area=el('textarea');area.value=text;area.readOnly=true;area.setAttribute('aria-label','Text to copy');status.appendChild(area);area.select();} }
  api.el=el;api.button=button;api.fmt=fmt;api.control=control;api.section=section;api.read=read;api.write=write;api.copy=copy;
  api.create=function(config){
    var prefix=config.prefix,key='shopWorkbench:'+config.user+':'+prefix,raw=read(key,{}),offers=[],byKey=new Map(),comparisons=[],listeners=[];
    raw=raw&&typeof raw==='object'?raw:{};
    function records(values,limit){return (Array.isArray(values)?values:[]).filter(function(x){return x&&typeof x.key==='string'&&x.key.length<=200&&integer(x.price,null)!==null&&['gold','tokens'].includes(x.currency);}).slice(0,limit).map(function(x){return {key:x.key,name:String(x.name||'Offer').slice(0,160),price:integer(x.price),currency:x.currency==='tokens'?'tokens':'gold',quantity:Math.max(1,Math.min(99,integer(x.quantity,1))),target:integer(x.target,null),note:String(x.note||'').slice(0,200)};});}
    var state={watch:records(raw.watch,50),plan:records(raw.plan,20),views:(Array.isArray(raw.views)?raw.views:[]).filter(function(v){return v&&typeof v.name==='string'&&v.name.trim()&&v.filters&&typeof v.filters==='object';}).slice(0,8).map(function(v){return {name:v.name.trim().slice(0,40),filters:v.filters};}),journal:(Array.isArray(raw.journal)?raw.journal:[]).filter(function(v){return v&&typeof v.message==='string'&&Number.isFinite(v.time)&&v.time>=0&&v.time<=8640000000000000;}).slice(0,30).map(function(v){return {message:v.message.slice(0,400),time:v.time};}),prefs:prefs(raw.prefs),gold:integer(raw.gold),tokens:integer(raw.tokens),limit:integer(raw.limit,null)};
    var root=el('details',null,'shop-workbench');root.id=config.id;root.appendChild(el('summary','Shopping workbench'));
    var body=el('div',null,'sw-body');root.appendChild(body);config.host.appendChild(root);
    var live=el('p',null,'sw-status');live.id=prefix+'Announcement';live.setAttribute('role','status');live.setAttribute('aria-live','polite');body.appendChild(live);
    var filters=section(body,'Find offers',prefix+'FilterSection'),filterInputs={};
    function input(id,label,type,choices){var c=control(filters,prefix+id,label,type,choices);filterInputs[id]=c;c.addEventListener('input',changed);return c;}
    var budgetHost=section(body,'Budget & reserves'),reserveGold=control(budgetHost,prefix+'GoldReserve','Gold reserve','number'),reserveTokens=control(budgetHost,prefix+'TokenReserve','Token reserve','number'),limit=control(budgetHost,prefix+'SpendLimit','Gold spending limit (blank = available)','number');
    reserveGold.value=state.gold;reserveTokens.value=state.tokens;limit.value=state.limit===null?'':state.limit;reserveTokens.parentElement.hidden=!config.tokens;
    var budgetSummary=el('p');budgetSummary.id=prefix+'BudgetSummary';budgetHost.appendChild(budgetSummary);
    [reserveGold,reserveTokens,limit].forEach(function(c){c.addEventListener('input',function(){state.gold=integer(reserveGold.value);state.tokens=integer(reserveTokens.value);state.limit=integer(limit.value,null);save();changed();});});
    [25,50].forEach(function(percent){budgetHost.appendChild(button(percent+'% gold budget',function(){limit.value=percentageAmount(config.wallet().gold,percent,1);limit.dispatchEvent(new Event('input',{bubbles:true}));}));});
    budgetHost.appendChild(button('Reset budget',function(){reserveGold.value=reserveTokens.value='0';limit.value='';limit.dispatchEvent(new Event('input',{bubbles:true}));}));
    var watchHost=section(body,'Watchlist'),watchList=el('div');watchList.id=prefix+'Watchlist';watchHost.appendChild(watchList);
    var planHost=section(body,'Shopping plan'),planList=el('div');planList.id=prefix+'PlanList';var planSummary=el('p');planSummary.id=prefix+'PlanSummary';planHost.append(planList,planSummary,el('small','Planning does not buy anything. Review each purchase in the storefront.'));
    planHost.appendChild(button('Clear plan',function(){state.plan=[];save();changed();announce('Shopping plan cleared.');}));
    planHost.appendChild(button('Copy plan',function(){copy(planText(),live);}));
    var compareHost=section(body,'Compare up to three offers'),compareTable=el('div',null,'sw-table-scroll');compareTable.id=prefix+'CompareTable';compareHost.append(compareTable,button('Clear comparison',function(){comparisons=[];refresh();}));
    var viewHost=section(body,'Saved shopping views'),viewName=control(viewHost,prefix+'SearchName','View name','text'),viewSelect=control(viewHost,prefix+'SavedSearches','Saved view','select',[]);viewName.maxLength=40;
    function capture(){var data={};Object.keys(filterInputs).forEach(function(k){var c=filterInputs[k];data[k]=c.type==='checkbox'?c.checked:c.value;});if(config.capture)data._store=config.capture();return data;}
    function viewOptions(){var value=viewSelect.value;viewSelect.replaceChildren();state.views.forEach(function(v){var o=el('option',v.name);o.value=v.name;viewSelect.appendChild(o);});if(state.views.some(function(v){return v.name===value;}))viewSelect.value=value;}
    var saveView=button('Save view',function(){var name=viewName.value.trim().slice(0,40);if(!name){announce('Name this view first.');viewName.focus();return;}var found=state.views.find(function(v){return v.name===name;});if(found)found.filters=capture();else if(state.views.length<8)state.views.push({name:name,filters:capture()});else{announce('Eight views saved. Delete a view or reuse its name.');return;}save();viewOptions();viewSelect.value=name;announce('Saved view '+name+'.');});saveView.id=prefix+'SaveSearch';
    var applyView=button('Apply view',function(){var v=state.views.find(function(v){return v.name===viewSelect.value;});if(!v)return;Object.keys(filterInputs).forEach(function(k){var c=filterInputs[k],value=v.filters[k];if(c.type==='checkbox')c.checked=value===true;else if(typeof value==='string')c.value=value;});if(config.restore&&v.filters._store)config.restore(v.filters._store);changed();});applyView.id=prefix+'ApplySearch';
    viewHost.append(saveView,applyView,button('Delete view',function(){state.views=state.views.filter(function(v){return v.name!==viewSelect.value;});save();viewOptions();announce('Saved view removed.');}));viewOptions();
    var journalHost=section(body,'Local purchase journal'),journalList=el('ol');journalList.id=prefix+'Journal';journalHost.append(journalList,el('small','Confirmed results recorded in this browser; this is not your server transaction history.'),button('Clear journal',function(){state.journal=[];save();renderJournal();}),button('Copy journal',function(){copy(state.journal.map(function(r){return new Date(r.time).toLocaleString()+' · '+r.message;}).join('\n'),live);}));
    var prefsHost=section(body,'Display & storage'),density=control(prefsHost,prefix+'Density','Card density','select',[['comfortable','Comfortable'],['compact','Compact']]),contrast=control(prefsHost,prefix+'Contrast','Contrast','select',[['normal','Normal'],['high','High contrast']]),textSize=control(prefsHost,prefix+'TextSize','Text size','select',[['normal','Normal'],['large','Large']]),annotations=control(prefsHost,prefix+'Annotations','Show offer annotations','checkbox');
    function syncPrefs(){density.value=state.prefs.density;contrast.value=state.prefs.contrast;textSize.value=state.prefs.text;annotations.checked=state.prefs.annotations;var target=config.surface;target.dataset.shopDensity=state.prefs.density;target.dataset.shopContrast=state.prefs.contrast;target.dataset.shopText=state.prefs.text;target.dataset.shopAnnotations=state.prefs.annotations?'on':'off';}
    [density,contrast,textSize,annotations].forEach(function(c){c.addEventListener('input',function(){state.prefs=prefs({density:density.value,contrast:contrast.value,text:textSize.value,annotations:annotations.checked});save();syncPrefs();});});
    prefsHost.append(button('Reset display preferences',function(){state.prefs=prefs();save();syncPrefs();}),el('small','Watchlists, plans, notes and preferences stay in this browser, separately for each account and shop. Clearing browser storage removes them.'));syncPrefs();
    global.addEventListener('storage',function(event){if(event.key!==key)return;var incoming=read(key,{});state.prefs=prefs(incoming&&incoming.prefs);syncPrefs();});
    function save(){if(!write(key,state))announce('Browser storage is unavailable; changes work for this visit.');}
    function announce(message){live.textContent=message;}
    function spendable(){var wallet=config.wallet();return {gold:budget(wallet.gold,state.gold,state.limit).available,tokens:budget(wallet.tokens,state.tokens).available};}
    function changed(){if(config.changed)config.changed();refresh();listeners.forEach(function(fn){fn();});}
    function record(offer){return {key:offer.key,name:offer.name,price:offer.price,currency:offer.currency||'gold',quantity:1,target:null,note:''};}
    function toggleWatch(offer){var index=state.watch.findIndex(function(x){return x.key===offer.key;});if(index>=0)state.watch.splice(index,1);else{if(state.watch.length>=50){announce('Watchlist holds up to 50 offers.');return;}state.watch.push(record(offer));}save();changed();announce(index>=0?'Removed from watchlist.':'Added to watchlist.');}
    function addPlan(offer){if(state.plan.some(function(x){return x.key===offer.key;})){announce('Already in your plan. Adjust its quantity in Shopping plan.');return;}if(state.plan.length>=20){announce('Plan holds up to 20 offers.');return;}state.plan.push(record(offer));save();changed();announce('Added '+offer.name+' to the plan.');}
    function addCompare(offer){if(comparisons.includes(offer.key))comparisons=comparisons.filter(function(k){return k!==offer.key;});else if(comparisons.length<3)comparisons.push(offer.key);else{announce('Compare up to three offers. Remove one first.');return;}refresh();}
    function annotate(offer){var card=offer.node;if(!card)return;var actions=card.querySelector('.sw-offer-actions');if(!actions){actions=el('div',null,'sw-offer-actions');card.appendChild(actions);actions.append(button('Watch',function(){toggleWatch(byKey.get(offer.key)||offer);},'ghost sw-watch'),button('Plan',function(){addPlan(byKey.get(offer.key)||offer);},'ghost sw-plan'),button('Compare',function(){addCompare(byKey.get(offer.key)||offer);},'ghost sw-compare'));}var watch=actions.querySelector('.sw-watch'),planned=actions.querySelector('.sw-plan'),compared=actions.querySelector('.sw-compare');watch.setAttribute('aria-pressed',String(state.watch.some(function(x){return x.key===offer.key;})));planned.setAttribute('aria-pressed',String(state.plan.some(function(x){return x.key===offer.key;})));compared.setAttribute('aria-pressed',String(comparisons.includes(offer.key)));}
    function planLines(){return state.plan.map(function(line){var current=byKey.get(line.key);return Object.assign({},line,current?{price:current.price,currency:current.currency}:{});});}
    function planText(){return planLines().map(function(x){return x.quantity+' × '+x.name+' · '+fmt(x.price*x.quantity)+' '+x.currency;}).join('\n')+'\n'+planSummary.textContent;}
    function renderLists(){
      watchList.replaceChildren();if(!state.watch.length)watchList.appendChild(el('p','Watch offers to track a target price.'));
      state.watch.forEach(function(line){var current=byKey.get(line.key),row=el('div',null,'sw-list-row');row.appendChild(el('strong',line.name));row.appendChild(el('span',current?fmt(current.price)+' '+current.currency:'Not in loaded stock'));
        var target=control(row,'','Target price','number'),note=control(row,'','Note','text');target.removeAttribute('id');note.removeAttribute('id');target.value=line.target===null?'':line.target;note.value=line.note;note.maxLength=200;
        target.addEventListener('change',function(){line.target=integer(target.value,null);save();refresh();});note.addEventListener('change',function(){line.note=note.value.slice(0,200);save();});
        if(current&&line.target!==null&&current.price<=line.target)row.appendChild(el('strong','Target price reached','sw-good'));
        var locate=button('Locate',function(){if(config.locate)config.locate(current);});locate.disabled=!current;row.append(locate,button('Remove',function(){state.watch=state.watch.filter(function(x){return x!==line;});save();changed();}));watchList.appendChild(row);
      });
      planList.replaceChildren();state.plan.forEach(function(line){var current=byKey.get(line.key),row=el('div',null,'sw-list-row');row.appendChild(el('strong',line.name));var quantity=control(row,'','Quantity','number');quantity.removeAttribute('id');quantity.min='1';quantity.max='99';quantity.value=line.quantity;quantity.addEventListener('change',function(){line.quantity=Math.max(1,Math.min(99,integer(quantity.value,1)));save();refresh();});row.appendChild(el('span',current?fmt(current.price*line.quantity)+' '+current.currency:'Unavailable · estimate '+fmt(line.price*line.quantity)+' '+line.currency));if(current&&current.price!==line.price)row.appendChild(el('small','Price changed: '+fmt(line.price)+' → '+fmt(current.price)));row.appendChild(button('Remove',function(){state.plan=state.plan.filter(function(x){return x!==line;});save();changed();},'ghost sw-plan-remove'));planList.appendChild(row);});
      var spend=spendable(),totals=plan(planLines(),spend,{});planSummary.textContent=!totals.valid?'Plan total cannot be represented safely.':fmt(totals.gold)+' gold · '+fmt(totals.tokens)+' tokens. Remaining: '+fmt(totals.goldLeft)+' gold / '+fmt(totals.tokenLeft)+' tokens.'+(totals.goldShortfall?' Need '+fmt(totals.goldShortfall)+' more gold.':'')+(totals.tokenShortfall?' Need '+fmt(totals.tokenShortfall)+' more tokens.':'');
      budgetSummary.textContent='Spendable: '+fmt(spend.gold)+' gold'+(config.tokens?' / '+fmt(spend.tokens)+' tokens':'')+'. Reserves: '+fmt(state.gold)+' gold'+(config.tokens?' / '+fmt(state.tokens)+' tokens':'')+'.';
    }
    function renderComparison(){compareTable.replaceChildren();var chosen=comparisons.map(function(k){return byKey.get(k);}).filter(Boolean);if(!chosen.length){compareTable.appendChild(el('p','Choose Compare on up to three offers.'));return;}var table=el('table'),head=el('tr');head.appendChild(el('th','Attribute'));chosen.forEach(function(o){var cell=el('th',o.name);cell.appendChild(button('Remove '+o.name,function(){comparisons=comparisons.filter(function(k){return k!==o.key;});refresh();}));head.appendChild(cell);});table.appendChild(head);
      var fields=[['Price',function(o){return fmt(o.price)+' '+o.currency;}],['Stat power',function(o){return fmt(o.power||0);}],['Power gain',function(o){return fmt(o.gain||0);}],['Effective XP %',function(o){return fmt(o.item&&o.item.xp_detail&&o.item.xp_detail.effective_bonus_pct||0);}],['Sockets',function(o){return fmt(o.item&&o.item.sockets||0);}],['HP / second',function(o){return fmt(o.item&&o.item.regen_rate||0);}]];
      var codes=Array.from(new Set(chosen.flatMap(function(o){return Object.keys(stats(o.item));}))).sort();codes.forEach(function(code){fields.push([code,function(o){return fmt(stats(o.item)[code]||0);}]);});fields.forEach(function(field){var tr=el('tr');tr.appendChild(el('th',field[0]));chosen.forEach(function(o){tr.appendChild(el('td',field[1](o)));});table.appendChild(tr);});compareTable.appendChild(table);if(new Set(chosen.map(function(o){return o.slot;})).size>1)compareTable.appendChild(el('p','Different slots: this table compares offers, not a combined equipment replacement.'));
    }
    function renderJournal(){journalList.replaceChildren();state.journal.forEach(function(entry){journalList.appendChild(el('li',new Date(entry.time).toLocaleString()+' · '+entry.message));});if(!state.journal.length)journalList.appendChild(el('li','No confirmed purchases recorded here yet.'));}
    function refresh(){offers.forEach(annotate);renderLists();renderComparison();}
    renderJournal();refresh();
    return {root:root,body:body,filters:filters,input:input,inputs:filterInputs,state:state,announce:announce,spendable:spendable,refresh:refresh,changed:changed,onChange:function(fn){listeners.push(fn);},isWatched:function(k){return state.watch.some(function(x){return x.key===k;});},isPlanned:function(k){return state.plan.some(function(x){return x.key===k;});},addPlan:addPlan,
      setOffers:function(values){offers=values;byKey=new Map(values.map(function(o){return [o.key,o];}));refresh();},receipt:function(message){state.journal.unshift({time:Date.now(),message:String(message).slice(0,400)});state.journal=state.journal.slice(0,30);save();renderJournal();},resetFilters:function(){Object.values(filterInputs).forEach(function(c){if(c.type==='checkbox')c.checked=false;else c.value='';});changed();}};
  };
})(window);
