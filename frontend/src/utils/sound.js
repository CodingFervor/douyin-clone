// ===================== Feature: Sound effects (操作音效反馈) =====================
// A tiny Web Audio API helper that synthesizes short UI tones on the fly — no
// audio files required. A lazily-created AudioContext is reused across calls.
// Browsers only allow AudioContext after a user gesture, so it is created on
// the first playBeep call (which is always triggered by a tap/keypress).

let audioCtx = null

// getCtx lazily creates and resumes the shared AudioContext. We resume() on
// each use because Safari/Chrome can suspend the context after inactivity.
function getCtx() {
  if (typeof window === 'undefined') return null
  if (!audioCtx) {
    const AC = window.AudioContext || window.webkitAudioContext
    if (!AC) return null
    try {
      audioCtx = new AC()
    } catch (e) {
      return null
    }
  }
  if (audioCtx.state === 'suspended') {
    audioCtx.resume().catch(() => {})
  }
  return audioCtx
}

// playBeep synthesizes a single oscillator tone at `frequency` (Hz) for
// `duration` milliseconds, ramped in/out to avoid clicks. Returns immediately.
export function playBeep(frequency, duration = 100) {
  const ctx = getCtx()
  if (!ctx) return
  const now = ctx.currentTime
  const dur = Math.max(10, duration) / 1000

  const osc = ctx.createOscillator()
  const gain = ctx.createGain()
  osc.type = 'sine'
  osc.frequency.setValueAtTime(frequency, now)

  // Envelope: quick attack, gentle release — keeps the tone subtle.
  const peak = 0.18
  gain.gain.setValueAtTime(0.0001, now)
  gain.gain.exponentialRampToValueAtTime(peak, now + 0.01)
  gain.gain.exponentialRampToValueAtTime(0.0001, now + dur)

  osc.connect(gain)
  gain.connect(ctx.destination)
  osc.start(now)
  osc.stop(now + dur + 0.02)
}

// playSequence plays a list of [{frequency, duration}] tones back-to-back,
// scheduling each one at the cumulative offset so they play seamlessly.
function playSequence(notes) {
  const ctx = getCtx()
  if (!ctx) return
  const now = ctx.currentTime
  let offset = 0
  for (const note of notes) {
    const freq = note.frequency
    const dur = Math.max(10, note.duration) / 1000
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()
    osc.type = 'sine'
    osc.frequency.setValueAtTime(freq, now + offset)

    const peak = 0.18
    gain.gain.setValueAtTime(0.0001, now + offset)
    gain.gain.exponentialRampToValueAtTime(peak, now + offset + 0.01)
    gain.gain.exponentialRampToValueAtTime(0.0001, now + offset + dur)

    osc.connect(gain)
    gain.connect(ctx.destination)
    osc.start(now + offset)
    osc.stop(now + offset + dur + 0.02)
    offset += dur
  }
}

// playLike — ascending two-tone (C5 -> G5) for a positive "liked" feedback.
export function playLike() {
  playSequence([
    { frequency: 523, duration: 100 },
    { frequency: 784, duration: 100 },
  ])
}

// playUnlike — descending two-tone (G5 -> C5) for un-liking.
export function playUnlike() {
  playSequence([
    { frequency: 784, duration: 100 },
    { frequency: 523, duration: 100 },
  ])
}

// playComment — single E5 tone, slightly longer, for sending a comment.
export function playComment() {
  playBeep(659, 150)
}

// playFollow — ascending three-tone (C5 -> E5 -> G5) for following a user.
export function playFollow() {
  playSequence([
    { frequency: 523, duration: 90 },
    { frequency: 659, duration: 90 },
    { frequency: 784, duration: 120 },
  ])
}
