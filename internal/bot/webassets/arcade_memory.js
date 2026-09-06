/* Free local skill challenge. No wager or reward endpoint is called. */
(() => {
  const board = document.getElementById('memoryBoard');
  const card = board.closest('.game-card');
  const result = card.querySelector('.game-result');
  const pairsLabel = document.getElementById('memoryPairs');
  const movesLabel = document.getElementById('memoryMoves');
  const timeLabel = document.getElementById('memoryTime');
  const bestLabel = document.getElementById('memoryBest');
  const pauseButton = document.getElementById('btn-memory-pause');
  const pauseCover = card.querySelector('.memory-pause-cover');
  const runes = [
    ['potion', 'Health potion'], ['gem', 'Sapphire'], ['ring', 'Ring'], ['rune', 'Rune'],
    ['sword', 'Sword'], ['crown', 'Crown'], ['sun', 'Sun'], ['moon', 'Moon'],
  ];
  let first = null, moves = 0, pairs = 0, started = null, busy = false;
  let ticker, concealTimer, best = null;
  let active = false, paused = false, pausedAt = 0, pausedFor = 0, resumeMessage = '';
  const formatTime = seconds => Math.floor(seconds / 60) + ':' + String(seconds % 60).padStart(2, '0');
  function elapsed() { return started === null ? 0 : Math.floor(((paused ? pausedAt : performance.now()) - started - pausedFor) / 1000); }
  function tick() { timeLabel.textContent = formatTime(elapsed()); }
  function focusTile(tile) {
    tile.focus({ preventScroll: true });
    tile.scrollIntoView({ block: 'nearest', inline: 'nearest' });
  }
  function setPaused(value) {
    if (!active || pairs === 8 || paused === value) return;
    if (value) {
      pausedAt = performance.now();
      resumeMessage = result.textContent;
      clearInterval(ticker);
    } else {
      if (started !== null) pausedFor += performance.now() - pausedAt;
      if (started !== null) ticker = setInterval(tick, 250);
    }
    paused = value;
    board.inert = value;
    board.setAttribute('aria-hidden', String(value));
    pauseCover.hidden = !value;
    pauseButton.textContent = value ? 'Resume challenge' : 'Pause challenge';
    result.textContent = value ? 'Challenge paused. Your time is frozen.' : resumeMessage;
    tick();
    if (!value) focusTile(board.querySelector('.memory-tile:not(.matched)'));
  }
  pauseButton.addEventListener('click', () => setPaused(!paused));
  document.addEventListener('visibilitychange', () => { if (document.hidden) setPaused(true); });
  board.addEventListener('keydown', event => {
    const tile = event.target.closest('.memory-tile');
    if (!tile || paused) return;
    const steps = { ArrowRight: 1, ArrowLeft: -1, ArrowDown: 4, ArrowUp: -4 };
    if (!(event.key in steps)) return;
    event.preventDefault();
    event.stopPropagation();
    const tiles = [...board.children];
    const index = tiles.indexOf(tile);
    for (let distance = 1; distance < tiles.length; distance++) {
      const next = tiles[(index + steps[event.key] * distance + tiles.length * 4) % tiles.length];
      if (!next.classList.contains('matched')) { focusTile(next); break; }
    }
  });
  function showBest() {
    if (best) bestLabel.textContent = 'Best on this browser: ' + best.moves + ' moves · ' + formatTime(best.seconds);
  }
  try {
    const saved = JSON.parse(localStorage.getItem('abyss.arcade.memory.best'));
    if (saved && Number.isSafeInteger(saved.moves) && saved.moves >= 8 &&
        Number.isSafeInteger(saved.seconds) && saved.seconds >= 0) best = saved;
  } catch (_) { /* The challenge still works when browser storage is unavailable. */ }
  showBest();
  function setRevealed(tile, revealed) {
    tile.classList.toggle('revealed', revealed);
    tile.setAttribute('aria-label', 'Tile ' + tile.dataset.position + ': ' + (revealed ? tile.dataset.name : 'hidden rune'));
    tile.setAttribute('aria-pressed', String(revealed));
  }
  function reveal(tile) {
    if (busy || paused || tile.classList.contains('matched') || tile === first) return;
    if (started === null) {
      started = performance.now();
      ticker = setInterval(tick, 250);
    }
    setRevealed(tile, true);
    if (!first) { first = tile; return; }
    movesLabel.textContent = ++moves;
    const previous = first;
    first = null;
    if (previous.dataset.rune === tile.dataset.rune) {
      for (const match of [previous, tile]) {
        match.classList.add('matched');
        match.tabIndex = -1;
        match.setAttribute('aria-disabled', 'true');
        match.setAttribute('aria-label', 'Tile ' + match.dataset.position + ': ' + match.dataset.name + ', matched');
      }
      pairsLabel.textContent = ++pairs + ' / 8';
      result.textContent = 'Pair found: ' + tile.dataset.name + '. ' + pairs + ' of 8 recovered.';
      if (pairs === 8) {
        active = false;
        pauseButton.hidden = true;
        clearInterval(ticker);
        const seconds = elapsed();
        timeLabel.textContent = formatTime(seconds);
        result.textContent = 'Archive restored in ' + moves + ' moves · ' + formatTime(seconds) + '.';
        card.dataset.result = 'win';
        if (!best || moves < best.moves || (moves === best.moves && seconds < best.seconds)) {
          best = { moves, seconds };
          try { localStorage.setItem('abyss.arcade.memory.best', JSON.stringify(best)); } catch (_) {}
          showBest();
          result.textContent += ' New personal best!';
        }
      }
    } else {
      busy = true;
      result.textContent = 'Different relics. Remember their positions.';
      // This viewing interval is part of the game, even with reduced motion.
      concealTimer = setTimeout(() => {
        setRevealed(previous, false); setRevealed(tile, false); busy = false;
      }, 900);
    }
  }
  window.startMemory = function () {
    clearInterval(ticker);
    clearTimeout(concealTimer);
    first = null; moves = 0; pairs = 0; started = null; busy = false;
    active = true; paused = false; pausedAt = 0; pausedFor = 0;
    board.inert = false;
    board.setAttribute('aria-hidden', 'false');
    pauseCover.hidden = true;
    pauseButton.hidden = false;
    pauseButton.textContent = 'Pause challenge';
    pairsLabel.textContent = '0 / 8'; movesLabel.textContent = '0'; timeLabel.textContent = '0:00';
    delete card.dataset.result;
    document.getElementById('btn-memory').textContent = 'Restart challenge';
    result.textContent = 'Reveal two tiles. Find eight pairs to restore the archive.';
    const deck = [...runes, ...runes];
    for (let i = deck.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [deck[i], deck[j]] = [deck[j], deck[i]];
    }
    board.replaceChildren();
    deck.forEach(([rune, name], index) => {
      const tile = document.createElement('button');
      tile.type = 'button';
      tile.className = 'memory-tile ghost';
      tile.dataset.rune = rune; tile.dataset.name = name; tile.dataset.position = index + 1;
      const art = document.createElement('span');
      art.className = 'arcade-art art-' + rune;
      art.setAttribute('aria-hidden', 'true');
      const back = document.createElement('span');
      back.className = 'memory-sigil'; back.textContent = '✦'; back.setAttribute('aria-hidden', 'true');
      tile.append(art, back);
      setRevealed(tile, false);
      tile.addEventListener('click', () => reveal(tile));
      board.appendChild(tile);
    });
    focusTile(board.firstElementChild);
  };
  // An engraved, noninteractive board introduces the challenge before starting.
  for (let i = 0; i < 16; i++) {
    const tile = document.createElement('span');
    tile.className = 'memory-placeholder';
    tile.textContent = '✦';
    tile.setAttribute('aria-hidden', 'true');
    board.appendChild(tile);
  }
})();
