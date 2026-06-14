<p align="center">
  <img src="logo.png" width="180" alt="TS3News Logo" />
</p>

<h1 align="center">TS3 Free Game RPG Bot 🎮</h1>

<p align="center">
  <a href="https://github.com/arumes31/ts3news/releases"><img src="https://img.shields.io/github/v/release/arumes31/ts3news?style=for-the-badge&color=7289da" alt="Latest Release" /></a>
  <a href="https://github.com/arumes31/ts3news/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/arumes31/ts3news/ci.yml?branch=main&style=for-the-badge" alt="CI Status" /></a>
  <a href="https://github.com/arumes31/ts3news/pkgs/container/ts3news"><img src="https://img.shields.io/badge/Container-GHCR-blue?style=for-the-badge&logo=docker" alt="GHCR Image" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/arumes31/ts3news?style=for-the-badge&color=success" alt="License" /></a>
</p>

<p align="center">
  <strong>A sophisticated, headless TeamSpeak 3 bot that notifies users of free PC games while featuring a deep, automated RPG progression system.</strong>
</p>

---

## 📐 Architecture & Flow

The bot runs a headless official TeamSpeak 3 client in a virtual framebuffer (Xvfb), controlled by a high-performance Go supervisor.

```mermaid
graph TD
    subgraph "Docker Container"
        TS3Client["🎮 Headless TS3 Client"]
        GoBot["🤖 Go Bot Supervisor"]
        PostgresDB["🗄️ PostgreSQL DB"]
        Xvfb["🖥️ Virtual Display"]
        
        Xvfb --> TS3Client
        GoBot <-->|ClientQuery TCP:25639| TS3Client
        GoBot <-->|SQL| PostgresDB
    end
    
    GoBot -->|Fetch| GamerPower["🌐 GamerPower API"]
    GoBot -->|Shorten| Redrx["🔗 RedRx API"]
    GoBot -->|Scrape| Reddit["📰 Reddit /r/FreeGameFindings"]
    
    TS3Client <-->|UDP:9987| TS3Server["🔊 TS3 Voice Server"]
    TS3Client -->|Poke & PM| TS3Users["👥 Online TS3 Users"]
    
    classDef default fill:#2d3748,stroke:#4a5568,stroke-width:1px,color:#fff;
    classDef Ext fill:#d69e2e,stroke:#b7791f,stroke-width:1px,color:#fff;
    classDef TS3 fill:#3182ce,stroke:#2b6cb0,stroke-width:1px,color:#fff;
    classDef DB fill:#336791,stroke:#224466,stroke-width:1px,color:#fff;
    class GamerPower,Redrx,Reddit Ext;
    class TS3Server TS3;
    class PostgresDB DB;
```

---

## 🚀 Key Features

*   🏆 **Legendary Leveling**: 10,000+ levels across 330+ tiers with procedurally generated fantasy names.
*   ⚔️ **Group Combat**: Users in the same channel automatically form a **Party** to fight randomly spawned mobs and bosses during every cycle.
*   🦾 **Massive Loot**: 24 equipment slots, 1,200+ gear variants, and 120+ rare titles.
*   🪄 **Skill System**: Over **300 unique skills and spells**. Users have 5 slots and automatically learn better skills found from loot.
*   ✨ **Enchantment System**: Rare mob drops that can be applied to gear for additional power or increased **Durability**.
*   🧪 **Consumables**: Potions and elixirs that are automatically consumed to restore HP or provide buffs.
*   🕒 **Persistence**: Full lifetime connection tracking and notification history stored in PostgreSQL.
*   ⚖️ **Auto-Balancing**: A **Combat Pity** system that buffs party stats if they suffer consecutive defeats.
*   🤖 **Contextual Personas**: The bot renames itself based on context, adopting the **godsfinger** persona for rare loot.
*   🌍 **Multilingual**: Every user-facing message — pokes, PMs, combat logs, loot, and content — is fully localized. Ships with **20 built-in languages**, selectable via a single `LANG` setting, with locale-aware number/currency formatting and pluralization.
*   🖥️ **Headless Reliability**: Runs the official TS3 desktop client in Xvfb with a robust Go watchdog for 24/7 uptime.

---

## 🕹️ RPG Systems Deep-Dive

### 📈 Progression Mechanics
Your XP award per cycle is influenced by a complex set of multipliers:
| Modifier | Condition | Bonus |
| :--- | :--- | :---: |
| **Critical Hit** | 5% random chance on every poke. | **3.0x** |
| **Claim Streak** | Stay active for 3 / 5 / 7+ consecutive days. | **1.25x / 1.5x / 2.0x** |
| **Server Population** | Every additional online user (excluding the bot). | **+5% per user (2x cap)** |
| **Party System** | Multiple users sitting in the same channel. | **1.25x** |
| **INT Stat** | Cumulative Intelligence stat from your gear. | **Passive % Boost** |

### 🛡️ Equipment & Stats
Users manage **24 slots**. The bot follows a **Smart Auto-Equip** policy: it only replaces items if the new drop has a higher rarity or better overall stat score.

*   **Combat Stats**: HP (Health), STR (Damage), DEF (Damage Reduction), SPD (Turn Priority), LCK (Drop Quality), INT (XP Boost), STA (Reduces Dura Loss), CRT (Crit Chance), DGE (Dodge).
*   **Flavour Stats**: Charisma, Stench, Shiny, Hunger — affecting your personal report messages.
*   **Durability**: Gear loses 1 durability per fight (**3 on defeat**). Broken gear is automatically deleted. Use **Reinforcing Enchantments** to restore or increase max durability.
*   **Unique Legendaries**: Ultra-rare, named items with massive stats but very low durability (e.g. *Infinity Edge*).

### ⚔️ Combat & Mobs
During every notification cycle, a random encounter occurs for each party.
*   **Mob Scaling**: Enemies level up with you and gain gear-aware difficulty boosts.
*   **Mob Effects**: Enemies can spawn with effects like **Enraged** (+STR), **Armored** (+DEF), **Regenerative**, or **Poisoned**.
*   **World Bosses**: Rarely, a legendary boss will spawn, requiring high stats and party cooperation to defeat.

---

## ⚙️ Configuration

The bot is configured via environment variables or a `config.env` file.

| Category | Variable | Description | Default |
| :--- | :--- | :--- | :---: |
| **Server** | `TS3_HOST` | Hostname or IP of the TeamSpeak 3 server. | *Required* |
| | `TS3_PORT` | Voice port of the server (UDP). | `9987` |
| | `TS3_IDENTITY` | Your exported TeamSpeak identity string. | *None* |
| | `TS3_NICKNAME` | Default nickname for the bot. | `MrFree` |
| **Cycle** | `MIN_INTERVAL_HOURS` | Minimum random sleep between cycles. | `1` |
| | `MAX_INTERVAL_HOURS` | Maximum random sleep between cycles. | `12` |
| **Localization** | `LANG` | Language for all bot messages (BCP-47 locale ID, see list below). Falls back to `en_US` if unset or unsupported. | `en_US` |
| **Web Portal** | `WEB_ENABLE` | Serve the player web portal and PM each user a login link per cycle. | `true` |
| | `WEB_LISTEN_ADDR` | Address the web server listens on. | `:18081` |
| | `WEB_BASE_URL` | Public base URL used to build per-user login links. | `http://localhost:18081` |
| **System** | `ENABLE_GAME_NEWS` | Master switch for the free game notification feature. | `true` |
| | `POKE_DELAY_MS` | Delay between individual pokes (anti-flood). | `1200` |
| | `RESEND_AFTER_DAYS` | Allow re-sending a game after N days. | `60` |
| | `DEAD_USER_DAYS` | Purge users inactive for N days. | `180` |
| **RPG** | `ENABLE_RPG` | Master switch for all RPG mechanics (combat, gear, skills, pets, AH). | `true` |
| | `RPG_BASE_HP` | Player starting HP, tweak to balance early-game win rates. | `100` |
| | `RPG_BASE_STR` | Player starting STR, tweak to balance early-game win rates. | `10` |
| | `RPG_BASE_DEF` | Player starting DEF, tweak to balance early-game win rates. | `5` |
| **Leveling** | `ENABLE_LEVELING` | Master switch for the XP and Rank systems. Works standalone. | `true` |
| | `ENABLE_XP_MODIFIERS` | Enable streaks, crits, and gear multipliers. | `true` |
| | `XP_SERVER_GROUPS` | Auto-create TS3 server groups for rank tiers. | `false` |
| | `CHEAPER_MORE_XP` | Invert XP scaling (cheaper games give more). | `false` |
| | `RESEND_AFTER_DAYS` | Allow re-sending a game after N days. | `60` |
| **Sources** | `ENABLE_GAMERPOWER` | Fetch from GamerPower API. | `true` |
| | `ENABLE_EPIC` | Fetch from Epic Games Store API. | `true` |
| | `ENABLE_REDDIT` | Fetch from /r/FreeGameFindings RSS. | `true` |
| | `REDRX_API_KEY` | API Key for [redrx.eu](https://redrx.eu/) link shortening. | *None* |
| **Database** | `DATABASE_URL` | PostgreSQL connection string. | *None* |
| | `DEAD_USER_DAYS` | Purge users inactive for N days. | `180` |

---

## 🌍 Localization

All user-facing text is externalized into locale files (`internal/i18n/locales/*.yaml`) and embedded into the binary at build time — no extra files to deploy. Set the active language with the `LANG` variable; anything missing in a locale automatically falls back to `en_US`.

```env
# config.env
LANG=de_DE
```

**20 supported locales:**

| Code | Language | Code | Language | Code | Language | Code | Language |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| `en_US` | English | `de_DE` | German | `es_ES` | Spanish | `fr_FR` | French |
| `it_IT` | Italian | `pt_BR` | Portuguese (BR) | `nl_NL` | Dutch | `sv_SE` | Swedish |
| `pl_PL` | Polish | `cs_CZ` | Czech | `tr_TR` | Turkish | `ru_RU` | Russian |
| `ja_JP` | Japanese | `ko_KR` | Korean | `zh_CN` | Chinese (Simpl.) | `zh_TW` | Chinese (Trad.) |
| `th_TH` | Thai | `vi_VN` | Vietnamese | `hi_IN` | Hindi | `ar_SA` | Arabic |

> Numbers, currency, and pluralization are formatted per-locale (e.g. `1,234.50` vs `1.234,50`). To add or refine a translation, copy `en_US.yaml`, translate the values, and add the locale ID to `AllLocales` in `internal/i18n/i18n.go`.

---

## 🌐 Web Portal

Alongside the TeamSpeak bot, the binary serves a token-authenticated **player web portal**. Every cycle each user is PM'd a personal, [redrx.eu](https://redrx.eu/)-shortened **login link** (a persistent unique token per user — keep it private). No passwords; the link logs you in.

| Page | What it does |
| :--- | :--- |
| **🛡️ Armoury** | WoW-armoury-style character sheet: rank, level, prestige, HP/XP bars, full attribute spread and all equipment slots. |
| **🎒 Inventory** | Owned, unequipped gear — equip, vendor for gold, or list on the auction house; plus consumables. |
| **⚔️ Auto-Battler** | A **Teamfight-Tactics-style board**: buy champions from a rolling shop, drag them onto your half of the grid, and start combat. Units auto-fight with animated sprites, HP bars and floating damage; 3 identical champions auto-combine into a star-up. Win to farm gold and gear. |
| **🎮 Arcade** | Five fully animated gold games: 🎰 5-reel **Slots**, 🎲 **Dice**, 🪙 3D **Coin Flip**, 🎡 canvas **Fortune Wheel**, 🃏 **High/Low** cards. |
| **🛒 Shop** | Currency exchange — **gold → XP at 1:3**, **XP → gold at 2:1** — plus a daily-rotating item shop with fair, combat-rating-based prices. |
| **🏛️ Auction House** | Browse and buy live player listings, list your own inventory items, and review your full buy/sell history. |

All gold/XP is the same economy used by the TS3 RPG, so farming the arcade or battler directly improves the player's character. The portal is self-contained (Go `html/template` + embedded CSS/JS, no build step) and configured via `WEB_ENABLE`, `WEB_LISTEN_ADDR` and `WEB_BASE_URL` (see Configuration).

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
        ports:
          - "18081:18081"
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
2.  **Configure**: Create `config.env` using `example.env` as a template.
3.  **Run**: `docker compose up -d`

### Option B: Building from Source

1.  **Clone the repository**.
2.  **Configure**: Create `config.env` with your settings.
3.  **Run**: Start the container and build:
    ```bash
    docker compose up -d --build
    ```

---

## 💻 Local Development & Testing

If you have Go installed, you can run the automated tests to verify the RPG logic:

```bash
# Run all unit tests
go test -v ./...
```

The tests verify notification filtering, database persistence, combat resolution, and loot logic.

---

## 📄 License

This project is licensed under the MIT License.

<p align="center">
  <em>Made with ⚔️ and 🎲 for the TeamSpeak community.</em>
</p>
