const {defineConfig}=require('@playwright/test');
const base=require('./playwright.boss-performance.config');
const port=process.env.ABYSS_E2E_PORT||'18098';
module.exports=defineConfig({...base,testMatch:'rift-warm-start.spec.js',outputDir:'test-results/warm-start',webServer:{...base.webServer,env:{...base.webServer.env,ABYSS_E2E_PORT:port}}});
