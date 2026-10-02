const {defineConfig}=require('@playwright/test');
const port=process.env.ABYSS_E2E_PORT||'18098';
module.exports=defineConfig({
 testDir:'./tests/performance',testMatch:'rift-boss-frames.spec.js',timeout:180000,workers:1,retries:0,reporter:'line',
 outputDir:'test-results/boss-frames',
 use:{baseURL:`http://127.0.0.1:${port}`,viewport:{width:1280,height:900},deviceScaleFactor:1,locale:'en-US',timezoneId:'UTC',serviceWorkers:'block',trace:'off'},
 webServer:{command:'go test -tags=e2e ./internal/bot -run TestAbyssE2EServer -count=1 -v -timeout=15m',url:`http://127.0.0.1:${port}/healthz`,reuseExistingServer:false,timeout:120000},
});
