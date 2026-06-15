-- Migration 0039: Augment System (Down)
-- Drops tables for TFT-style augment system

DROP TABLE IF EXISTS tft_augment_state;
DROP TABLE IF EXISTS tft_augment_offers;
DROP TABLE IF EXISTS tft_player_augments;
DROP TABLE IF EXISTS tft_augments;
