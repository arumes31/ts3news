const { test, expect } = require('@playwright/test');
const path = require('path');
const fs = require('fs');
const SELF = 'ally:contact', ENEMY = 'enemy:contact';
function state() { return {ok:true, session_id:'contact-e2e', phase:'planning', round:3, version:1,
 deadline:new Date(Date.now()+600000).toISOString(), tactic:'balanced', pause_mode:'adaptive', policy:{}, social:{},
 allies:[{id:SELF, entity_id:SELF,name:'Guardian',hp:800,max_hp:1000,mana:100,max_mana:100,shield:100,max_shield:100,is_self:true,is_player:true}],
 enemies:[{id:'enemy:0',entity_id:ENEMY,name:'Warden',hp:900,max_hp:1000,role:'boss',effects:[]}],
 options:[{kind:'attack',id:'',name:'Basic Attack',target:'enemy',cooldown:0}], recent_logs:[],log_cursor:0,initiative:[],enemy_intents:[],presentation_events:[],presentation_cursor:0}; }
async function open(page, initial) {
 for (const name of ['abyss_fight_visuals.js','abyss_combat_animation.js']) await page.route('**/static/'+name+'*', route=>route.fulfill({contentType:'text/javascript',body:fs.readFileSync(path.join(__dirname,'../../internal/bot/webassets',name),'utf8')}));
 await page.route('**/api/abyss/combat/state', route=>route.fulfill({json:{ok:false}}));
 await page.goto('/abyss?active=1');
 await page.evaluate(s=>{connectLiveCombat=()=>{};startLiveCombat({state:s},800,1000);},initial);
 await expect(page.locator('#liveCombat')).toBeVisible();
}
for (const mode of ['normal','fast','reduced']) test(`shield feedback and outcome history wait for contact (${mode})`, async ({page})=>{
 const initial=state(); await page.setViewportSize({width:1440,height:1000});
 if(mode==='reduced') await page.emulateMedia({reducedMotion:'reduce'});
 await open(page,initial);
 const next=structuredClone(initial);next.version++; next.allies[0].shield=0;next.allies[0].hp=750;next.presentation_cursor=1;
 next.presentation_events=[{seq:1,round:3,kind:'attack',actor_id:ENEMY,ability_name:'Warden strike',element:'physical',targets:[{target_id:SELF,damage:50,absorbed:100}]}];
 const before=await page.evaluate(({next,mode})=>{
   if(mode==='fast'){const speed=document.getElementById('liveAnimationSpeed');speed.value='fast';speed.dispatchEvent(new Event('change'));}
   window.contactEffects=[]; window.contactOrder=[];
   const host=document.getElementById('livePixelStage');
   new MutationObserver(records=>records.forEach(r=>r.addedNodes.forEach(n=>{
     if(n.nodeType!==1)return;
     if(n.matches('.ab-fight-flourish.absorb'))contactEffects.push(n.dataset.targetId);
     if(n.matches('.ab-combat-number'))contactOrder.push(n.textContent);
   }))).observe(host,{childList:true,subtree:true});
   renderLiveCombat(next);
   return {flourishes:host.querySelectorAll('.ab-fight-flourish.absorb').length,trails:host.querySelectorAll('.ab-hp-trail').length,history:document.getElementById('liveVisualHistory').textContent,announcement:document.getElementById('liveAnimationAnnouncement').textContent};
 },{next,mode});
 expect(before.flourishes).toBe(0);expect(before.trails).toBe(0);
 expect(before.history).not.toContain('50 HP');expect(before.history).not.toContain('absorbed 100');expect(before.announcement).not.toContain('absorbed 100');
 await expect(page.locator('#liveVisualHistory')).toContainText('Guardian: −50 HP, absorbed 100');
 await expect(page.locator('#livePixelStage')).toHaveAttribute('data-last-event-seq','1');
 expect(await page.evaluate(()=>contactEffects)).toEqual(mode==='reduced'?[]:[SELF]);
 expect(await page.evaluate(()=>contactOrder)).toEqual(['−50','ABSORB 100']);
 await page.evaluate(s=>renderLiveCombat(s),next);
 expect(await page.locator('#liveVisualHistory li[data-event-seq="1"]').count()).toBe(1);
 await page.screenshot({path:test.info().outputPath(`contact-${mode}.png`)});
});

test('chain history reveals each target only when its own projectile arrives', async ({ page }) => {
  const initial = state();
  initial.enemies.push({ id: 'enemy:1', entity_id: 'enemy:echo', name: 'Echo', hp: 600, max_hp: 600, effects: [] });
  await open(page, initial);
  const next = structuredClone(initial);
  next.version++; next.presentation_cursor = 1;
  next.presentation_events = [{ seq: 1, round: 3, kind: 'skill', actor_id: SELF,
    ability_id: 'arc-bolt', ability_name: 'Chain lightning', element: 'storm',
    targets: [{ target_id: ENEMY, damage: 71 }, { target_id: 'enemy:absent', damage: 101 }, { target_id: 'enemy:echo', damage: 29 }] }];
  await page.evaluate(snapshot => {
    window.chainHistory = [];
    const stage = document.getElementById('livePixelStage');
    new MutationObserver(records => records.forEach(record => record.addedNodes.forEach(node => {
      if (node.nodeType === 1 && node.matches('.ab-combat-number')) {
        chainHistory.push({ target: node.dataset.targetId, history: document.getElementById('liveVisualHistory').textContent });
      }
    }))).observe(stage, { childList: true, subtree: true });
    renderLiveCombat(snapshot);
  }, next);
  await expect(page.locator('#livePixelStage')).toHaveAttribute('data-last-event-seq', '1');
  const history = await page.evaluate(() => chainHistory);
  expect(history).toHaveLength(2);
  expect(history[0].history).toContain('Warden: −71 HP');
  expect(history[0].history).not.toContain('Echo: −29 HP');
  expect(history[1].history).toContain('Echo: −29 HP');
  await expect(page.locator('#liveVisualHistory li[data-event-seq="1"]')).toHaveCount(1);
  await expect(page.locator('#liveVisualHistory')).toContainText('Target: hit');
  await expect(page.locator('#liveVisualHistory')).not.toContainText('101');
});

test('concealed outcomes stay out of history, accessibility announcements and number titles', async ({ page }) => {
  const initial = state(); initial.enemies[0].hp_hidden = true;
  await open(page, initial);
  const next = structuredClone(initial); next.version++; next.presentation_cursor = 1;
  next.presentation_events = [{ seq: 1, round: 3, kind: 'attack', actor_id: SELF,
    ability_name: 'Basic Attack', targets: [{ target_id: ENEMY, damage: 98765, absorbed: 4321 }] }];
  await page.evaluate(snapshot => renderLiveCombat(snapshot), next);
  await expect(page.locator('#liveVisualHistory')).toContainText('Warden: hit, absorbed');
  const visible = await page.evaluate(() => ({
    history: document.getElementById('liveVisualHistory').textContent,
    announcement: document.getElementById('liveAnimationAnnouncement').textContent,
    numbers: [...document.querySelectorAll('.ab-combat-number')].map(n => ({ text: n.textContent, title: n.title })),
    actor: document.querySelector('[data-entity-id="enemy:contact"]').getAttribute('aria-label')
  }));
  expect(JSON.stringify(visible)).not.toMatch(/98765|4321|98[.,]765|4[.,]321/);
  expect(visible.actor).toContain('health concealed');
});
