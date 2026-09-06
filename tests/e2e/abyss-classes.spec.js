const {test,expect}=require('@playwright/test');

async function openBuild(page,{locked=false,fail=false,fresh=false}={}){
 let payload;let posts=[];
 await page.route('**/api/abyss/classes',async route=>{
  if(!payload){const response=await route.fetch();payload=await response.json();payload.locked=locked;
   if(fresh)for(const c of payload.catalog){payload.class_progress[c.id]={xp:75000,points:5,clears:75,next_xp:575000,foundation:[],subclass_unlocked:false,best_depth:75};payload.state.progress[c.id]={xp:75000,foundation:[]};}
  }
  if(fail){await route.fulfill({status:500,json:{ok:false,error:'Build service unavailable. Reload to retry.'}});return;}
  if(route.request().method()==='POST'){
   const body=route.request().postDataJSON();posts.push(body);
   if(locked){await route.fulfill({json:{ok:false,error:'Bank your run first.'}});return;}
   payload.state.selected=body.selected;payload.state.class=body.class||'';payload.state.revision++;
   if(body.foundation){payload.class_progress[body.class].foundation=body.foundation;payload.class_progress[body.class].subclass_unlocked=body.foundation.length===5;}
   if(body.profile)payload.state.profiles[body.selected]=body.profile;
  }
  await route.fulfill({json:payload});
 });
 await page.goto('/abyss#abyssMyBuild');
 await expect(page.locator('#abyssMyBuild')).toBeVisible();
 if(!fail)await expect(page.locator('#abyssClassContent')).toBeVisible();
 return posts;
}

test('all six classes expose both subclasses, distinct empowered portraits and actionable signature costs',async({page})=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await openBuild(page);
 const expected={warrior:['vanguard','berserker'],ranger:['marksman','beastmaster'],arcanist:['elementalist','chronomancer'],warden:['oracle','geomancer'],reaver:['bloodblade','voidwalker'],artificer:['runesmith','alchemist']};const frames=new Set();
 for(const [classID,subs]of Object.entries(expected)){
  await page.locator('[data-class="'+classID+'"]').click();
  for(const sub of subs){await page.locator('#abyssSubclass').selectOption(sub);await expect(page.locator('.ab-class-signature')).toHaveCount(2);await expect(page.locator('#abyssClassSignatures')).toContainText('mana');
   frames.add(await page.locator('#abyssClassPortrait').evaluate(el=>el.style.backgroundImage+'|'+el.style.backgroundPosition));
   await page.locator('#abyssClassChoose').click();await expect(page.locator('#abyssClassStatus')).toContainText('Build saved');await expect(page.locator('#abyssClassChoose')).toHaveText('Active subclass');
  }
 }
 expect(frames.size).toBe(12);expect(errors).toEqual([]);
});

test('skill selections and pins return after switching subclasses and reload',async({page})=>{
 const posts=await openBuild(page);await page.locator('[data-class="arcanist"]').click();await page.locator('#abyssSubclass').selectOption('elementalist');
 await page.locator('[data-equip="S0_2"]').uncheck();await page.locator('[data-pin="S0_1"]').check();await page.locator('#abyssClassSaveSkills').click();await expect(page.locator('#abyssClassStatus')).toContainText('Build saved');
 await page.locator('#abyssSubclass').selectOption('chronomancer');await page.locator('#abyssClassChoose').click();await expect(page.locator('#abyssClassStatus')).toContainText('Build saved');
 await page.locator('#abyssSubclass').selectOption('elementalist');await expect(page.locator('[data-pin="S0_1"]')).toBeChecked();await expect(page.locator('[data-equip="S0_2"]')).not.toBeChecked();
 await page.locator('#abyssClassChoose').click();await expect(page.locator('#abyssClassStatus')).toContainText('Build saved');await page.locator('#abyssClassReload').click();await expect(page.locator('#abyssClassChoose')).toHaveText('Active subclass');await expect(page.locator('[data-pin="S0_1"]')).toBeChecked();
 expect(posts.find(p=>p.profile)?.profile.pins).toEqual(['S0_1']);
});

test('active runs lock mutation but keep subclass inspection usable',async({page})=>{
 const posts=await openBuild(page,{locked:true});await page.locator('[data-class="reaver"]').click();await page.locator('#abyssSubclass').selectOption('voidwalker');
 await expect(page.locator('#abyssClassChoose')).toBeDisabled();await expect(page.locator('#abyssClassSaveSkills')).toBeDisabled();await expect(page.locator('#abyssClassStatus')).toContainText('locked');expect(posts).toHaveLength(0);
});

test('failed reads preserve an explicit retry action',async({page})=>{await openBuild(page,{fail:true});await expect(page.locator('#abyssClassStatus')).toContainText('unavailable');await expect(page.locator('#abyssClassReload')).toBeEnabled();});

test('class sprite identity wins over equipped weapon for every subclass',async({page})=>{
 await openBuild(page);const identities=await page.evaluate(()=>{
  const art=window.AbyssCombatArt;const subs=['vanguard','berserker','marksman','beastmaster','elementalist','chronomancer','oracle','geomancer','bloodblade','voidwalker','runesmith','alchemist'];
  return subs.map(sub=>{const a=art.actorFrame({is_player:true,class:'warrior',subclass:sub,weapon_type:'bow'},'cast',1);const b=art.actorFrame({is_player:true,class:'warrior',subclass:sub,weapon_type:'staff'},'cast',1);return {identity:a.identity,same:a.asset===b.asset&&a.position===b.position,asset:a.asset,position:a.position};});
 });expect(new Set(identities.map(x=>x.asset+x.position)).size).toBe(12);expect(identities.every(x=>x.same)).toBe(true);
});

for(const viewport of [{width:1440,height:900},{width:390,height:844}])test('build layout at '+viewport.width+'px is usable without page overflow',async({page},testInfo)=>{
 await page.setViewportSize(viewport);await openBuild(page);await page.locator('[data-class="artificer"]').click();await page.locator('#abyssSubclass').selectOption('alchemist');
 await expect(page.locator('#abyssClassChoose')).toBeVisible();expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
 await page.locator('#abyssMyBuild').screenshot({path:testInfo.outputPath('my-build-'+viewport.width+'.png')});
 await page.locator('#abyssFoundationTree').screenshot({path:testInfo.outputPath('foundation-'+viewport.width+'.png')});await page.locator('#abyssSubclassTree').screenshot({path:testInfo.outputPath('subclass-'+viewport.width+'.png')});
});


test('reloaded active builds can return to the original build without losing saved skills',async({page})=>{
 await openBuild(page);await page.locator('#abyssClassChoose').click();await expect(page.locator('#abyssClassStatus')).toContainText('Build saved');
 await page.locator('#abyssClassReload').click();await expect(page.locator('#abyssClassStatus')).toContainText('active.');await expect(page.locator('#abyssClassLegacy')).toBeEnabled();
 await page.locator('#abyssClassLegacy').click();await expect(page.locator('#abyssClassStatus')).toContainText('Your original build is active');await expect(page.locator('#abyssClassLegacy')).toBeDisabled();
});


test('foundation choices unlock a subclass and cannot select every competing talent',async({page})=>{
 const posts=await openBuild(page,{fresh:true});
 await page.locator('#abyssSubclass').selectOption('vanguard');await expect(page.locator('#abyssClassChoose')).toBeDisabled();
 for(let tier=1;tier<=5;tier++)await page.locator('[data-talent="warrior_t'+tier+'_1"]').click();
 await page.locator('[data-talent="warrior_t1_2"]').click();await expect(page.locator('[data-talent="warrior_t1_2"]')).toBeFocused();
 await expect(page.locator('#abyssFoundationTree [aria-pressed=true]')).toHaveCount(5);
 await expect(page.locator('[data-talent="warrior_t1_1"]')).toHaveAttribute('aria-pressed','false');
 await page.locator('#abyssTalentSave').click();await expect(page.locator('#abyssClassStatus')).toContainText('Talent build saved');
 expect(posts.at(-1).foundation).toHaveLength(5);expect(posts.at(-1).selected).toBe('vanguard');
 await expect(page.locator('#abyssClassChoose')).toHaveText('Active subclass');
 await expect(page.locator('[data-talent="vanguard_t1_1"]')).toHaveAttribute('aria-disabled','true');await expect(page.locator('[data-talent="vanguard_t1_1"]')).toHaveAttribute('aria-label',/Earn the next/);
});

test('subclass path enforces ten-point budget and a single capstone',async({page})=>{
 const posts=await openBuild(page);await page.locator('#abyssSubclass').selectOption('vanguard');
 await expect(page.locator('[data-talent="vanguard_t6_1"]')).toHaveAttribute('aria-disabled','true');await expect(page.locator('[data-talent="vanguard_t6_1"]')).toHaveAttribute('aria-label',/previous tier/);
 for(let tier=1;tier<=4;tier++)for(const branch of [1,2])await page.locator('[data-talent="vanguard_t'+tier+'_'+branch+'"]').click();
 await page.locator('[data-talent="vanguard_t5_1"]').click();await page.locator('[data-talent="vanguard_t6_1"]').click();
 await page.locator('[data-talent="vanguard_t6_2"]').click();await expect(page.locator('#abyssSubclassTree [aria-pressed=true]')).toHaveCount(10);
 await expect(page.locator('[data-talent="vanguard_t6_1"]')).toHaveAttribute('aria-pressed','false');
 await page.locator('#abyssTalentSave').click();await expect(page.locator('#abyssClassStatus')).toContainText('Talent build saved');expect(posts.at(-1).profile.talents).toHaveLength(10);
 await page.locator('#abyssSubclass').selectOption('berserker');await page.locator('#abyssSubclass').selectOption('vanguard');await expect(page.locator('#abyssSubclassTree [aria-pressed=true]')).toHaveCount(10);
});

test('every talent icon loads and artwork differs across all 306 nodes',async({page})=>{
 await openBuild(page);const result=await page.evaluate(async()=>{
  const data=await fetch('/api/abyss/classes').then(r=>r.json());const icons=Object.values(data.talent_catalog).flatMap(t=>t.nodes.map(n=>n.art));
  const content=await Promise.all(icons.map(async path=>{const r=await fetch(path);if(!r.ok)throw Error(path);return (await r.text()).replace(/<title>.*?<\/title>/s,'');}));return {count:icons.length,unique:new Set(content).size};
 });expect(result).toEqual({count:306,unique:306});
});

test('one connected class tree fans out into both subclass extensions', async ({page}) => {
 await openBuild(page);
 const canvas = page.locator('#abyssTalentCanvas');
 await expect(canvas).toBeVisible();
 await expect(canvas.locator('[data-talent]')).toHaveCount(51);
 await expect(canvas.locator('[data-tree="warrior"] [data-talent]')).toHaveCount(15);
 await expect(canvas.locator('[data-tree="vanguard"] [data-talent]')).toHaveCount(18);
 await expect(canvas.locator('[data-tree="berserker"] [data-talent]')).toHaveCount(18);
 for (const subclass of ['vanguard', 'berserker']) {
  await expect(canvas.locator('[data-extends="warrior"][data-branch="'+subclass+'"]')).toHaveCount(3);
 }
 const layout = await canvas.evaluate(node => {
  const main = node.querySelector('[data-tree="warrior"]').getBoundingClientRect();
  return [...node.querySelectorAll('[data-tree="vanguard"], [data-tree="berserker"]')].every(branch => branch.getBoundingClientRect().top > main.bottom);
 });
 expect(layout).toBe(true);
});

test('hover and focus inspect node effects without allocating points', async ({page}) => {
 const posts = await openBuild(page);
 const talent = page.locator('[data-talent="warrior_t1_2"]');
 await talent.hover();
 await expect(page.getByRole('tooltip')).toContainText('+8% maximum HP');
 await expect(talent).toHaveAttribute('aria-pressed', 'false');
 await page.keyboard.press('Escape');
 await expect(page.getByRole('tooltip')).toBeHidden();
 await talent.focus();
 await expect(page.locator('#abyssTalentDetailTitle')).toContainText('Vitality');
 await expect(page.getByRole('tooltip')).toBeVisible();
 expect(posts).toHaveLength(0);
});

test('subclass branches share foundation choices and only the chosen extension can be edited', async ({page}) => {
 await openBuild(page);
 await page.locator('[data-subclass-choice="vanguard"]').click();
 await page.locator('[data-talent="vanguard_t1_1"]').click();
 await expect(page.locator('[data-talent="vanguard_t1_1"] .ab-talent-rank')).toHaveText('1/1');
 await expect(page.locator('[data-talent="berserker_t1_1"]')).toHaveAttribute('aria-disabled', 'true');
 await page.locator('[data-talent="berserker_t1_1"]').click({force:true});
 await expect(page.locator('[data-talent="berserker_t1_1"]')).toHaveAttribute('aria-pressed', 'false');
 await page.locator('[data-subclass-choice="berserker"]').click();
 await expect(page.locator('#abyssFoundationTree [aria-pressed=true]')).toHaveCount(5);
 await page.locator('[data-talent="berserker_t1_1"]').click();
 await page.locator('[data-subclass-choice="vanguard"]').click();
 await expect(page.locator('[data-talent="vanguard_t1_1"]')).toHaveAttribute('aria-pressed', 'true');
 await expect(page.locator('[data-talent="berserker_t1_1"]')).toHaveAttribute('aria-disabled', 'true');
});

test('tree zoom and mobile panning keep all nodes reachable without page overflow', async ({page}) => {
 await page.setViewportSize({width:390,height:844});
 await openBuild(page);
 const viewport = page.locator('#abyssTalentViewport');
 await expect(viewport).toBeVisible();
 await page.getByRole('button', {name:'Fit talent tree',exact:true}).click();
 expect(await viewport.evaluate(node => node.scrollWidth - node.clientWidth)).toBeLessThanOrEqual(1);
 await page.getByRole('button', {name:'Zoom in on talent tree',exact:true}).click();
 await page.getByRole('button', {name:'Zoom in on talent tree',exact:true}).click();
 await page.locator('[data-subclass-choice="berserker"]').click();
 await page.locator('[data-talent="berserker_t6_3"]').focus();
 await expect(page.locator('#abyssTalentDetailTitle')).toContainText('Endless Renewal');
 expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
});
