// Advanced Animation Framework for 3D Arcade Games
// Provides smooth animations, visual effects, and performance optimizations

class AnimationFramework {
    constructor(scene, renderer) {
        // Store scene and renderer references for adding particles and screen shake
        this.scene = scene || null;
        this.renderer = renderer || null;

        // Store camera reference for screen shake (extracted from scene or set later)
        this.camera = null;
        this.originalCameraPosition = null;

        // Particle system pools for performance
        this.particlePools = {};
        this.activeParticles = [];

        // Animation queues for sequential effects
        this.animationQueues = {};

        // Screen shake effect variables
        this.shakeIntensity = 0;
        this.shakeDuration = 0;
        this.shakeStartTime = 0;

        // Object pooling for frequently created/destroyed objects
        this.objectPools = {};

        // Frustum culling cache
        this.frustum = new THREE.Frustum();
        this.cameraMatrix = new THREE.Matrix4();

        // Initialize GSAP if available
        this.hasGSAP = typeof gsap !== 'undefined';

        // Setup animation ticker
        this.setupAnimationTicker();
    }

    // Set the camera reference for screen shake effects.
    // Call this after creating the AnimationFramework if camera is not available at construction time.
    setCamera(camera) {
        this.camera = camera;
        if (camera) {
            this.originalCameraPosition = camera.position.clone();
        }
    }

    // Setup animation ticker for smooth updates
    setupAnimationTicker() {
        this.tickCallbacks = [];
        this.lastTickTime = performance.now();

        const tick = (time) => {
            const deltaTime = Math.min(100, time - this.lastTickTime) / 1000; // Cap at 100ms
            this.lastTickTime = time;

            // Update screen shake
            this.updateScreenShake(deltaTime);

            // Update particles
            this.updateParticles(deltaTime);

            // Run tick callbacks
            for (const callback of this.tickCallbacks) {
                callback(deltaTime);
            }

            requestAnimationFrame(tick);
        };

        requestAnimationFrame(tick);
    }

    // Register a callback to be called every animation frame
    onTick(callback) {
        this.tickCallbacks.push(callback);
    }

    // Remove a tick callback
    offTick(callback) {
        const index = this.tickCallbacks.indexOf(callback);
        if (index !== -1) {
            this.tickCallbacks.splice(index, 1);
        }
    }

    // ====================
    // EASING FUNCTIONS
    // ====================

    // Smooth easing functions using GSAP or fallback implementations
    easeInOut(t) {
        if (this.hasGSAP) {
            return gsap.parseEase("power2.inOut")(t);
        }
        return t < 0.5 ? 2 * t * t : -1 + (4 - 2 * t) * t;
    }

    easeBounce(t) {
        if (this.hasGSAP) {
            return gsap.parseEase("bounce.out")(t);
        }
        if (t < 1 / 2.75) {
            return 7.5625 * t * t;
        } else if (t < 2 / 2.75) {
            return 7.5625 * (t -= 1.5 / 2.75) * t + 0.75;
        } else if (t < 2.5 / 2.75) {
            return 7.5625 * (t -= 2.25 / 2.75) * t + 0.9375;
        } else {
            return 7.5625 * (t -= 2.625 / 2.75) * t + 0.984375;
        }
    }

    easeElastic(t) {
        if (this.hasGSAP) {
            return gsap.parseEase("elastic.out")(t);
        }
        if (t === 0 || t === 1) return t;
        return Math.pow(2, -10 * t) * Math.sin((t * 10 - 0.75) * (2 * Math.PI) / 3) + 1;
    }

    // ====================
    // PARTICLE SYSTEMS
    // ====================

    // Create a particle pool for reuse
    createParticlePool(type, size = 100) {
        if (!this.particlePools[type]) {
            this.particlePools[type] = [];
        }

        for (let i = 0; i < size; i++) {
            let particle;
            switch (type) {
                case 'explosion':
                    particle = new THREE.Mesh(
                        new THREE.SphereGeometry(0.1, 8, 8),
                        new THREE.MeshBasicMaterial({ color: 0xff4400 })
                    );
                    break;
                case 'sparkle':
                    particle = new THREE.Mesh(
                        new THREE.SphereGeometry(0.05, 6, 6),
                        new THREE.MeshBasicMaterial({ color: 0xffff00 })
                    );
                    break;
                case 'smoke':
                    particle = new THREE.Mesh(
                        new THREE.PlaneGeometry(0.3, 0.3),
                        new THREE.MeshBasicMaterial({
                            color: 0xaaaaaa,
                            transparent: true,
                            opacity: 0.7
                        })
                    );
                    break;
                default:
                    particle = new THREE.Mesh(
                        new THREE.SphereGeometry(0.1, 8, 8),
                        new THREE.MeshBasicMaterial({ color: 0xffffff })
                    );
            }

            particle.visible = false;
            particle.userData = {
                active: false,
                velocity: new THREE.Vector3(),
                life: 0,
                maxLife: 1,
                type: type
            };

            this.particlePools[type].push(particle);
        }
    }

    // Get an inactive particle from the pool
    getParticle(type) {
        if (!this.particlePools[type]) {
            this.createParticlePool(type, 50);
        }

        const pool = this.particlePools[type];
        for (let i = 0; i < pool.length; i++) {
            if (!pool[i].userData.active) {
                pool[i].userData.active = true;
                pool[i].visible = true;
                // Add particle to scene if not already added
                if (this.scene && !pool[i].parent) {
                    this.scene.add(pool[i]);
                }
                return pool[i];
            }
        }

        // If no inactive particles, create a new one
        const particle = this.createSingleParticle(type);
        particle.userData.active = true;
        particle.visible = true;
        pool.push(particle);
        // Add particle to scene
        if (this.scene) {
            this.scene.add(particle);
        }
        return particle;
    }

    // Create a single particle of specified type
    createSingleParticle(type) {
        let particle;
        switch (type) {
            case 'explosion':
                particle = new THREE.Mesh(
                    new THREE.SphereGeometry(0.1, 8, 8),
                    new THREE.MeshBasicMaterial({ color: 0xff4400 })
                );
                break;
            case 'sparkle':
                particle = new THREE.Mesh(
                    new THREE.SphereGeometry(0.05, 6, 6),
                    new THREE.MeshBasicMaterial({ color: 0xffff00 })
                );
                break;
            case 'smoke':
                particle = new THREE.Mesh(
                    new THREE.PlaneGeometry(0.3, 0.3),
                    new THREE.MeshBasicMaterial({
                        color: 0xaaaaaa,
                        transparent: true,
                        opacity: 0.7
                    })
                );
                break;
            default:
                particle = new THREE.Mesh(
                    new THREE.SphereGeometry(0.1, 8, 8),
                    new THREE.MeshBasicMaterial({ color: 0xffffff })
                );
        }

        particle.visible = false;
        particle.userData = {
            active: false,
            velocity: new THREE.Vector3(),
            life: 0,
            maxLife: 1,
            type: type
        };

        return particle;
    }

    // Return a particle to the pool
    releaseParticle(particle) {
        if (particle && particle.userData) {
            particle.userData.active = false;
            particle.visible = false;
            particle.position.set(0, 0, 0);
            particle.userData.velocity.set(0, 0, 0);
            particle.userData.life = 0;
        }
    }

    // Update all active particles
    updateParticles(deltaTime) {
        for (const type in this.particlePools) {
            const pool = this.particlePools[type];
            for (let i = pool.length - 1; i >= 0; i--) {
                const particle = pool[i];
                if (particle.userData.active) {
                    this.updateParticle(particle, deltaTime);
                }
            }
        }
    }

    // Update a single particle
    updateParticle(particle, deltaTime) {
        particle.userData.life += deltaTime;

        if (particle.userData.life >= particle.userData.maxLife) {
            this.releaseParticle(particle);
            return;
        }

        // Update position based on velocity
        particle.position.add(
            particle.userData.velocity.clone().multiplyScalar(deltaTime)
        );

        // Apply gravity to some particles
        if (particle.userData.type === 'explosion' || particle.userData.type === 'smoke') {
            particle.userData.velocity.y -= 9.8 * deltaTime;
        }

        // Fade out particles
        if (particle.material.transparent) {
            const lifeRatio = particle.userData.life / particle.userData.maxLife;
            particle.material.opacity = 1 - lifeRatio;
        }

        // Change color over time for explosions
        if (particle.userData.type === 'explosion' && particle.material.color) {
            const lifeRatio = particle.userData.life / particle.userData.maxLife;
            if (lifeRatio > 0.5) {
                particle.material.color.setHex(0xff8800); // Orange
            }
            if (lifeRatio > 0.8) {
                particle.material.color.setHex(0xffdd00); // Yellow
            }
        }
    }

    // Create an explosion effect at a position
    createExplosion(position, count = 20, color = 0xff4400) {
        for (let i = 0; i < count; i++) {
            const particle = this.getParticle('explosion');
            if (particle) {
                particle.position.copy(position);
                particle.material.color.setHex(color);

                // Random velocity
                const angle = Math.random() * Math.PI * 2;
                const speed = 2 + Math.random() * 3;
                particle.userData.velocity.set(
                    Math.cos(angle) * speed,
                    Math.random() * 2,
                    Math.sin(angle) * speed
                );

                particle.userData.maxLife = 0.5 + Math.random() * 0.5;
                particle.userData.life = 0;
            }
        }
    }

    // Create a sparkle effect at a position
    createSparkle(position, count = 10) {
        for (let i = 0; i < count; i++) {
            const particle = this.getParticle('sparkle');
            if (particle) {
                particle.position.copy(position);

                // Random velocity upward
                particle.userData.velocity.set(
                    (Math.random() - 0.5) * 2,
                    Math.random() * 3,
                    (Math.random() - 0.5) * 2
                );

                particle.userData.maxLife = 0.3 + Math.random() * 0.2;
                particle.userData.life = 0;
            }
        }
    }

    // ====================
    // SCREEN EFFECTS
    // ====================

    // Apply screen shake effect
    applyScreenShake(intensity = 1, duration = 0.5) {
        this.shakeIntensity = intensity;
        this.shakeDuration = duration;
        this.shakeStartTime = performance.now();
    }

    // Update screen shake effect
    updateScreenShake(deltaTime) {
        if (this.shakeIntensity <= 0) return;

        const elapsed = (performance.now() - this.shakeStartTime) / 1000;
        const progress = Math.min(1, elapsed / this.shakeDuration);

        if (progress >= 1) {
            this.shakeIntensity = 0;
            // Reset camera position
            if (this.camera && this.originalCameraPosition) {
                this.camera.position.x = this.originalCameraPosition.x;
                this.camera.position.y = this.originalCameraPosition.y;
                this.camera.position.z = this.originalCameraPosition.z;
            }
            return;
        }

        // Apply shake effect to camera
        if (this.camera) {
            // Store original position if not already stored
            if (!this.originalCameraPosition) {
                this.originalCameraPosition = this.camera.position.clone();
            }

            const shakeAmount = this.shakeIntensity * (1 - progress);
            const offsetX = (Math.random() - 0.5) * shakeAmount * 2;
            const offsetY = (Math.random() - 0.5) * shakeAmount * 2;

            this.camera.position.x = this.originalCameraPosition.x + offsetX;
            this.camera.position.y = this.originalCameraPosition.y + offsetY;
        }
    }

    // ====================
    // VISUAL EFFECTS
    // ====================

    // Add glow effect to an object
    addGlow(object, color = 0xffff00, intensity = 0.5) {
        if (!object) return;

        // Store original material
        if (!object.userData.originalMaterial) {
            object.userData.originalMaterial = object.material;
        }

        // Create glowing material
        const glowMaterial = new THREE.MeshBasicMaterial({
            color: color,
            transparent: true,
            opacity: intensity,
            blending: THREE.AdditiveBlending
        });

        object.material = glowMaterial;

        // Store glow data for removal
        object.userData.glow = {
            material: glowMaterial,
            color: color,
            intensity: intensity
        };
    }

    // Remove glow effect from an object
    removeGlow(object) {
        if (!object || !object.userData.glow) return;

        object.material = object.userData.originalMaterial;
        delete object.userData.glow;
    }

    // Pulse an object's scale
    pulseObject(object, scaleMultiplier = 1.2, duration = 0.3) {
        if (!object) return;

        if (this.hasGSAP) {
            // Use GSAP for smoother animation
            const originalScale = object.scale.clone();
            gsap.to(object.scale, {
                x: originalScale.x * scaleMultiplier,
                y: originalScale.y * scaleMultiplier,
                z: originalScale.z * scaleMultiplier,
                duration: duration / 2,
                yoyo: true,
                repeat: 1,
                ease: "power1.inOut"
            });
        } else {
            // Fallback animation
            const originalScale = object.scale.clone();
            const startTime = performance.now();
            const halfDuration = duration * 500; // Convert to ms

            const animate = (time) => {
                const elapsed = time - startTime;
                if (elapsed < duration * 1000) {
                    const progress = Math.min(1, elapsed / halfDuration);
                    const scale = 1 + (scaleMultiplier - 1) *
                        (progress <= 1 ? progress : 2 - progress);

                    object.scale.set(
                        originalScale.x * scale,
                        originalScale.y * scale,
                        originalScale.z * scale
                    );

                    requestAnimationFrame(animate);
                } else {
                    object.scale.copy(originalScale);
                }
            };

            requestAnimationFrame(animate);
        }
    }

    // ====================
    // TRANSITIONS
    // ====================

    // Fade in an object
    fadeIn(object, duration = 0.5) {
        if (!object) return;

        if (object.material) {
            object.material.transparent = true;
            object.material.opacity = 0;
            object.visible = true;

            if (this.hasGSAP) {
                gsap.to(object.material, {
                    opacity: 1,
                    duration: duration,
                    ease: "power2.out"
                });
            } else {
                // Simple fade without GSAP
                const startTime = performance.now();
                const animate = (time) => {
                    const elapsed = time - startTime;
                    if (elapsed < duration * 1000) {
                        const progress = elapsed / (duration * 1000);
                        object.material.opacity = progress;
                        requestAnimationFrame(animate);
                    } else {
                        object.material.opacity = 1;
                    }
                };
                requestAnimationFrame(animate);
            }
        }
    }

    // Fade out an object
    fadeOut(object, duration = 0.5, onComplete) {
        if (!object) return;

        if (object.material) {
            object.material.transparent = true;

            if (this.hasGSAP) {
                gsap.to(object.material, {
                    opacity: 0,
                    duration: duration,
                    ease: "power2.in",
                    onComplete: () => {
                        object.visible = false;
                        if (onComplete) onComplete();
                    }
                });
            } else {
                // Simple fade without GSAP
                const startTime = performance.now();
                const animate = (time) => {
                    const elapsed = time - startTime;
                    if (elapsed < duration * 1000) {
                        const progress = elapsed / (duration * 1000);
                        object.material.opacity = 1 - progress;
                        requestAnimationFrame(animate);
                    } else {
                        object.material.opacity = 0;
                        object.visible = false;
                        if (onComplete) onComplete();
                    }
                };
                requestAnimationFrame(animate);
            }
        }
    }

    // Slide in an object from a direction
    slideIn(object, direction = 'left', distance = 10, duration = 0.5) {
        if (!object) return;

        // Store original position
        if (!object.userData.originalPosition) {
            object.userData.originalPosition = object.position.clone();
        }

        // Set starting position
        const startPos = object.userData.originalPosition.clone();
        switch (direction) {
            case 'left':
                startPos.x -= distance;
                break;
            case 'right':
                startPos.x += distance;
                break;
            case 'top':
                startPos.y += distance;
                break;
            case 'bottom':
                startPos.y -= distance;
                break;
        }

        object.position.copy(startPos);
        object.visible = true;

        if (this.hasGSAP) {
            gsap.to(object.position, {
                x: object.userData.originalPosition.x,
                y: object.userData.originalPosition.y,
                z: object.userData.originalPosition.z,
                duration: duration,
                ease: "back.out(1.7)"
            });
        } else {
            // Simple slide without GSAP
            const startTime = performance.now();
            const animate = (time) => {
                const elapsed = time - startTime;
                if (elapsed < duration * 1000) {
                    const progress = this.easeBounce(elapsed / (duration * 1000));
                    object.position.lerpVectors(
                        startPos,
                        object.userData.originalPosition,
                        progress
                    );
                    requestAnimationFrame(animate);
                } else {
                    object.position.copy(object.userData.originalPosition);
                }
            };
            requestAnimationFrame(animate);
        }
    }

    // ====================
    // PERFORMANCE OPTIMIZATIONS
    // ====================

    // Create an object pool
    createObjectPool(createFn, resetFn, size = 20) {
        const pool = {
            objects: [],
            createFn: createFn,
            resetFn: resetFn
        };

        for (let i = 0; i < size; i++) {
            const obj = createFn();
            obj.userData = obj.userData || {};
            obj.userData.pooled = true;
            obj.userData.active = false;
            pool.objects.push(obj);
        }

        return pool;
    }

    // Get an object from a pool
    getObjectFromPool(pool) {
        for (const obj of pool.objects) {
            if (!obj.userData.active) {
                obj.userData.active = true;
                if (pool.resetFn) {
                    pool.resetFn(obj);
                }
                return obj;
            }
        }

        // No inactive objects, create a new one
        const newObj = pool.createFn();
        newObj.userData = newObj.userData || {};
        newObj.userData.pooled = true;
        newObj.userData.active = true;
        pool.objects.push(newObj);

        if (pool.resetFn) {
            pool.resetFn(newObj);
        }

        return newObj;
    }

    // Return an object to its pool
    returnObjectToPool(object) {
        if (object.userData && object.userData.pooled) {
            object.userData.active = false;
            object.visible = false;
        }
    }

    // Update frustum for culling
    updateFrustum(camera) {
        this.cameraMatrix.multiplyMatrices(camera.projectionMatrix, camera.matrixWorldInverse);
        this.frustum.setFromProjectionMatrix(this.cameraMatrix);
    }

    // Check if an object is in the camera frustum
    isInFrustum(object) {
        return this.frustum.intersectsObject(object);
    }

    // Apply LOD (Level of Detail) to an object
    applyLOD(object, distances) {
        // Store LOD data
        object.userData.lod = {
            distances: distances,
            originalGeometry: object.geometry
        };
    }

    // Update LOD based on camera distance
    updateLOD(object, camera) {
        if (!object.userData.lod) return;

        const distance = object.position.distanceTo(camera.position);
        const lodData = object.userData.lod;

        // Find appropriate LOD level
        for (let i = 0; i < lodData.distances.length; i++) {
            if (distance < lodData.distances[i].distance) {
                if (object.geometry !== lodData.distances[i].geometry) {
                    object.geometry = lodData.distances[i].geometry;
                }
                return;
            }
        }

        // Use original geometry if no LOD matches
        if (object.geometry !== lodData.originalGeometry) {
            object.geometry = lodData.originalGeometry;
        }
    }
}

// Export for use in games
window.AnimationFramework = AnimationFramework;