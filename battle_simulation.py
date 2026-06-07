# Battle Simulation
# Aligned with Go: internal/bot/xp.go, internal/content/mobs.go, skills.go

import random
from data_structures import Player, Gear, Mob, xp_for_level, do_prestige, ALL_SLOTS, ELEMENTS, RARITIES

CRIT_CHANCE = 0.05
CRIT_MULT = 3.0
DURA_LOSS_PER_FIGHT = 1
DURA_LOSS_PENALTY = 3
DEATH_XP_PENALTY = 0.05

RARITY_POWER = {'Common': 1.0, 'Uncommon': 1.3, 'Rare': 1.6, 'Epic': 1.9, 'Legendary': 2.2, 'Mythic': 2.5, 'Divine': 3.0}

def get_element_mult(attacker, defender):
    # Fire > Air > Earth > Water > Fire
    if attacker == 'Fire':
        if defender == 'Air': return 2.0
        if defender == 'Water': return 0.5
    elif attacker == 'Air':
        if defender == 'Earth': return 2.0
        if defender == 'Fire': return 0.5
    elif attacker == 'Earth':
        if defender == 'Water': return 2.0
        if defender == 'Air': return 0.5
    elif attacker == 'Water':
        if defender == 'Fire': return 2.0
        if defender == 'Earth': return 0.5
    return 1.0

def spawn_mob(player_level, difficulty=1.0):
    lvl_scale = 1.0 + 0.005 * max(0, player_level - 1)
    effective_diff = 1.0 + (difficulty - 1.0) * 0.3
    total_scale = lvl_scale * effective_diff
    
    stats = {
        'HP': int(100 * total_scale),
        'STR': int(15 * total_scale),
        'DEF': int(5 * total_scale),
        'SPD': int(10 * total_scale),
    }
    reward_xp = int(20 * total_scale)
    reward_gold = int(reward_xp * 5)
    
    element = random.choice(ELEMENTS) if random.random() < 0.4 else 'Physical'
    return Mob("Test Mob", "Common", player_level, stats, reward_xp, reward_gold, element)

def resolve_round(player, mob, intensify=1.0, heal_penalty=1.0, round_num=1, party_size=1, player_starts=True):
    logs = []
    user_dmg = 0
    mob_dmg = 0
    
    def user_turn_action():
        nonlocal user_dmg
        u_stats = player.total_stats()
        u_str = u_stats['STR']
        if random.random() < 0.1: u_str = int(u_str * 1.1)

        fatigue_mult = 1.0
        if round_num > 5: fatigue_mult = max(0.1, 1.0 - (round_num - 5) * 0.1)

        dmg_mult = 1.0 * fatigue_mult
        
        # Skill Combo System (Improvement 6)
        if player.skills and random.random() < 0.3:
            skill = random.choice(player.skills)
            dmg_mult *= skill.get('power', 1.0)
            if player.last_skill_id == skill['id']:
                dmg_mult *= 1.25
                logs.append(f"🔥 COMBO! {skill['name']}")
            player.last_skill_id = skill['id']
            
            if skill.get('stun', 0) > 0 and random.random() < skill['stun']:
                mob.stats['SPD'] = 0
                logs.append(f"💫 {mob.name} Stunned!")
        else:
            player.last_skill_id = ""

        # Elemental System (Improvement 1)
        user_element = 'Physical'
        for g in player.gear:
            if g.gear_type == 'MainHand':
                user_element = g.element
        
        e_mult = get_element_mult(user_element, mob.element)
        dmg_mult *= e_mult
        
        # Position Bonus (Improvement 2)
        if player.position == 'Backline':
            dmg_mult *= 1.10

        eff_def = mob.stats['DEF']
        dmg = int((u_str * dmg_mult - eff_def) * intensify)
        min_dmg = int(u_str * 0.15 * intensify)
        dmg = max(min_dmg, dmg)
        
        mob.stats['HP'] -= dmg
        user_dmg += dmg
        
        # Lifesteal check
        lifesteal = 0
        for g in player.gear:
            if g.special == 'Vampiric': lifesteal += 5
        if lifesteal > 0:
            player.current_hp = min(u_stats['HP'], player.current_hp + int(dmg * lifesteal / 100 * heal_penalty))

    def mob_turn_action():
        nonlocal mob_dmg
        if mob.stats['HP'] > 0 and mob.stats['SPD'] > 0:
            m_str = mob.stats['STR']
            dmg_mult = 1.0
            
            # Elemental System (Improvement 1)
            target_element = 'Physical'
            for g in player.gear:
                if g.gear_type == 'Chest':
                    target_element = g.element
            
            e_mult = get_element_mult(mob.element, target_element)
            dmg_mult *= e_mult
            
            # Position Targeting (Improvement 2)
            # Sim is solo, so frontline/backline targeting logic is simplified
            if player.position == 'Frontline':
                dmg_mult *= 0.9 # DEF bonus
            
            if player.position == 'Backline' and mob.element == 'Physical':
                if random.random() < 0.5:
                    logs.append("💨 Evaded!")
                    return

            dmg = int((m_str * dmg_mult - player.total_stats()['DEF']) * intensify)
            min_dmg = int(m_str * 0.15 * intensify)
            dmg = max(min_dmg, dmg)
            
            player.current_hp -= dmg
            mob_dmg += dmg

    if player_starts:
        user_turn_action()
        if mob.stats['HP'] > 0: mob_turn_action()
    else:
        mob_turn_action()
        if player.current_hp > 0: user_turn_action()

    return logs, user_dmg, mob_dmg, max(0, player.current_hp), mob.stats['HP']

def simulate_battle(player, difficulty=1.0, party_size=1):
    max_rounds = 10
    
    # Wave Logic (1-3 waves)
    waves = 1
    if random.random() < 0.2: waves = 2
    if random.random() < 0.05: waves = 3
    
    victory = False
    all_mobs = []
    
    for w in range(1, waves + 1):
        mob = spawn_mob(player.level, difficulty)
        all_mobs.append(mob)
        player_starts = random.random() < 0.5
        
        wave_victory = False
        for rnd in range(1, max_rounds + 1):
            intensify = 1.0 + (rnd - 1) * 0.15
            heal_penalty = 1.0 if rnd <= 5 else max(0, 1.0 - (rnd - 5) * 0.2)
            
            # Status Stacking (Improvement 4) simulation simplified
            # Potion auto-use (Improvement 40)
            if player.current_hp < player.total_stats()['HP'] // 2:
                heal = int(player.total_stats()['HP'] * 0.3)
                player.current_hp = min(player.total_stats()['HP'], player.current_hp + heal)
            
            rlogs, ud, md, ph, mh = resolve_round(player, mob, intensify, heal_penalty, rnd, party_size, player_starts)
            player.current_hp = ph
            mob.stats['HP'] = mh
            
            if mob.stats['HP'] <= 0:
                wave_victory = True
                break
            if player.current_hp <= 0: break
            
        if player.current_hp <= 0: break
        if wave_victory:
            if w == waves: victory = True
            else: continue
            
    return victory, 10, all_mobs, []

def run_combat_cycle(player, difficulty=1.0, party_size=1, system_gold=0):
    battles = 1 + random.randint(0, 2)
    wins = 0
    total_xp = 0
    total_gold = 0

    for _ in range(battles):
        victory, rounds, mobs, logs = simulate_battle(player, difficulty, party_size)
        player.battles_simulated += 1

        if victory:
            wins += 1
            # Economic Inflation (Improvement 44)
            inflation_mult = 1.0
            if system_gold > 10000000:
                inflation_mult = 1.0 / (1.0 + (system_gold - 10000000) / 5000000.0)
            
            for mob in mobs:
                total_xp += mob.reward_xp
                total_gold += int(mob.reward_gold * inflation_mult)
                
                # Salvaging (Improvement 50)
                if random.random() < 0.1: # gear drop chance
                    rarity = random.choices(RARITIES, weights=[60, 25, 10, 4, 0.8, 0.15, 0.05])[0]
                    # if not an upgrade, salvage
                    player.scrap_stack += (1 + RARITIES.index(rarity))
            
            if total_xp > 0:
                # Apply gear XP multipliers to combat rewards
                total_xp = int(total_xp * player.gear_xp_multiplier())
                
                # Dynamic Level Scaling (Improvement 24)
                if player.level > max(m.level for m in mobs) + 20:
                    total_xp = 0
                
            player.add_xp(total_xp)
            player.gold += total_gold
        else:
            penalty = int(player.experience * DEATH_XP_PENALTY)
            player.experience = max(0, player.experience - penalty)
            player.current_hp = player.total_stats()['HP']

    return wins, total_xp, total_gold
