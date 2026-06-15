# 3D Arcade Games Migration Guide

This guide explains how to migrate the remaining 3D arcade games to use the new common game framework with difficulty scaling.

## Overview

The common game framework (`common/game-framework.js`) provides:

1. **Difficulty Scaling System**: Adjusts game parameters based on player performance and time
2. **Standardized Game Mechanics**: Consistent input handling, scoring, and game loops
3. **AI Balancing**: Dynamic adjustment of AI behavior based on difficulty
4. **Progression System**: Score-based unlocks and visual indicators

## Migration Steps

### Step 1: Add Framework Reference

Add the framework script reference after the Three.js script:

```html
<script src="https://cdnjs.cloudflare.com/ajax/libs/three.js/r128/three.min.js"></script>
<script src="common/game-framework.js"></script>
```

### Step 2: Initialize the Framework

In your game initialization function, add:

```javascript
let gameFramework;

function init() {
    // ... existing initialization code ...
    
    // Initialize the game framework
    gameFramework = new GameFramework('game-id'); // Replace 'game-id' with actual game ID
    restart();
}
```

### Step 3: Update Game State Management

Replace direct score management with framework:

```javascript
function restart() {
    // ... reset game objects ...
    
    gameFramework.score = 0; // Use framework score instead of local score variable
    // ... rest of reset code ...
}
```

### Step 4: Integrate Difficulty Scaling

Replace fixed game parameters with difficulty-adjusted ones:

```javascript
// Instead of fixed speed
// let speed = 0.1;

// Use difficulty-adjusted speed
let speed = gameFramework.getAdjustedParams({speed: 0.1}).speed;

// Or use direct scaling
let speed = gameFramework.scaleWithDifficulty(0.1, 1.1);
```

### Step 5: Report Player Actions

Report player successes and failures to adjust difficulty:

```javascript
// When player succeeds at something
gameFramework.reportSuccess();
gameFramework.updateScore(points);

// When player fails
gameFramework.reportFailure();

// When game ends
gameFramework.gameOver();
gameFramework.sendReward();
```

### Step 6: Update Game Loop

Add difficulty updates to your main game loop:

```javascript
function loop() {
    requestAnimationFrame(loop);
    if(alive) {
        update();
        
        // Update difficulty periodically
        gameFramework.updateDifficulty();
    }
    renderer.render(scene, camera);
}
```

## Game-Specific Examples

### Pong-style Games
- Scale CPU paddle speed based on difficulty
- Report player hits as successes, misses as failures
- Adjust ball speed with difficulty

### Shooter Games
- Increase enemy spawn rate/frequency with difficulty
- Boost enemy health/speed based on difficulty
- Report hits on enemies as successes

### Platformer/Avoidance Games
- Increase obstacle spawn rate with difficulty
- Make obstacles move faster based on difficulty
- Report successful dodges as successes

### Strategy/Turn-based Games
- Adjust AI decision-making complexity with difficulty
- Modify resource generation rates based on difficulty
- Report successful strategies as successes

## Game IDs for API

Each game should use the appropriate game ID in the GameFramework constructor:

- Snake: `'snake'`
- Pong: `'pong'`
- Space Invaders: `'invaders'`
- Frogger: `'frogger'`
- Endless Runner: `'runner'`
- Meteor Dodger: `'meteor'`
- Breakout: `'breakout'`
- TFT Battler: `'tft'`
- And so on...

## Testing Checklist

After migrating each game, verify:

1. [ ] Game loads without JavaScript errors
2. [ ] Difficulty indicator appears and updates
3. [ ] Game gets progressively harder
4. [ ] Score reporting works correctly
5. [ ] Game over/reward submission works
6. [ ] Controls remain responsive at higher difficulties
7. [ ] Game remains playable (not impossibly hard)

## Common Patterns

### Input Handling
The framework includes an InputHandler class for standardized controls:

```javascript
let inputHandler = new InputHandler();

// Check for directional input
if(inputHandler.isDirectionPressed('left')) { /* move left */ }

// Check for action input
if(inputHandler.isActionPressed()) { /* fire weapon */ }
```

### Performance Tracking
The framework automatically tracks:
- Player success/failure ratios
- Game duration
- Score progression
- Difficulty level changes

This information is used to dynamically adjust game parameters for optimal challenge.