CREATE TABLE IF NOT EXISTS abyss_shop_demand (
    demand_day DATE NOT NULL,
    item_key TEXT NOT NULL,
    purchases BIGINT NOT NULL DEFAULT 0 CHECK (purchases >= 0),
    PRIMARY KEY (demand_day, item_key)
);
