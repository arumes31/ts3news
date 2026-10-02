const {test,expect}=require('@playwright/test');

test.describe('Result details, mobile responsiveness, and print view',()=>{
  test.beforeEach(async({page})=>{
    await page.goto('/abyss/rift?scenario=terminal-defeated');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('0850: enemies defeated by family in encounter summary and run statistics',async({page})=>{
    await page.locator('.rift-run-statistics > summary').click();
    const result=await page.evaluate(async()=>{
      const {run}=await(await fetch('/api/abyss/rift')).json();
      run.monster_records={
        'monster:Ghoul':{defeats:4,first_seen_ms:1000},
        'monster:Skeleton':{defeats:2,first_seen_ms:1000}
      };
      RiftHUD.update(run,false);
      const dt=[...document.querySelectorAll('#rift-statistics dt')].find(el=>el.textContent==='Enemies defeated by family');
      const summary=RiftHUD.buildResultSummary(run);
      return {text:dt?dt.nextElementSibling.textContent:'',summary};
    });
    expect(result.text).toMatch(/Ghoul: 4|Skeleton: 2/);
    expect(result.summary).toContain('Defeated by Family:');
  });

  test('0853: elapsed active play time in run statistics',async({page})=>{
    await page.locator('.rift-run-statistics > summary').click();
    const result=await page.evaluate(async()=>{
      const {run}=await(await fetch('/api/abyss/rift')).json();
      run.stats.seconds=125.4;
      RiftHUD.update(run,false);
      const dt=[...document.querySelectorAll('#rift-statistics dt')].find(el=>el.textContent==='Elapsed active play time');
      return dt?dt.nextElementSibling.textContent:'';
    });
    expect(result).toContain('2m 5.4s');
    expect(result).toContain('125.4s');
  });

  test('0854: room-by-room time split in encounter and statistics',async({page})=>{
    await page.locator('.rift-run-statistics > summary').click();
    const result=await page.evaluate(async()=>{
      const {run}=await(await fetch('/api/abyss/rift')).json();
      run.room_splits=[14.2,28.5,45.1];
      RiftHUD.update(run,false);
      const dt=[...document.querySelectorAll('#rift-statistics dt')].find(el=>el.textContent==='Room time splits');
      const summary=RiftHUD.buildResultSummary(run);
      return {text:dt?dt.nextElementSibling.textContent:'',summary};
    });
    expect(result.text).toContain('Tier 1: 14.2s');
    expect(result.text).toContain('Tier 2: 28.5s');
    expect(result.text).toContain('Tier 3: 45.1s');
    expect(result.summary).toContain('Room Splits: Tier 1: 14.2s');
  });

  test('0855: completed optional objectives in statistics',async({page})=>{
    await page.locator('.rift-run-statistics > summary').click();
    const result=await page.evaluate(async()=>{
      const {run}=await(await fetch('/api/abyss/rift')).json();
      run.objectives={
        mission:1,finished:true,entries:[
          {name:'Clean Sweep',status:'complete',description:'Defeat all foes'},
          {name:'Untouchable',status:'failed',description:'Take no damage',reason:'hit by trap'}
        ]
      };
      RiftHUD.update(run,false);
      const dt=[...document.querySelectorAll('#rift-statistics dt')].find(el=>el.textContent==='Completed optional objectives');
      const summary=RiftHUD.buildResultSummary(run);
      return {text:dt?dt.nextElementSibling.textContent:'',summary};
    });
    expect(result.text).toContain('Clean Sweep');
    expect(result.text).toContain('1 completed');
    expect(result.summary).toContain('Completed Objectives: Clean Sweep');
  });

  test('0856: failed optional objectives with explanations in statistics',async({page})=>{
    await page.locator('.rift-run-statistics > summary').click();
    const result=await page.evaluate(async()=>{
      const {run}=await(await fetch('/api/abyss/rift')).json();
      run.objectives={
        mission:1,finished:true,entries:[
          {name:'Speed Clear',status:'failed',description:'Beat in 30s',reason:'exceeded time limit'}
        ]
      };
      RiftHUD.update(run,false);
      const dt=[...document.querySelectorAll('#rift-statistics dt')].find(el=>el.textContent==='Failed optional objectives');
      const summary=RiftHUD.buildResultSummary(run);
      return {text:dt?dt.nextElementSibling.textContent:'',summary};
    });
    expect(result.text).toContain('Speed Clear');
    expect(result.text).toContain('exceeded time limit');
    expect(result.summary).toContain('Failed Objectives: Speed Clear (exceeded time limit)');
  });

  test('0857: newly earned cosmetic milestones in statistics',async({page})=>{
    await page.locator('.rift-run-statistics > summary').click();
    const result=await page.evaluate(async()=>{
      const {run}=await(await fetch('/api/abyss/rift')).json();
      run.stats.perfect_guards=15;
      run.completed_levels=Array.from({length:10},(_,i)=>i+1);
      RiftHUD.update(run,false);
      const dt=[...document.querySelectorAll('#rift-statistics dt')].find(el=>el.textContent==='Cosmetic milestones');
      const summary=RiftHUD.buildResultSummary(run);
      return {text:dt?dt.nextElementSibling.textContent:'',summary};
    });
    expect(result.text).toContain('First Region Complete');
    expect(result.text).toContain('Perfect guard I');
    expect(result.summary).toContain('Cosmetic Milestones:');
  });

  test('0858: improved personal records in encounter and statistics',async({page})=>{
    await page.locator('.rift-run-statistics > summary').click();
    const result=await page.evaluate(async()=>{
      const {run}=await(await fetch('/api/abyss/rift')).json();
      run.last_clear={mission:1,first:false,records:['time','hits']};
      RiftHUD.update(run,false);
      const dt=[...document.querySelectorAll('#rift-statistics dt')].find(el=>el.textContent==='Improved personal records');
      const summary=RiftHUD.buildResultSummary(run);
      return {text:dt?dt.nextElementSibling.textContent:'',summary};
    });
    expect(result.text).toContain('Fastest clear time');
    expect(result.text).toContain('Fewest damaging hits');
    expect(result.summary).toContain('Improved Records: Fastest clear time · Fewest damaging hits');
  });

  test('0878: keep results readable at narrow mobile widths',async({page})=>{
    await page.setViewportSize({width:360,height:640});
    const banner=page.locator('#rift-result-banner');
    await expect(banner).toBeVisible();
    const bannerBox=await banner.boundingBox();
    expect(bannerBox.width).toBeLessThanOrEqual(360);
    const actions=page.locator('#rift-result-actions');
    await expect(actions).toBeVisible();
    const campaignBtn=page.locator('#rift-result-campaign');
    await expect(campaignBtn).toBeVisible();
  });

  test('0879: preserve the final encounter visual scene behind results',async({page})=>{
    const canvas=page.locator('#rift-canvas');
    await expect(canvas).toBeVisible();
    const overlay=page.locator('#rift-overlay');
    await expect(overlay).toBeVisible();
    const canvasStyle=await canvas.evaluate(el=>window.getComputedStyle(el).opacity);
    expect(parseFloat(canvasStyle)).toBeGreaterThan(0.5);
  });

  test('0880: provide a print-friendly personal result view',async({page})=>{
    const printBtn=page.locator('#rift-print-result');
    await expect(printBtn).toBeVisible();
    let printCalled=false;
    await page.exposeFunction('mockPrint',()=>{printCalled=true;});
    await page.evaluate(()=>{
      window.print=window.mockPrint;
    });
    await printBtn.click();
    expect(printCalled).toBe(true);
  });
});
