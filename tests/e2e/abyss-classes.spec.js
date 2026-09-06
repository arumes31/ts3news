const {test,expect}=require('@playwright/test');

async function openBuild(page,{locked=false,fail=false}={}){
 let payload;let posts=[];
 await page.route('**/api/abyss/classes',async route=>{
  if(!payload){const response=await route.fetch();payload=await response.json();payload.locked=locked;}
  if(fail){await route.fulfill({status:500,json:{ok:false,error:'Build service unavailable. Reload to retry.'}});return;}
  if(route.request().method()==='POST'){
   const body=route.request().postDataJSON();posts.push(body);
   if(locked){await route.fulfill({json:{ok:false,error:'Bank your run first.'}});return;}
   payload.state.selected=body.selected;payload.state.revision++;
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
});


test('reloaded active builds can return to the original build without losing saved skills',async({page})=>{
 await openBuild(page);await page.locator('#abyssClassChoose').click();await expect(page.locator('#abyssClassStatus')).toContainText('Build saved');
 await page.locator('#abyssClassReload').click();await expect(page.locator('#abyssClassStatus')).toContainText('active.');await expect(page.locator('#abyssClassLegacy')).toBeEnabled();
 await page.locator('#abyssClassLegacy').click();await expect(page.locator('#abyssClassStatus')).toContainText('Your original build is active');await expect(page.locator('#abyssClassLegacy')).toBeDisabled();
});
