const {defineConfig}=require('@playwright/test');
const base=require('./playwright.boss-performance.config');
module.exports=defineConfig({...base,testMatch:'rift-crowd-frames.spec.js',outputDir:'test-results/crowd-frames'});
