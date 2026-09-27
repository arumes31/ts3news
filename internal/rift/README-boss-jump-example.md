# Animated boss-jump teaching example

Boss phase practice includes an expandable timing example with Play, Next still
frame and Preview slam sound controls. The reference illustrates a standard
1.15-second ground slam, jumping at 0.9 seconds with the existing 0.65-second jump
arc and 52-unit visual lift. It shows warning, jump, airborne impact and landing.
The explanation explicitly says real timings vary by boss and phase and that
moving outside the marked area is another valid response.

Artwork uses the player's class sprite and the shared Abyss boss-frame provider.
The jump prompt uses the current control binding. Text descriptions accompany the
canvas and announce stage changes. Reduced motion or zero motion intensity uses
still frames; users can step through all four. The example fits narrow screens.

Starting any preview pauses an active drill. Animation stays on a separate canvas
and does not advance simulation, consume resources or grant rewards. It stops on
closing the example, hiding/leaving the page or starting the drill; sound previews
are cancelled too. The separate slam-sound button respects the existing mix/mute
settings and reports unavailable or zero-volume audio.

Browser tests verify all animated stages, still-frame progression, jump height at
impact, current key text, no state-changing requests from idle preview playback,
close cancellation, audio cleanup, mobile width and pausing a live boss drill.
The final impact screenshot was inspected for complete, readable sprite artwork.
This implements improvement 0674.
