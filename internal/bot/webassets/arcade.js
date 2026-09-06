/* Presentation only: wagers and outcomes are always resolved by the server. */
const WHEEL = window.ARCADE_WHEEL;
const SLOT_RELICS = [
  { symbol: '🍒', art: 'potion', name: 'Health potion' },
  { symbol: '🍋', art: 'gem', name: 'Sapphire' },
  { symbol: '🔔', art: 'ring', name: 'Ring' },
  { symbol: '⭐', art: 'rune', name: 'Rune' },
  { symbol: '💎', art: 'sword', name: 'Sword' },
  { symbol: '7️⃣', art: 'crown', name: 'Crown' },
];
const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
const pause = ms => new Promise(resolve => setTimeout(resolve, reducedMotion.matches ? 0 : ms));
const money = value => Number(value).toLocaleString();
let roundBusy = false;
let autoTimer;
const wagerCards = [...document.querySelectorAll('[data-game]:not([data-game="memory"])')];
const choiceNames = { heads: 'Heads', tails: 'Tails', high: 'High', low: 'Low', '1': 'Chest 1', '2': 'Chest 2', '3': 'Chest 3', scout: 'Scout', delve: 'Delve', abyss: 'Abyss' };
for (const card of wagerCards) {
  const preview = document.createElement('p');
  preview.className = 'game-wager';
  const target = card.dataset.game === 'vault' ? card.querySelector('.vault-stage') : card.querySelector('.game-actions');
  target.before(preview);
}
function updateWagerPreviews() {
  const wager = bet();
  const valid = Number.isSafeInteger(wager) && wager >= 1 && wager <= 100000000;
  const risk = document.querySelector('input[name="expeditionRisk"]:checked').value;
  const multipliers = { slots: 88, dice: 2.4, coinflip: 1.95, wheel: Math.max(...WHEEL), highlow: 2, vault: 2.85, expedition: { scout: 1.2, delve: 2, abyss: 4 }[risk] };
  for (const card of wagerCards) {
    const game = card.dataset.game;
    const returned = game === 'coinflip' ? Math.floor(wager * 195 / 100) : game === 'vault' ? Math.floor(wager * 285 / 100) : Math.floor(wager * multipliers[game]);
    card.querySelector('.game-wager').textContent = valid
      ? 'Wager ' + money(wager) + ' gold · ' + (game === 'slots' ? 'Top match returns ' : game === 'wheel' ? 'Up to ' : 'Win returns ') + money(returned) + ' gold' + (game === 'slots' ? ' + jackpot' : '')
      : 'Enter a whole-gold wager between 1 and 100,000,000.';
  }
}
document.getElementById('bet').addEventListener('input', updateWagerPreviews);

function stopAuto() {
  clearTimeout(autoTimer);
  document.getElementById('autoBet').checked = false;
}
document.getElementById('autoBet').addEventListener('change', event => {
  if (!event.target.checked) clearTimeout(autoTimer);
});
window.addEventListener('pagehide', stopAuto);
document.addEventListener('visibilitychange', () => { if (document.hidden) stopAuto(); });
function setBet(value) { document.getElementById('bet').value = Math.max(1, Math.min(100000000, Math.floor(value))); updateWagerPreviews(); }
function mulBet(multiplier) { setBet(Number(document.getElementById('bet').value) * multiplier); }
function bet() { return Number(document.getElementById('bet').value); }
function message(text, tone = '') {
  const node = document.getElementById('arcadeMsg');
  node.textContent = text;
  node.dataset.tone = tone;
}
function updateGold(gold) {
  document.getElementById('goldHere').textContent = money(gold);
  const pill = document.getElementById('goldPill');
  if (pill) pill.textContent = '🪙 ' + money(gold);
}
function lockWagers(locked) {
  roundBusy = locked;
  document.querySelectorAll('.arcade-toolbar button, #bet, [data-game]:not([data-game="memory"]) button, [data-game="expedition"] input, #btn-daily').forEach(button => { button.disabled = locked; });
}
async function arcadeRequest(path, body) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 15000);
  try {
    const response = await fetch(path, {
      method: 'POST', signal: controller.signal,
      headers: { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
    if (!response.ok) throw new Error('Unconfirmed round');
    return await response.json();
  } finally { clearTimeout(timeout); }
}
function requestRound(game, choice, wager) {
  return arcadeRequest('/api/arcade/play', { game, bet: wager, choice: choice || '' });
}
function showResult(card, result) {
  const push = result.net === 0;
  const state = push ? 'push' : result.win ? 'win' : 'loss';
  let text = (card.dataset.choiceLabel ? card.dataset.choiceLabel + ' · ' : '') + result.detail + ' · ';
  if (push) text += 'Bet returned: ' + money(result.payout) + ' gold.';
  else text += 'Returned ' + money(result.payout) + ' gold. Net ' + (result.net > 0 ? '+' : '') + money(result.net) + '.';
  if (result.jackpot_win) text += ' Jackpot claimed!';
  if (result.gear_won) text += ' Loot: ' + result.gear_won + ' → inventory.';
  card.dataset.result = state;
  if (!push) card.classList.add(result.win ? 'win-glow' : 'loss-shake');
  card.querySelector('.game-result').textContent = text;
  message(card.querySelector('h2').textContent + ': ' + text, push ? '' : result.win ? 'good' : 'bad');
  updateGold(result.gold);
  if (Number.isFinite(result.new_jackpot)) document.getElementById('jackpotSlots').textContent = money(result.new_jackpot);
}
function prepareRound(card, game, choice) {
  card.dataset.choiceLabel = choiceNames[choice] ? 'Your choice: ' + choiceNames[choice] : '';
  card.querySelectorAll('.selected, .chosen').forEach(node => node.classList.remove('selected', 'chosen'));
  if (game === 'coinflip' || game === 'highlow') {
    const buttons = game === 'coinflip' ? { heads: 'btn-coin', tails: 'btn-coin-tails' } : { high: 'btn-high', low: 'btn-low' };
    document.getElementById(buttons[choice]).classList.add('chosen');
  }
  if (game === 'slots') card.querySelectorAll('.reel').forEach(node => node.classList.remove('matched'));
  if (game === 'wheel') {
    drawWheel(wheelAngle);
    card.querySelector('.game-hint').textContent = 'The wheel is turning…';
  }
  if (game === 'dice') {
    card.querySelector('.stage-caption').textContent = 'READING THE BONES…';
    document.getElementById('die').classList.add('rolling');
  }
  if (game === 'highlow') {
    document.getElementById('pcard').classList.remove('flip');
    card.querySelector('.pcard-front').setAttribute('aria-hidden', 'false');
    card.querySelector('.pcard-back').setAttribute('aria-hidden', 'true');
  }
  if (game === 'vault') {
    card.querySelectorAll('.treasure-chest').forEach((chest, index) => {
      chest.classList.remove('opened', 'treasure');
      chest.querySelector('.chest-outcome').textContent = 'Sealed';
      chest.setAttribute('aria-label', 'Open chest ' + (index + 1));
      chest.classList.toggle('chosen', index + 1 === Number(choice));
    });
  }
  if (game === 'expedition') {
    card.querySelector('.expedition-marker').hidden = true;
    card.querySelector('.expedition-scene').removeAttribute('data-outcome');
    card.querySelector('.expedition-scene').classList.add('travelling');
  }
}
async function playRound(game, choice, animate) {
  if (roundBusy) return;
  clearTimeout(autoTimer);
  const wager = bet();
  if (!Number.isSafeInteger(wager) || wager < 1 || wager > 100000000) {
    stopAuto();
    message('Enter a whole-gold wager between 1 and 100,000,000.', 'bad');
    document.getElementById('bet').focus();
    return;
  }
  const card = document.querySelector('[data-game="' + game + '"]');
  lockWagers(true);
  card.setAttribute('aria-busy', 'true');
  card.classList.remove('win-glow', 'loss-shake');
  delete card.dataset.result;
  prepareRound(card, game, choice);
  card.querySelector('.game-result').textContent = 'Resolving your round…';
  message('Wager placed: ' + money(wager) + ' gold. Revealing your result…');
  let result;
  try {
    result = await requestRound(game, choice, wager);
    if (!result.ok) {
      stopAuto();
      const text = result.error || 'The round could not be played.';
      message(text, 'bad');
      card.querySelector('.game-result').textContent = text;
      return;
    }
    await animate(result);
    showResult(card, result);
  } catch (_) {
    stopAuto();
    const text = 'Could not confirm the round. Refresh to check your balance before playing again.';
    message(text, 'bad');
    card.querySelector('.game-result').textContent = text;
  } finally {
    card.querySelectorAll('.rolling, .travelling').forEach(node => node.classList.remove('rolling', 'travelling'));
    if (!card.dataset.result) {
      if (game === 'dice') cardCaption('dice', 'READY FOR YOUR NEXT ROLL');
      if (game === 'wheel') card.querySelector('.game-hint').textContent = 'No new wheel result. Check the round message below.';
    }
    card.removeAttribute('aria-busy');
    lockWagers(false);
  }
  if (result?.ok && document.getElementById('autoBet').checked && !document.hidden &&
      ['slots', 'dice', 'wheel'].includes(game) && result.gold >= bet()) {
    autoTimer = setTimeout(() => {
      if (document.getElementById('autoBet').checked && !roundBusy) playRound(game, choice, animate);
    }, 1000);
  } else if (result?.ok && result.gold < bet()) stopAuto();
}
async function playDaily() {
  if (roundBusy) return;
  clearTimeout(autoTimer);
  lockWagers(true);
  try {
    const result = await arcadeRequest('/api/arcade/daily-spin');
    if (result.ok) {
      message(result.reward, 'good');
      updateGold(result.new_gold);
      document.querySelector('.arcade-daily').remove();
    } else message(result.error || 'Daily tribute is unavailable.', 'bad');
  } catch (_) {
    message('Could not confirm the daily spin. Refresh to check your balance.', 'bad');
  } finally { lockWagers(false); }
}
function relicFor(symbol) { return SLOT_RELICS.find(relic => relic.symbol === symbol); }
function symbolNode(relic) {
  const node = document.createElement('div');
  node.className = 'sym';
  node.setAttribute('aria-hidden', 'true');
  const art = document.createElement('span');
  art.className = 'arcade-art art-' + (relic?.art || 'oracle');
  node.appendChild(art);
  return node;
}
function buildStrip(index, result) {
  const strip = document.getElementById('strip' + index);
  strip.replaceChildren();
  const length = reducedMotion.matches ? 0 : 18 + index * 3;
  for (let i = 0; i < length; i++) strip.appendChild(symbolNode(SLOT_RELICS[Math.floor(Math.random() * SLOT_RELICS.length)]));
  strip.appendChild(symbolNode(result));
  strip.style.transition = 'none';
  strip.style.transform = 'translateY(0)';
  return strip;
}
function playSlots() {
  return playRound('slots', '', async result => {
    const counts = {};
    result.symbols.forEach(symbol => { counts[symbol] = (counts[symbol] || 0) + 1; });
    for (let i = 0; i < 5; i++) {
      const strip = buildStrip(i, relicFor(result.symbols[i]));
      strip.parentElement.classList.remove('matched');
      const height = strip.firstElementChild.getBoundingClientRect().height;
      void strip.offsetHeight;
      strip.style.transition = reducedMotion.matches ? 'none' : 'transform ' + (1.1 + i * .22) + 's cubic-bezier(.15,.65,.18,1)';
      strip.style.transform = 'translateY(-' + ((strip.children.length - 1) * height) + 'px)';
    }
    await pause(2050);
    result.symbols.forEach((symbol, index) => {
      document.getElementById('strip' + index).parentElement.classList.toggle('matched', counts[symbol] >= 3);
    });
    const best = Math.max(...Object.values(counts));
    document.querySelectorAll('.relic-paytable > span').forEach((node, index) => node.classList.toggle('selected', best >= 3 && index === best - 3));
    document.getElementById('reels').setAttribute('aria-label', result.symbols.map(symbol => relicFor(symbol)?.name || symbol).join(', '));
  });
}
function playDice() {
  return playRound('dice', '', async result => {
    const die = document.getElementById('die');
    await pause(800);
    die.classList.remove('rolling');
    document.getElementById('diePips').dataset.roll = result.roll;
    die.setAttribute('aria-label', 'Die: ' + result.roll);
    cardCaption('dice', 'ROLLED ' + result.roll + (result.roll === 4 ? ' · BET RETURNED' : result.roll > 4 ? ' · WIN' : ' · LOSS'));
    document.querySelectorAll('.dice-guide > span').forEach((node, index) => {
      node.classList.toggle('selected', index === (result.roll < 4 ? 0 : result.roll === 4 ? 1 : 2));
    });
  });
}
function cardCaption(game, text) { document.querySelector('[data-game="' + game + '"] .stage-caption').textContent = text; }
function playCoin(side) {
  return playRound('coinflip', side, async result => {
    const coin = document.getElementById('coin');
    const angle = result.side === 'tails' ? 180 : 0;
    coin.style.transition = reducedMotion.matches ? 'none' : 'transform 1.2s cubic-bezier(.2,.75,.25,1)';
    coin.style.transform = 'rotateY(' + (1800 + angle) + 'deg)';
    await pause(1250);
    coin.style.transition = 'none';
    coin.style.transform = 'rotateY(' + angle + 'deg)';
    coin.setAttribute('aria-label', 'Coin: ' + result.side);
    cardCaption('coinflip', 'CALLED ' + side.toUpperCase() + ' · LANDED ' + result.side.toUpperCase());
  });
}
const wheelCanvas = document.getElementById('wheel');
const wheelContext = wheelCanvas.getContext('2d');
let wheelAngle = 0;
function drawWheel(angle, selected = -1) {
  const ctx = wheelContext;
  ctx.setTransform(2, 0, 0, 2, 0, 0);
  ctx.clearRect(0, 0, 220, 220);
  ctx.save();
  ctx.translate(110, 110);
  const arc = Math.PI * 2 / WHEEL.length;
  WHEEL.forEach((multiplier, index) => {
    const start = angle + index * arc;
    ctx.beginPath(); ctx.moveTo(0, 0); ctx.arc(0, 0, 99, start, start + arc); ctx.closePath();
    ctx.fillStyle = multiplier >= 5 ? '#866536' : multiplier >= 3 ? '#514568' : multiplier >= 2 ? '#235e59' : multiplier > 0 ? '#27465e' : index % 2 ? '#121d2c' : '#1b2838';
    ctx.fill();
    ctx.strokeStyle = index === selected ? '#ffdea0' : '#85714b';
    ctx.lineWidth = index === selected ? 3 : .8; ctx.stroke();
    ctx.save(); ctx.rotate(start + arc / 2);
    ctx.textAlign = 'right'; ctx.fillStyle = multiplier ? '#fff0d0' : '#97a9bf'; ctx.font = 'bold 12px Consolas, monospace';
    ctx.fillText(multiplier ? '×' + multiplier : '—', 88, 4);
    ctx.restore();
  });
  for (const radius of [103, 106, 25]) {
    ctx.beginPath(); ctx.arc(0, 0, radius, 0, Math.PI * 2);
    if (radius === 25) { ctx.fillStyle = '#0b131f'; ctx.fill(); }
    ctx.strokeStyle = '#b5975f'; ctx.lineWidth = radius === 106 ? 2 : 1; ctx.stroke();
  }
  for (let i = 0; i < 12; i++) {
    const a = angle + i * arc;
    ctx.beginPath(); ctx.arc(Math.cos(a) * 103, Math.sin(a) * 103, 1.5, 0, Math.PI * 2);
    ctx.fillStyle = '#ffe0a0'; ctx.fill();
  }
  ctx.restore();
}
drawWheel(0);
function playWheel() {
  return playRound('wheel', '', async result => {
    const tau = Math.PI * 2;
    const target = -Math.PI / 2 - (result.segment + .5) * tau / WHEEL.length;
    const start = wheelAngle;
    const delta = ((target - start) % tau + tau) % tau;
    const end = start + 4 * tau + delta;
    if (!reducedMotion.matches) {
      const began = performance.now();
      await new Promise(resolve => {
        function frame(now) {
          const progress = Math.min(1, (now - began) / 2300);
          drawWheel(start + (end - start) * (1 - Math.pow(1 - progress, 3)));
          if (progress < 1) requestAnimationFrame(frame); else resolve();
        }
        requestAnimationFrame(frame);
      });
    }
    wheelAngle = target;
    drawWheel(target, result.segment);
    wheelCanvas.setAttribute('aria-label', 'Wheel result: ' + (WHEEL[result.segment] ? '×' + WHEEL[result.segment] : 'blank'));
    document.querySelector('#game-wheel .game-hint').textContent = WHEEL[result.segment]
      ? 'Landed ×' + WHEEL[result.segment] + ' · ' + money(result.payout) + ' gold returned'
      : 'Landed on a blank · No wheel payout';
  });
}
function playHL(choice) {
  return playRound('highlow', choice, async result => {
    const card = document.getElementById('pcard');
    await pause(650);
    document.getElementById('cardVal').textContent = result.card;
    card.querySelector('.pcard-front').setAttribute('aria-hidden', 'true');
    card.querySelector('.pcard-back').setAttribute('aria-hidden', 'false');
    card.classList.add('flip');
    document.querySelectorAll('.oracle-range > span').forEach((node, index) => node.classList.toggle('selected', index === (result.card < 7 ? 0 : result.card === 7 ? 1 : 2)));
    await pause(650);
  });
}
for (let i = 0; i < 5; i++) document.getElementById('strip' + i).appendChild(symbolNode(SLOT_RELICS[i]));

function playVault(choice) {
  return playRound('vault', choice, async result => {
    const chests = [...document.querySelectorAll('.treasure-chest')];
    await pause(600);
    chests[result.chest - 1].classList.add('treasure');
    chests.forEach((chest, index) => {
      chest.classList.add('opened');
      chest.querySelector('.chest-outcome').textContent = index + 1 === result.chest ? 'Sapphire!' : 'Empty';
      chest.setAttribute('aria-label', 'Open chest ' + (index + 1) + ' (last round: ' + (index + 1 === result.chest ? 'Sapphire!' : 'Empty') + ')');
    });
    await pause(650);
  });
}
const expeditionRoutes = {
  scout: { chance: 80, depth: 'THE OUTSKIRTS' },
  delve: { chance: 48, depth: 'THE SUNKEN HALLS' },
  abyss: { chance: 24, depth: 'THE ABYSS' },
};
function updateExpedition() {
  const choice = document.querySelector('input[name="expeditionRisk"]:checked').value;
  const route = expeditionRoutes[choice];
  const meter = document.querySelector('.expedition-meter');
  meter.style.setProperty('--chance', route.chance + '%');
  meter.setAttribute('aria-label', choice + ': survive on rolls 1 through ' + route.chance);
  meter.querySelector('.expedition-marker').hidden = true;
  document.getElementById('expeditionDepth').textContent = route.depth;
  document.querySelector('.expedition-scene').dataset.route = choice;
  document.getElementById('expeditionHint').textContent = 'Survive on 1–' + route.chance + '. Rolls ' + (route.chance + 1) + '–100 lose the wager.';
  updateWagerPreviews();
}
document.querySelectorAll('input[name="expeditionRisk"]').forEach(input => input.addEventListener('change', updateExpedition));
function playExpedition() {
  const choice = document.querySelector('input[name="expeditionRisk"]:checked').value;
  return playRound('expedition', choice, async result => {
    const scene = document.querySelector('.expedition-scene');
    await pause(1300);
    scene.classList.remove('travelling');
    scene.dataset.outcome = result.roll <= result.chance ? 'survived' : 'lost';
    const marker = document.querySelector('.expedition-marker');
    marker.style.left = result.roll + '%';
    marker.hidden = false;
    document.querySelector('.expedition-meter').setAttribute('aria-label', 'Rolled ' + result.roll + '; survival threshold ' + result.chance);
    document.getElementById('expeditionHint').textContent = 'Rolled ' + result.roll + ' / 100 · Survival threshold: ' + result.chance;
  });
}
updateWagerPreviews();
