# Battle Simulation
# Aligned with Go: internal/bot/xp.go, internal/content/mobs.go, skills.go

import random
from data_structures import Player, Gear, Mob, xp_for_level, do_prestige, ALL_SLOTS

CRIT_CHANCE = 0.05
CRIT_MULT = 3.0
DURA_LOSS_PER_FIGHT = 1
DURA_LOSS_PENALTY = 3
DEATH_XP_PENALTY = 0.05

SKILL_CHANCE = 0.25
ULT_CHANCE = 0.02
STUN_CHANCE = 0.15
HEAL_CHANCE = 0.10


SKILL_PREFIXES = [
    "Mortal", "Heroic", "Flash", "Greater", "Lesser", "Chaos", "Fel", "Shadow", "Holy", "Frost",
    "Fire", "Arcane", "Divine", "Primal", "Ancient", "Abyssal", "Spectral", "Vengeful", "Spiteful", "Cursed",
    "Hallowed", "Glacial", "Volcanic", "Static", "Thunderous", "Corrupting", "Blighted", "Toxic", "Metallic", "Glass",
    "Lunar", "Solar", "Celestial", "Infernal", "Mystic", "Raging", "Silent", "Eternal", "Void", "Astral",
]
SKILL_ACTIONS = [
    "Strike", "Blast", "Roar", "Slash", "Burst", "Touch", "Nova", "Pulse", "Drain",
    "Bolt", "Ray", "Wave", "Aura", "Shield", "Fury", "Vortex", "Sunder", "Mend",
    "Bash", "Cleave", "Execute", "Rend", "Charge", "Leap", "Smite", "Shock",
]
ULTIMATE_VERBS = [
    "Annihilating", "Devastating", "Obliterating", "Shattering", "Eradicating",
    "Decimating", "Destroying", "Crushing", "Smashing", "Pulverizing",
    "Incinerating", "Freezing", "Corrupting", "Banishing", "Unleashing",
    "Rending", "Piercing", "Shredding", "Blasting", "Storming",
]
ULTIMATE_NOUNS = [
    "Strike", "Blast", "Wave", "Storm", "Fury",
    "Wrath", "Rage", "Nova", "Burst", "Flare",
    "Surge", "Pulse", "Beam", "Bolt", "Slash",
    "Barrage", "Volley", "Onslaught",
]

RARITY_POWER = {'Common': 1.0, 'Uncommon': 1.3, 'Rare': 1.6, 'Epic': 1.9, 'Legendary': 2.2}


def generate_skill(level):
    prefix = random.choice(SKILL_PREFIXES)
    action = random.choice(SKILL_ACTIONS)
    name = prefix + " " + action
    rar = random.choices(['Common', 'Uncommon', 'Rare', 'Epic', 'Legendary'],
                        weights=[50, 25, 15, 7, 3])[0]
    power = RARITY_POWER[rar]
    ign_def = 0.0
    stun = 0.0
    heal = 0.0
    if action == 'Sunder' or action == 'Execute':
        ign_def = 0.3 + random.random() * 0.4
    if action == 'Bash' or action == 'Shock':
        stun = 0.1 + random.random() * 0.3
    if action == 'Mend' or action == 'Heal':
        heal = 0.1 + random.random() * 0.4
    return {'name': name, 'rarity': rar, 'power': power, 'ignore_def': ign_def,
            'stun': stun, 'heal': heal, 'cooldown': 0}


def generate_ultimate():
    verb = random.choice(ULTIMATE_VERBS)
    noun = random.choice(ULTIMATE_NOUNS)
    name = verb + " " + noun
    rar = random.choices(['Rare', 'Epic', 'Legendary', 'Mythic', 'Divine'],
                        weights=[50, 25, 15, 7, 3])[0]
    power_map = {'Rare': 6.0, 'Epic': 8.0, 'Legendary': 10.0, 'Mythic': 12.0, 'Divine': 14.0}
    cd_map = {'Rare': 5, 'Epic': 7, 'Legendary': 9, 'Mythic': 11, 'Divine': 13}
    return {'name': name, 'rarity': rar, 'power': power_map[rar], 'cooldown': cd_map[rar], 'current_cd': 0} 

# Drop rates from Go xp.go rollLootForUser
ULTIMATE_SKILL_CHANCE = 0.005
TITLE_CHANCE = 0.005
UNIQUE_ITEM_CHANCE = 0.01
ARTIFACT_CHANCE = 0.01
ENCHANT_CHANCE = 0.02
SKILL_CHANCE = 0.05
CONS_CHANCE = 0.1
GEAR_CHANCE = 0.10

MOB_TEMPLATES = [
    ('Rat',      'Common',  {'HP': 20,  'STR': 5,  'DEF': 2,  'SPD': 5},  5),
    ('Slime',    'Common',  {'HP': 25,  'STR': 4,  'DEF': 3,  'SPD': 3},  5),
    ('Goblin',   'Common',  {'HP': 30,  'STR': 8,  'DEF': 3,  'SPD': 6},  8),
    ('Spider',   'Common',  {'HP': 22,  'STR': 7,  'DEF': 2,  'SPD': 8},  6),
    ('Zombie',   'Common',  {'HP': 35,  'STR': 6,  'DEF': 4,  'SPD': 4},  7),
    ('Wolf',     'Common',  {'HP': 28,  'STR': 10, 'DEF': 3,  'SPD': 10}, 9),
    ('Skeleton', 'Common',  {'HP': 32,  'STR': 9,  'DEF': 6,  'SPD': 5},  10),
    ('Bat',      'Common',  {'HP': 15,  'STR': 6,  'DEF': 1,  'SPD': 12}, 4),
    ('Orc',      'Common',  {'HP': 45,  'STR': 12, 'DEF': 5,  'SPD': 5},  12),
    ('Troll',    'Common',  {'HP': 60,  'STR': 14, 'DEF': 4,  'SPD': 4},  15),
    ('Skeletal Warrior', 'EliteMinion', {'HP': 70, 'STR': 18, 'DEF': 10, 'SPD': 8}, 20),
    ('Frenzied Ghoul',   'EliteMinion', {'HP': 65, 'STR': 22, 'DEF': 6,  'SPD': 12}, 22),
    ('Dread Knight', 'Elite', {'HP': 150, 'STR': 30, 'DEF': 20, 'SPD': 10}, 25),
    ('Orc Warchief', 'Miniboss', {'HP': 350, 'STR': 60, 'DEF': 35, 'SPD': 15}, 60),
    ('Ancient Dragon', 'Boss', {'HP': 1000, 'STR': 100, 'DEF': 50, 'SPD': 20}, 100),
    ('THE VOID LORD', 'Legendary', {'HP': 5000, 'STR': 300, 'DEF': 100, 'SPD': 50}, 500),
]

MOB_SPAWN_WEIGHTS = {
    'Common': 0.70,
    'EliteMinion': 0.15,
    'Elite': 0.08,
    'Miniboss': 0.04,
    'Boss': 0.02,
    'Legendary': 0.01,
}

MOB_RARITY_BONUS_XP = {
    'Common': 1.0,
    'EliteMinion': 1.25,
    'Elite': 1.5,
    'Miniboss': 2.0,
    'Boss': 2.5,
    'Legendary': 4.0,
}


def spawn_mob(player_level, difficulty=1.0):
    r = random.random()
    cumulative = 0.0
    mob_type = 'Common'
    for mt, weight in MOB_SPAWN_WEIGHTS.items():
        cumulative += weight
        if r <= cumulative:
            mob_type = mt
            break
    if mob_type == 'Legendary' and player_level < 25:
        mob_type = 'Common'
    elif mob_type == 'Boss' and player_level < 15:
        mob_type = 'Common'
    elif mob_type == 'Miniboss' and player_level < 10:
        mob_type = 'Common'
    elif mob_type == 'Elite' and player_level < 5:
        mob_type = 'Common'

    candidates = [t for t in MOB_TEMPLATES if t[1] == mob_type]
    if not candidates:
        candidates = [MOB_TEMPLATES[0]]
    template = random.choice(candidates)
    name, mtype, base_stats, base_xp = template

    lvl_scale = 1.0 + 0.005 * max(0, player_level - 1)
    effective_diff = 1.0 + (difficulty - 1.0) * 0.3
    total_scale = lvl_scale * effective_diff
    if total_scale < 0.1:
        total_scale = 0.1

    scaled_stats = {}
    for k, v in base_stats.items():
        if k == 'DEF':
            def_scale = 1.0 + (total_scale - 1.0) * 0.5
            scaled_stats[k] = max(1, int(v * def_scale))
        else:
            scaled_stats[k] = max(1, int(v * total_scale))

    level = max(1, int(player_level * lvl_scale))
    scaled_stats['SPD'] = level + random.randint(1, 5)

    reward_xp = int(base_xp * lvl_scale * difficulty * MOB_RARITY_BONUS_XP[mtype])
    if reward_xp < 1:
        reward_xp = 1

    # Increased gold drop for better AH simulation
    reward_gold = int(reward_xp * (5.0 + random.random() * 5.0))

    return Mob(name, mtype, level, scaled_stats, reward_xp, reward_gold)


def resolve_round(player, mob, intensify=1.0, heal_penalty=1.0, round_num=1, party_size=1, player_starts=True):
    logs = []
    user_dmg = 0
    mob_dmg = 0
    
    # Helper for user turn
    def user_turn_action():
        nonlocal user_dmg
        u_stats = player.total_stats()
        u_str = u_stats['STR']
        if random.random() < 0.1: u_str = int(u_str * 1.1)

        fatigue_mult = 1.0
        if round_num > 5: fatigue_mult = max(0.1, 1.0 - (round_num - 5) * 0.1)

        lifesteal = 0
        multi_strike = 0
        if player.title:
            lifesteal = getattr(player.title, 'lifesteal', 0)
            multi_strike = getattr(player.title, 'multi_strike', 0)
        for g in player.gear:
            if getattr(g, 'special', '') == 'Vampiric': lifesteal += 5

        hits = 1
        if multi_strike > 0 and random.randint(0, 99) < multi_strike:
            hits = 2
            logs.append(f"⚔️ Double attack!")

        for _ in range(hits):
            if mob.stats['HP'] <= 0: break
            dmg_mult = 1.0 * fatigue_mult
            ignore_def = 0.0
            heal_amount = 0
            stun_applied = False

            crit_chance = min(u_stats['CRT'] / 100.0, 0.25)
            if random.random() < crit_chance:
                dmg_mult *= CRIT_MULT
                logs.append("💥 CRITICAL HIT!")

            if player.skills and random.random() < 0.3:
                skill = random.choice(player.skills)
                if isinstance(skill, dict):
                    dmg_mult *= skill.get('power', 1.0)
                    ignore_def = skill.get('ignore_def', 0.0)
                    heal_amount = int(u_stats['HP'] * skill.get('heal', 0.0))
                    stun_applied = skill.get('stun', 0) > 0 and random.random() < skill['stun']
                    logs.append(f"📖 Skill: {skill['name']}!")

            eff_def = mob.stats['DEF'] * (1.0 - ignore_def)
            dmg = int((u_str * dmg_mult - eff_def) * intensify)
            min_dmg = int(u_str * 0.15 * intensify)
            dmg = max(min_dmg, dmg)
            if dmg < 1: dmg = 1

            mob.stats['HP'] -= dmg
            user_dmg += dmg
            if lifesteal > 0:
                ls_heal = int(dmg * lifesteal / 100.0 * heal_penalty)
                player.current_hp = min(u_stats['HP'], player.current_hp + ls_heal)
            if party_size >= 3 and random.random() < 0.3:
                user_dmg += dmg // 2
            if stun_applied and mob.stats['HP'] > 0:
                logs.append(f"💫 {mob.name} stunned!")
                mob.effects.append('Stunned')
            if heal_amount > 0 and player.current_hp > 0:
                player.current_hp = min(u_stats['HP'], player.current_hp + heal_amount)

    # Helper for mob turn
    def mob_turn_action():
        nonlocal mob_dmg
        if mob.stats['HP'] > 0 and 'Stunned' not in mob.effects:
            # Stealth check
            has_stealth = any(getattr(g, 'special', '') == 'Stealth' for g in player.gear)
            if round_num == 1 and has_stealth:
                logs.append("👤 Stealthed! Mob attack missed.")
                return

            # Parry check
            has_parry = any(getattr(g, 'special', '') == 'Parry' for g in player.gear)
            if has_parry and random.random() < 0.1:
                counter_dmg = int(player.total_stats()['STR'] * 0.5 * intensify)
                mob.stats['HP'] -= counter_dmg
                logs.append(f"🛡️ PARRIED and countered {mob.name} for {counter_dmg}!")
                return

            dodge = min(player.total_stats()['DGE'], 25)
            if random.randint(0, 99) < dodge:
                logs.append(f"💨 Dodged {mob.name}!")
                return

            m_str = mob.stats['STR']
            for eff in mob.effects:
                if eff == 'Enraged': m_str = int(m_str * 1.5)
                elif eff == 'Weakened': m_str = int(m_str * 0.5)

            spell_mult = 1.0
            if mob.spells and random.random() < 0.2:
                spell = random.choice(mob.spells)
                spell_mult = spell['power']

            dmg = int((m_str * spell_mult - player.total_stats()['DEF']) * intensify)
            # Tuned Damage Floor: 20% of STR
            min_dmg = int(m_str * 0.20 * intensify)
            dmg = max(min_dmg, dmg)
            if dmg < 1: dmg = 1

            if 'Blinded' in mob.effects and random.random() < 0.5:
                dmg = 0

            player.current_hp -= dmg
            mob_dmg += dmg

            if player.current_hp <= 0:
                has_phoenix = any(getattr(g, 'special', '') == 'Phoenix' for g in player.gear)
                if has_phoenix:
                    player.current_hp = player.total_stats()['HP'] // 2
                    logs.append("🔥 PHOENIX REBIRTH! Revived with 50% HP.")
                else:
                    logs.append(f"💀 You were slain by {mob.name}!")

    if player_starts:
        user_turn_action()
        if mob.stats['HP'] > 0: mob_turn_action()
    else:
        mob_turn_action()
        if player.current_hp > 0: user_turn_action()

    return logs, user_dmg, mob_dmg, max(0, player.current_hp), mob.stats['HP']


def simulate_battle(player, difficulty=1.0, party_size=1):
    max_rounds = 10
    mob_count = 1
    
    player_hp = player.total_stats()['HP']
    player.current_hp = player_hp
    
    # Wave Logic (1-3 waves)
    waves = 1
    if random.random() < 0.2: waves = 2
    if random.random() < 0.05: waves = 3
    
    total_logs = [f"⚔️ BATTLE! {waves} Waves incoming!"]
    victory = False
    all_encountered_mobs = []
    
    for w in range(1, waves + 1):
        mobs = [spawn_mob(player.level, difficulty) for _ in range(mob_count)]
        all_encountered_mobs.extend(mobs)
        if w > 1:
            total_logs.append(f"📢 WAVE {w} APPROACHES!")
            
        player_starts = random.random() < 0.5
        if not player_starts: total_logs.append("⚠️ AMBUSH! Enemies attack first!")
        
        wave_victory = False
        for rnd in range(1, max_rounds + 1):
            intensify = 1.0 + (rnd - 1) * 0.15
            heal_penalty = 1.0 if rnd <= 5 else max(0, 1.0 - (rnd - 5) * 0.2)
            
            alive_mobs = [m for m in mobs if m.stats['HP'] > 0]
            if not alive_mobs:
                wave_victory = True
                break
                
            for mob in alive_mobs:
                rlogs, ud, md, ph, mh = resolve_round(player, mob, intensify, heal_penalty, rnd, party_size, player_starts)
                total_logs.extend(rlogs)
                player.current_hp = ph
                mob.stats['HP'] = mh
                
            if player.regen_stacks > 0:
                heal = int(player.regen_stacks * 2 * heal_penalty)
                player.current_hp = min(player.total_stats()['HP'], player.current_hp + heal)
            
            if player.current_hp <= 0: break
            
        if player.current_hp <= 0:
            victory = False
            break
        if wave_victory:
            if w == waves: 
                victory = True
                total_logs.append("🏁 VICTORY! All waves defeated.")
            else: 
                total_logs.append(f"🏁 WAVE {w} CLEARED!")
                continue
            
    return victory, 10, all_encountered_mobs, total_logs


def roll_loot(player, difficulty=1.0):
    r = random.random()
    quality_mult = max(1.0, difficulty)

    if r < ULTIMATE_SKILL_CHANCE * quality_mult:
        return None
    elif r < (ULTIMATE_SKILL_CHANCE + TITLE_CHANCE) * quality_mult:
        return None
    elif r < (ULTIMATE_SKILL_CHANCE + TITLE_CHANCE + UNIQUE_ITEM_CHANCE) * quality_mult:
        return {'type': 'gear', 'item': None, 'note': 'Unique Item drop!'} 
    elif r < (ULTIMATE_SKILL_CHANCE + TITLE_CHANCE + UNIQUE_ITEM_CHANCE + ARTIFACT_CHANCE) * quality_mult:  
        return None
    elif r < (ULTIMATE_SKILL_CHANCE + TITLE_CHANCE + UNIQUE_ITEM_CHANCE + ARTIFACT_CHANCE + ENCHANT_CHANCE) * quality_mult:
        return None
    elif r < (ULTIMATE_SKILL_CHANCE + TITLE_CHANCE + UNIQUE_ITEM_CHANCE + ARTIFACT_CHANCE + ENCHANT_CHANCE + SKILL_CHANCE) * quality_mult:
        return {'type': 'skill', 'item': None, 'note': 'Learned skill'}
    elif r < (ULTIMATE_SKILL_CHANCE + TITLE_CHANCE + UNIQUE_ITEM_CHANCE + ARTIFACT_CHANCE + ENCHANT_CHANCE + SKILL_CHANCE + CONS_CHANCE) * quality_mult:
        return {'type': 'xp', 'item': 1, 'note': 'Consumable'}
    elif r < (ULTIMATE_SKILL_CHANCE + TITLE_CHANCE + UNIQUE_ITEM_CHANCE + ARTIFACT_CHANCE + ENCHANT_CHANCE + SKILL_CHANCE + CONS_CHANCE + GEAR_CHANCE) * quality_mult:
        return None
    else:
        # Scrap Stack Logic
        player.scrap_stack = getattr(player, 'scrap_stack', 0) + 1
        if player.scrap_stack > 5: player.scrap_stack = 5
        return {'type': 'xp', 'item': player.scrap_stack, 'note': f'Looted Scrap (+{player.scrap_stack} XP)'}


def run_combat_cycle(player, difficulty=1.0):
    battles = 1 + random.randint(0, 2)
    wins = 0
    losses = 0
    gear_drops = []
    total_xp = 0
    total_gold = 0
    logs = []

    for _ in range(battles):
        victory, rounds, mobs, battle_logs = simulate_battle(player, difficulty)
        player.battles_simulated += 1
        logs.extend(battle_logs)

        if victory:
            wins += 1
            player.win_count += 1
            player.consecutive_losses = 0
            battle_xp_accum = 0
            for mob in mobs:
                if mob.stats['HP'] <= 0:
                    battle_xp_accum += mob.reward_xp
                    total_gold += mob.reward_gold
                    drop = roll_loot(player, difficulty)
                    if drop:
                        if drop['type'] == 'gear':
                            pass
                        elif drop['type'] == 'xp':
                            battle_xp_accum += drop['item']
                        logs.append(f"🎁 {drop['note']}")
            
            # Apply gear XP multipliers to combat rewards
            total_xp += int(battle_xp_accum * player.gear_xp_multiplier())
            
            if player.regen_stacks > 0:
                player.regen_stacks += 1
        else:
            losses += 1
            player.lose_count += 1
            player.consecutive_losses += 1
            player.current_hp = 0
            for g in player.gear:
                if hasattr(g, 'durability') and isinstance(g.durability, int):
                    g.durability -= DURA_LOSS_PENALTY
            cur_xp = player.experience
            penalty = int(cur_xp * DEATH_XP_PENALTY)
            if penalty < 10:
                penalty = 10
            total_xp -= penalty
            player.regen_stacks = 0

    return {
        'wins': wins, 'losses': losses, 'gear_drops': gear_drops,
        'total_xp': total_xp, 'total_gold': total_gold, 'logs': logs, 'broken': 0
    }
