// Common Game Framework for 3D Arcade Games
// Provides standardized mechanics, difficulty scaling, and game loop management
// Integrated with AnimationFramework for enhanced visual effects

class GameFramework {
    constructor(gameId) {
        this.gameId = gameId;
        this.score = 0;
        this.difficulty = 1.0; // Base difficulty starts at 1.0
        this.level = 1;
        this.playerPerformance = 0; // Tracks player success rate
        this.gameStartTime = Date.now();
        this.lastDifficultyUpdate = Date.now();
        
        // Performance tracking
        this.successCount = 0;
        this.attemptCount = 0;
        
        // Initialize difficulty scaling parameters
        this.difficultyParams = {
            baseIncreaseRate: 0.0001,      // How fast difficulty increases naturally
            performanceWeight: 0.0002,     // Weight of performance in difficulty adjustment
            maxDifficulty: 10.0,           // Upper limit for difficulty
            minDifficulty: 0.5,            // Lower limit for difficulty
            updateInterval: 5000,          // Update difficulty every 5 seconds
            performanceWindow: 50          // Track last N actions for performance
        };
        
        // Animation framework instance
        this.animationFramework = new AnimationFramework();
        
        // Common game elements
        this.hudElements = {};
        this.isGameOver = false;
        this.gameActive = false;
        
        this.initHUD();
    }
    
    // Initialize common HUD elements
    initHUD() {
        // Create common HUD elements if they don't exist
        if (!document.getElementById('difficulty-indicator')) {
            const diffIndicator = document.createElement('div');
            diffIndicator.id = 'difficulty-indicator';
            diffIndicator.style.cssText = `
                position: fixed;
                top: 12px;
                right: 12px;
                font-size: 16px;
                z-index: 2;
                text-shadow: 0 1px 3px #000;
                background: rgba(0,0,0,0.5);
                padding: 4px 8px;
                border-radius: 4px;
            `;
            document.body.appendChild(diffIndicator);
        }
        
        this.hudElements.difficulty = document.getElementById('difficulty-indicator');
        this.updateDifficultyDisplay();
    }
    
    // Update difficulty based on player performance and time
    updateDifficulty() {
        const currentTime = Date.now();
        if (currentTime - this.lastDifficultyUpdate < this.difficultyParams.updateInterval) {
            return; // Don't update too frequently
        }
        
        this.lastDifficultyUpdate = currentTime;
        
        // Calculate performance ratio (0.0 to 1.0)
        const performanceRatio = this.attemptCount > 0 ? this.successCount / this.attemptCount : 0.5;
        
        // Adjust difficulty based on performance
        // If player is doing well (performance > 0.7), increase difficulty
        // If player is struggling (performance < 0.3), decrease difficulty
        const targetDifficulty = 1.0 + (performanceRatio - 0.5) * 2 * this.difficultyParams.performanceWeight * 10;
        
        // Gradually move towards target difficulty
        const diffChange = (targetDifficulty - this.difficulty) * 0.05; // Small adjustments
        this.difficulty = Math.max(
            this.difficultyParams.minDifficulty,
            Math.min(this.difficultyParams.maxDifficulty, this.difficulty + diffChange)
        );
        
        // Natural difficulty increase over time
        const timeFactor = (currentTime - this.gameStartTime) / (1000 * 60 * 5); // Every 5 minutes
        this.difficulty = Math.min(this.difficultyParams.maxDifficulty, this.difficulty + (this.difficultyParams.baseIncreaseRate * timeFactor));
        
        this.updateDifficultyDisplay();
    }
    
    // Update the difficulty display
    updateDifficultyDisplay() {
        if (this.hudElements.difficulty) {
            const level = Math.floor(this.difficulty * 10);
            const color = this.getDifficultyColor(level);
            this.hudElements.difficulty.innerHTML = `Level: ${level} <span style="color:${color}">●</span>`;
        }
    }
    
    // Get color based on difficulty level
    getDifficultyColor(level) {
        if (level < 3) return '#4CAF50';    // Green - Easy
        if (level < 6) return '#FFC107';    // Yellow - Medium  
        if (level < 8) return '#FF9800';    // Orange - Hard
        return '#F44336';                   // Red - Very Hard
    }
    
    // Report a successful action to adjust difficulty
    reportSuccess() {
        this.successCount++;
        this.attemptCount++;
        this.updatePerformanceStats();
    }
    
    // Report a failed action to adjust difficulty
    reportFailure() {
        this.attemptCount++;
        this.updatePerformanceStats();
    }
    
    // Update performance statistics
    updatePerformanceStats() {
        // Keep track of recent performance
        this.playerPerformance = this.attemptCount > 0 ? this.successCount / this.attemptCount : 0.5;
    }
    
    // Scale a value based on current difficulty
    scaleWithDifficulty(baseValue, exponent = 1.0) {
        return baseValue * Math.pow(this.difficulty, exponent);
    }
    
    // Get adjusted game parameters based on difficulty
    getAdjustedParams(baseParams) {
        const adjusted = { ...baseParams };
        
        // Adjust various parameters based on difficulty
        if (adjusted.speed !== undefined) {
            adjusted.speed = this.scaleWithDifficulty(adjusted.speed, 1.2);
        }
        
        if (adjusted.frequency !== undefined) {
            adjusted.frequency = this.scaleWithDifficulty(adjusted.frequency, 1.1);
        }
        
        if (adjusted.delay !== undefined) {
            // Inverse relationship - lower delay means faster
            adjusted.delay = adjusted.delay / this.scaleWithDifficulty(1.0, 0.8);
        }
        
        if (adjusted.chance !== undefined) {
            adjusted.chance = Math.min(1.0, this.scaleWithDifficulty(adjusted.chance, 0.7));
        }
        
        return adjusted;
    }
    
    // Standardized game loop with 60 FPS target
    startGameLoop(updateFunction, renderFunction) {
        this.gameActive = true;
        const targetFrameTime = 1000 / 60; // Target 60 FPS
        
        let lastTime = 0;
        const gameLoop = (currentTime) => {
            if (!this.gameActive) return;
            
            const deltaTime = currentTime - lastTime;
            
            if (deltaTime >= targetFrameTime) {
                // Update game state
                if (updateFunction) {
                    updateFunction(deltaTime);
                }
                
                // Render frame
                if (renderFunction) {
                    renderFunction();
                }
                
                lastTime = currentTime - (deltaTime % targetFrameTime);
            }
            
            requestAnimationFrame(gameLoop);
        };
        
        requestAnimationFrame(gameLoop);
    }
    
    // Handle game over
    gameOver() {
        this.gameActive = false;
        this.isGameOver = true;
        
        // Show game over screen with difficulty info
        if (document.getElementById('msg')) {
            const subElement = document.getElementById('sub');
            if (subElement) {
                subElement.innerHTML = `Score: ${this.score}<br/>Difficulty Level: ${Math.floor(this.difficulty * 10)}`;
            }
        }
    }
    
    // Update score and potentially level
    updateScore(points) {
        this.score += points;
        
        // Update level based on score milestones
        const newLevel = Math.floor(this.score / 100) + 1;
        if (newLevel > this.level) {
            this.level = newLevel;
            // Slight difficulty bump when leveling up
            this.difficulty *= 1.05;
            this.difficulty = Math.min(this.difficulty, this.difficultyParams.maxDifficulty);
            
            // Visual feedback for level up
            if (this.animationFramework) {
                this.animationFramework.createSparkle(new THREE.Vector3(0, 0, 0), 30);
            }
        }
        
        // Update score display if it exists
        const scoreElement = document.getElementById('score');
        if (scoreElement) {
            scoreElement.textContent = this.score;
        }
    }
    
    // Send reward to server
    async sendReward() {
        try {
            const response = await fetch('/api/arcade3d/reward', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    game: this.gameId,
                    score: Math.max(0, Math.round(this.score))
                })
            });
            
            const data = await response.json();
            if (data.ok) {
                const rewardElement = document.getElementById('reward');
                if (rewardElement) {
                    rewardElement.innerHTML = `🪙 +${data.gold_won} gold` + 
                        (data.gear ? `<br/>🎁 ${data.gear}` : '');
                }
                
                // Visual feedback for reward
                if (this.animationFramework && rewardElement) {
                    this.animationFramework.pulseObject(rewardElement.parentElement, 1.1, 0.5);
                }
            }
        } catch (e) {
            console.error('Reward submission failed:', e);
        }
    }
    
    // Get current game state
    getGameState() {
        return {
            score: this.score,
            difficulty: this.difficulty,
            level: this.level,
            playerPerformance: this.playerPerformance,
            gameDuration: (Date.now() - this.gameStartTime) / 1000
        };
    }
    
    // Apply screen shake effect
    applyScreenShake(intensity = 1, duration = 0.5) {
        if (this.animationFramework) {
            this.animationFramework.applyScreenShake(intensity, duration);
        }
    }
    
    // Create explosion effect at a position
    createExplosion(position, count = 20, color = 0xff4400) {
        if (this.animationFramework) {
            this.animationFramework.createExplosion(position, count, color);
        }
    }
    
    // Create sparkle effect at a position
    createSparkle(position, count = 10) {
        if (this.animationFramework) {
            this.animationFramework.createSparkle(position, count);
        }
    }
    
    // Add glow effect to an object
    addGlow(object, color = 0xffff00, intensity = 0.5) {
        if (this.animationFramework) {
            this.animationFramework.addGlow(object, color, intensity);
        }
    }
    
    // Remove glow effect from an object
    removeGlow(object) {
        if (this.animationFramework) {
            this.animationFramework.removeGlow(object);
        }
    }
    
    // Pulse an object's scale
    pulseObject(object, scaleMultiplier = 1.2, duration = 0.3) {
        if (this.animationFramework) {
            this.animationFramework.pulseObject(object, scaleMultiplier, duration);
        }
    }
    
    // Fade in an object
    fadeIn(object, duration = 0.5) {
        if (this.animationFramework) {
            this.animationFramework.fadeIn(object, duration);
        }
    }
    
    // Fade out an object
    fadeOut(object, duration = 0.5, onComplete) {
        if (this.animationFramework) {
            this.animationFramework.fadeOut(object, duration, onComplete);
        }
    }
    
    // Slide in an object from a direction
    slideIn(object, direction = 'left', distance = 10, duration = 0.5) {
        if (this.animationFramework) {
            this.animationFramework.slideIn(object, direction, distance, duration);
        }
    }
}

// Input handler for standardized controls
class InputHandler {
    constructor() {
        this.keys = {};
        this.setupEventListeners();
    }
    
    setupEventListeners() {
        document.addEventListener('keydown', (e) => {
            this.keys[e.key.toLowerCase()] = true;
        });
        
        document.addEventListener('keyup', (e) => {
            this.keys[e.key.toLowerCase()] = false;
        });
    }
    
    isPressed(key) {
        return !!this.keys[key.toLowerCase()];
    }
    
    // Check if arrow key or WASD is pressed
    isDirectionPressed(direction) {
        switch(direction.toLowerCase()) {
            case 'up':
                return this.isPressed('arrowup') || this.isPressed('w');
            case 'down':
                return this.isPressed('arrowdown') || this.isPressed('s');
            case 'left':
                return this.isPressed('arrowleft') || this.isPressed('a');
            case 'right':
                return this.isPressed('arrowright') || this.isPressed('d');
            default:
                return false;
        }
    }
    
    // Check for space or enter press
    isActionPressed() {
        return this.isPressed(' ') || this.isPressed('enter');
    }
}

// Export for use in games
window.GameFramework = GameFramework;
window.InputHandler = InputHandler;