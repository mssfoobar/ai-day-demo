# Deck design rules

Three rules govern this deck. Apply them to every slide.

## 1. Extreme minimalism and whitespace

No dense paragraphs, no corporate bullet lists, no clutter. A slide carries a
single dominant element: one high-resolution image, one massive number, or one
clean diagram. The speaker supplies the narrative; the slide supplies the
evidence.

If a slide needs a sentence to explain itself, the sentence belongs in the
speaker notes.

## 2. Visual hierarchy

Type size, weight, and high-contrast colour steer the eye to the takeaway
before the viewer reads anything. Stark white on matte black, or the reverse.
One thing on the slide should be unmistakably the biggest.

Rank every element. If two things compete for first place, cut one.

## 3. Seamless continuity

Transitions are narrative, not decoration. Concepts evolve across slides
rather than being replaced by them. An element that persists between two
slides should move rather than disappear and reappear.

Prefer a build that transforms what is already on screen over a cut to a new
layout.

## How the deck applies them

- **One element per slide.** The mandate, the AOH definition, and each layer
  takeaway are single statements on their own slides. Everything explanatory
  lives in speaker notes.
- **Overview is a deliberate exception.** It pairs two supporting points on
  the left with the diagram on the right. The diagram stays dominant: the
  points are short, unemphasised, and reveal one at a time ahead of it.
- **Hierarchy by type size.** Statement slides run at `text-5xl` or `text-6xl`
  bold, with any supporting command at `text-2xl` and reduced opacity. Nothing
  competes for first place.
- **Continuity over cuts.** The deck transition is `fade`. On the diagram
  slide the project caps and domain blocks never move; only the module blocks
  cross-fade in place into the shared foundation, so the click transforms what
  is on screen rather than replacing it.

## Still open

Colour scheme is the theme default rather than stark white on matte black.
Going full contrast is a deck-wide theme decision, not a per-slide one.
