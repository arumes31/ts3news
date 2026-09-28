const {defineConfig}=require('@playwright/test');
const base=require('./playwright.boss-performance.config');
module.exports=defineConfig({...base,testMatch:'rift-input-performance.spec.js',timeout:600000,outputDir:'test-results/input-performance'});
