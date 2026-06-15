const fs = require('fs');
const path = require('path');

// List of all game files
const gameFiles = [
    'game_04_frogger.html',
    'game_05_endless_runner.html',
    'game_06_meteor_dodger.html',
    'game_07_3d_tetris.html',
    'game_08_maze_chaser.html',
    'game_09_asteroids.html',
    'game_10_breakout.html',
    'game_11_missile_command.html',
    'game_12_tunnel_flyer.html',
    'game_13_tower_stack.html',
    'game_14_block_dodger.html',
    'game_15_whack_a_mole.html',
    'game_16_air_hockey.html',
    'game_17_tank_battle.html',
    'game_18_helix_drop.html',
    'game_19_galaxy_shooter.html',
    'game_20_cube_runner.html',
    'game_21_gem_collector.html',
    'game_22_laser_defense.html',
    'game_23_star_dodger.html',
    'game_24_color_catch.html',
    'game_25_platform_jumper.html',
    'game_26_lunar_lander.html',
    'game_27_crossy_road.html',
    'game_28_light_cycles.html',
    'game_29_ring_flyer.html',
    'game_30_simon_memory.html',
    'game_31_tft_battler.html'
];

// Function to enhance a game file
function enhanceGameFile(filePath) {
    console.log(`Enhancing ${filePath}...`);
    
    // Read the file
    let content = fs.readFileSync(filePath, 'utf8');
    
    // Add GSAP and animation framework scripts
    content = content.replace(
        '<script src="https://cdnjs.cloudflare.com/ajax/libs/three.js/r128/three.min.js"></script>\n<script src="common/game-framework.js"></script>',
        '<script src="https://cdnjs.cloudflare.com/ajax/libs/three.js/r128/three.min.js"></script>\n<script src="https://cdnjs.cloudflare.com/ajax/libs/gsap/3.12.2/gsap.min.js"></script>\n<script src="common/animation-framework.js"></script>\n<script src="common/game-framework.js"></script>'
    );
    
    // Add enhanced visual effects based on game type
    // This is a simplified approach - in reality, each game would need specific enhancements
    content = addGenericVisualEnhancements(content);
    
    // Write the enhanced file
    fs.writeFileSync(filePath, content);
    console.log(`Enhanced ${filePath}`);
}

// Function to add generic visual enhancements
function addGenericVisualEnhancements(content) {
    // Add basic visual enhancements that work across most games
    
    // Add enhanced materials with emissive properties
    content = content.replace(
        /new THREE\.MeshStandardMaterial\({color:([^}]+)}\)/g,
        'new THREE.MeshStandardMaterial({color:$1, emissive:$1&0x333333, emissiveIntensity: 0.5})'
    );
    
    // Add point lights to spheres for glow effects
    content = content.replace(
        /new THREE\.SphereGeometry\([^)]+\)/g,
        (match) => {
            return `${match}\n// Add point light for glow effect\nconst glowLight = new THREE.PointLight(0xffffff, 1, 5);\nobject.add(glowLight);`;
        }
    );
    
    // Add basic animation framework calls where gameFramework is used
    content = content.replace(
        /(gameFramework\.reportSuccess\(\);?)/g,
        '$1\n        // Visual feedback for success\n        if (gameFramework.animationFramework) {\n            gameFramework.createSparkle(new THREE.Vector3(0, 0, 0), 10);\n        }'
    );
    
    content = content.replace(
        /(gameFramework\.reportFailure\(\);?)/g,
        '$1\n        // Visual feedback for failure\n        if (gameFramework.animationFramework) {\n            gameFramework.applyScreenShake(0.5, 0.2);\n        }'
    );
    
    content = content.replace(
        /(gameFramework\.updateScore\([^)]+\);?)/g,
        '$1\n        // Visual feedback for scoring\n        if (gameFramework.animationFramework) {\n            gameFramework.animationFramework.pulseObject(document.getElementById(\'score\'), 1.2, 0.3);\n        }'
    );
    
    return content;
}

// Process all game files
gameFiles.forEach(file => {
    const filePath = path.join(__dirname, file);
    if (fs.existsSync(filePath)) {
        enhanceGameFile(filePath);
    } else {
        console.log(`File not found: ${filePath}`);
    }
});

console.log('All games enhanced!');