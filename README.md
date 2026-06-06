<p align="center">
  <img src="logo.png" width="160" alt="TS3News Logo" />
</p>

<h1 align="center">TS3 Free Game Notification Bot 🎮</h1>

<p align="center">
  <a href="https://github.com/arumes31/ts3news"><img src="https://img.shields.io/github/v/release/arumes31/ts3news?style=flat-square" alt="Latest Release" /></a>
  <a href="https://github.com/arumes31/ts3news/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/arumes31/ts3news/ci.yml?branch=main&style=flat-square" alt="CI Status" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/arumes31/ts3news?style=flat-square" alt="License" /></a>
</p>

A sophisticated, headless TeamSpeak 3 bot that notifies users of free PC games across multiple platforms (Steam, Epic, GOG, and more) while featuring a deep, automated RPG progression system.

---

## 🚀 Key Features

*   🏆 **Leveling System**: Users earn XP across **10,000 levels** (30 levels per tier), with procedurally generated names and Roman numerals.
*   ⚔️ **Group Combat**: Users in the same channel automatically form a **Party** to fight randomly spawned mobs and bosses during every notification cycle.
*   🦾 **24 Gear Slots**: A complete equipment system (Head, Chest, Finger 1 & 2, Mount, Companion, etc.) with functional combat stats.
*   ✨ **Enchantment System**: Rare mob drops that can be applied to gear for additional power. Overriden by higher-tier enchants.
*   👑 **Rare Titles**: Over 120 unique fantasy titles (e.g. *Overlord*, *Godslayer*) assigned as TS3 server groups with a **name prefix**.
*   🧪 **Consumables**: Potions and elixirs that are automatically consumed to restore HP or provide combat buffs.
*   🕒 **Stat Tracking**: Tracks **lifetime connection time** to reward long-term community members.
*   💀 **Sloth Decay**: Inactivity penalty for users offline for more than 7 days (2% XP loss per day).
*   ⚖️ **Auto-Balancing**: A **Combat Pity** system that buffs party stats if they suffer consecutive defeats.
*   🤖 **Persona Switching**: The bot renames itself based on context, adopting the **godsfinger** persona when delivering rare loot.

---

## 🕹️ Progression & RPG Systems

### 📈 Earning XP
*   **Game Pokes**: Scaled by the game's original price.
*   **Idle XP**: Even without new games, online users receive 50% base XP.
*   **Daily Login**: A flat **+5 XP** for the first connection each day.
*   **Combat**: Victory against mobs provides a significant XP boost.

### ✖️ XP Multipliers
| Modifier | Condition | Bonus |
| :--- | :--- | :---: |
| **Critical Hit** | 5% random chance on every poke. | **3.0x** |
| **Claim Streak** | Stay active for 3 / 5 / 7+ consecutive days. | **1.25x / 1.5x / 2.0x** |
| **Server Pop** | Every additional online user (excluding the bot). | **+5% per user (2x cap)** |
| **Party System** | Multiple users in the same channel. | **1.25x** |
| **INT Stat** | Base Intelligence stat from gear. | **Passive % Boost** |

### 🛡️ Equipment & Stats
Users manage **24 slots**. The bot follows a **Smart Auto-Equip** policy: it only replaces items if the new drop has a higher rarity or better overall stat score.
*   **Stats**: HP, STR (Damage), DEF (Reduces Damage), SPD (Turn Priority), LCK (Drop Rates), INT (XP Boost), STA (Reduces Dura Loss), CRT (Crit Chance), DGE (Dodge).
*   **Durability**: Gear loses 1 durability per fight (3 on defeat). Broken gear is automatically deleted.
*   **Unique Legendaries**: Rare, named items with massive stats but very low durability (e.g. *God-Slayer's Heart*).
*   **Enchantments**: Rare drops that add extra stats to a specific gear slot. Some enchantments specifically increase the item's **Durability**.

### ⚔️ Combat & Mobs
During every cycle, the bot spawns a group of mobs for each party.
*   **Mob Levels**: Mobs scale based on the party's average level and gear strength.
*   **Mob Effects**: Enemies can be **Enraged**, **Armored**, **Regenerative**, **Poisoned**, etc.
*   **Pity System**: Each consecutive defeat adds a stack of the **Combat Pity** buff (+20% stats per stack) until a victory is achieved.

---

## 🛠️ Setup & Deployment

### Option A: Using the Pre-built GHCR Image (Recommended)

1.  **Create `docker-compose.yml`**:
    ```yaml
    services:
      db:
        image: postgres:15-alpine
        container_name: ts3-news-db
        restart: unless-stopped
        environment:
          POSTGRES_USER: ${DB_USER:-ts3bot}
          POSTGRES_PASSWORD: ${DB_PASS:-ts3botpass}
          POSTGRES_DB: ${DB_NAME:-ts3news}
        volumes:
          - postgres_data:/var/lib/postgresql/data
        healthcheck:
          test: ["CMD-SHELL", "pg_isready -U ${DB_USER:-ts3bot} -d ${DB_NAME:-ts3news}"]
          interval: 5s
          timeout: 5s
          retries: 5

      ts3-bot:
        image: ghcr.io/arumes31/ts3news:latest
        container_name: ts3-news-bot
        restart: unless-stopped
        stop_grace_period: 30s
        depends_on:
          db:
            condition: service_healthy
        env_file:
          - config.env
        environment:
          - DATABASE_URL=postgres://${DB_USER:-ts3bot}:${DB_PASS:-ts3botpass}@db:5432/${DB_NAME:-ts3news}?sslmode=disable
        logging:
          driver: "json-file"
          options:
            max-size: "10m"
            max-file: "3"

    volumes:
      postgres_data:
    ```
2.  **Configure**: Create `config.env` (see `example.env`).
3.  **Run**: `docker compose up -d`

---

## 📄 License

This project is licensed under the MIT License.
