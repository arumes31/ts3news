const {test,expect}=require('@playwright/test');

test.describe('class performance, guard/dodge counts, and result hints',()=>{
  test.beforeEach(async({page})=>{
    await page.goto('/abyss/rift?scenario=terminal-defeated');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('0870: concise class-performance summary in run statistics',async({page})=>{
    await page.locator('.rift-run-statistics > summary').click();
    const row=page.locator('#rift-statistics dt').filter({hasText:'Class performance'}).locator('xpath=following-sibling::dd[1]');
    await expect(row).toBeVisible();
    const text=await row.textContent();
    expect(text).toMatch(/\d+ casts/);
    expect(text).toMatch(/\d+ damage/);
  });

  test('0871: successful guard counts in run statistics',async({page})=>{
    await page.locator('.rift-run-statistics > summary').click();
    // Set guards > 0 via run mutation
    const hasGuards=await page.evaluate(async()=>{
      const {run}=await(await fetch('/api/abyss/rift')).json();
      run.stats.guards=7;run.stats.perfect_guards=2;
      RiftHUD.update(run,false);
      const dt=[...document.querySelectorAll('#rift-statistics dt')].find(el=>el.textContent==='Successful guards');
      return dt?dt.nextElementSibling.textContent:'';
    });
    expect(hasGuards).toContain('7');
    expect(hasGuards).toContain('2 perfect');
  });

  test('0872: hazard evasion (dodge) counts in run statistics',async({page})=>{
    await page.locator('.rift-run-statistics > summary').click();
    const hasDodges=await page.evaluate(async()=>{
      const {run}=await(await fetch('/api/abyss/rift')).json();
      run.stats.dodges=4;
      RiftHUD.update(run,false);
      const dt=[...document.querySelectorAll('#rift-statistics dt')].find(el=>el.textContent==='Hazard evasions (airborne dodges)');
      return dt?dt.nextElementSibling.textContent:'';
    });
    expect(hasDodges).toContain('4');
  });

  test('0874: most effective skill by confirmed damage',async({page})=>{
    await page.locator('.rift-run-statistics > summary').click();
    const result=await page.evaluate(async()=>{
      const {run}=await(await fetch('/api/abyss/rift')).json();
      const skills=run.build.skills;
      run.stats.skill_damage={[skills[0].id]:10,[skills[1].id]:50,[skills[2].id]:20};
      RiftHUD.update(run,false);
      const dt=[...document.querySelectorAll('#rift-statistics dt')].find(el=>el.textContent==='Highest recorded skill damage');
      return {text:dt?dt.nextElementSibling.textContent:'',name:skills[1].name};
    });
    expect(result.text).toContain(result.name);
    expect(result.text).toContain('50 damage');
  });

  test('0875: shows a useful next-step hint on defeat',async({page})=>{
    // Clear any dismissed hints
    await page.evaluate(()=>localStorage.removeItem('riftDismissedHints'));
    await page.evaluate(async()=>{
      const {run}=await(await fetch('/api/abyss/rift')).json();
      run.status='defeated';run.stats.guards=0;run.stats.hits_taken=5;
      RiftHUD.update(run,false);
    });
    const hint=page.locator('#rift-result-hint');
    await expect(hint).toBeVisible();
    const text=await page.locator('#rift-result-hint-text').textContent();
    expect(text.length).toBeGreaterThan(10);
  });

  test('0876: does not repeat the same hint after dismissal',async({page})=>{
    await page.evaluate(()=>localStorage.removeItem('riftDismissedHints'));
    const firstHint=await page.evaluate(async()=>{
      const {run}=await(await fetch('/api/abyss/rift')).json();
      run.status='defeated';run.stats.guards=0;run.stats.hits_taken=5;
      RiftHUD.update(run,false);
      return document.getElementById('rift-result-hint-text').textContent;
    });
    expect(firstHint.length).toBeGreaterThan(0);
    // Dismiss
    await page.locator('#rift-dismiss-hint').click();
    await expect(page.locator('#rift-result-hint')).toBeHidden();
    // Re-render — should pick a different hint or hide if only one available
    const secondState=await page.evaluate(async()=>{
      const {run}=await(await fetch('/api/abyss/rift')).json();
      run.status='defeated';run.stats.guards=0;run.stats.hits_taken=5;
      run.id='different-run-id';
      RiftHUD.update(run,false);
      const el=document.getElementById('rift-result-hint');
      const txt=document.getElementById('rift-result-hint-text').textContent;
      return {hidden:el.hidden,text:txt};
    });
    // Either hidden (only one hint available and it was dismissed) or different text
    if(!secondState.hidden)expect(secondState.text).not.toBe(firstHint);
  });

  test('0877: allow permanent dismissal of result hints',async({page})=>{
    await page.evaluate(()=>localStorage.removeItem('riftDismissedHints'));
    await page.evaluate(async()=>{
      const {run}=await(await fetch('/api/abyss/rift')).json();
      run.status='defeated';run.stats.guards=0;run.stats.hits_taken=5;
      RiftHUD.update(run,false);
    });
    await expect(page.locator('#rift-result-hint')).toBeVisible();
    await page.locator('#rift-dismiss-hint').click();
    await expect(page.locator('#rift-result-hint')).toBeHidden();
    // Check localStorage persists the dismissal
    const persisted=await page.evaluate(()=>{
      const raw=localStorage.getItem('riftDismissedHints');
      return raw?JSON.parse(raw):[];
    });
    expect(persisted.length).toBeGreaterThan(0);
  });
});
