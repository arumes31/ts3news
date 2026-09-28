const {defineConfig}=require('@playwright/test');
const base=require('./playwright.session-memory.config');
module.exports=defineConfig({...base,testMatch:'rift-direct-memory.spec.js',timeout:process.env.BRAWL_DIRECT_MEMORY_SMOKE==='1'?600000:3600000,outputDir:'test-results/direct-memory'});
