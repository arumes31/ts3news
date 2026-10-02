const {test,expect}=require('@playwright/test');
test('expired expedition keeps completed missions selectable through reload and restart',async({page})=>{
 await page.goto('/abyss/rift?scenario=terminal-expired');await expect(page.locator('#rift-start')).toBeEnabled();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;const expired=await saved();expect(expired.status).toBe('expired');expect(expired.completed_levels).toEqual([1,7]);
 await expect(page.locator('#rift-progress')).toContainText('2/100 completed');await page.locator('#rift-completion').selectOption('complete');
 const visible=()=>page.locator('#rift-levels [data-level]').evaluateAll(nodes=>nodes.filter(n=>!n.hidden).map(n=>Number(n.dataset.level)));expect(await visible()).toEqual([1,7]);await expect(page.locator('#rift-history-1')).toContainText('Best 42.0s');
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();expect(await visible()).toEqual([1,7]);expect((await saved()).mission_history).toEqual(expired.mission_history);
 const mission=page.locator('#rift-levels [data-level="7"]');await expect(mission).toBeEnabled();await mission.click();await expect(page.locator('#rift-start')).toHaveText('Enter mission 7');await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();await page.keyboard.press('Escape');
 const restarted=await saved();expect(restarted.id).not.toBe(expired.id);expect(restarted.level.id).toBe(7);expect(restarted.completed_levels).toEqual([1,7]);expect(restarted.mission_history['1']).toEqual(expired.mission_history['1']);expect(restarted.mission_history['7'].completions).toBe(1);expect(restarted.mission_history['7'].attempts).toBe(2);
 await expect(page.locator('#rift-progress')).toContainText('2/100 completed');
});
