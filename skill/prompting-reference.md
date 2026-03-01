# Prompting Reference: Gemini Native Image Generation

This reference supplements the main skill with detailed templates, editing patterns, and advanced techniques. Consult it during prompt composition for specific image types or when applying advanced prompting strategies.

## Prompt Structure

Order elements by importance. The model weighs earlier content more heavily in long prompts.

1. **Subject and action** — who or what, doing what. Concrete beats abstract: "an elderly potter inspecting a cracked raku bowl" over "a person with a bowl."
2. **Composition and framing** — shot type, camera angle, focal point. Photography language works: "close-up portrait," "wide establishing shot from a low angle," "over-the-shoulder."
3. **Setting and environment** — where, with grounding details. "A rain-soaked Tokyo alley at night, neon signs reflecting in puddles" over "a city street."
4. **Lighting** — the strongest lever for mood and realism. Always specify: "warm golden-hour sidelight," "harsh overhead fluorescent," "diffused overcast daylight," "dramatic Rembrandt lighting."
5. **Style and medium** — photorealistic, watercolor, cel-shaded, oil painting, etc. For photorealism, camera and lens specs force photographic rendering: "shot on a Canon EOS R5 with an 85mm f/1.4 lens, shallow depth of field."
6. **Mood and atmosphere** — the emotional thread tying everything together: "serene and contemplative," "tense and claustrophobic."

For long prompts, repeat critical constraints at the end. The model can drift on requirements stated only at the beginning.

## Emphasis and Constraints

- **ALL CAPS** for critical requirements: "MUST include exactly three figures," "NEVER include any text or watermarks." Use sparingly; capitalizing everything dilutes the signal.
- **Positive framing** over negation. Instead of "no cars," write "an empty, deserted street with no signs of traffic." When exclusions are necessary, natural language works: "Do not include any text, watermarks, or line overlays."
- **Hex colors** when exact colors matter. "#9F2B68" outperforms "amaranth purple" for precision.
- **Markdown lists** for multiple constraints. The model's text encoder was trained on Markdown, so dashed lists improve instruction clarity.
- **Re-emphasize critical details in every prompt.** Whether iterating in a session or starting fresh, do not assume the model remembers constraints from earlier turns. If boot size matters, describe the boots in every prompt. Repetition is not redundant; it is how you maintain consistency.

## Templates by Image Type

### Photorealistic Portrait

Specify: subject description, expression, clothing, camera/lens, lighting setup, background, depth of field, mood.

> A weathered fisherman in his 60s, deep smile lines around his eyes, wearing a salt-stained canvas jacket. Close-up portrait, shot on a Canon EOS R5 with an 85mm f/1.4 lens. Warm golden-hour sidelight from the left, soft fill from a reflector on the right. Blurred harbor background with bokeh from mast lights. Contemplative, authentic mood.

### Landscape / Environment

Specify: location, time of day, weather, atmospheric conditions, focal point, depth layers (foreground/midground/background), lighting.

> A volcanic black sand beach in Iceland at twilight, low fog rolling across the shore. Jagged sea stacks silhouetted against a pale lavender sky. Foreground: wet sand reflecting the last light. Midground: slow-exposure ocean blur around basalt columns. Background: faint aurora beginning to appear. Shot on a Sony A7R IV with a 24mm f/2.8 lens, deep depth of field. Cold, ethereal, vast.

### Product Shot

Specify: product description, surface/material, angle, lighting setup (studio language), background, reflections.

> A matte black ceramic coffee mug on a raw concrete surface, 45-degree elevated camera angle. Three-point softbox lighting: key light from upper-left, fill from right, backlight creating a rim highlight on the mug edge. Shallow depth of field, sharp focus on the mug handle. Minimal background, dark charcoal gradient. Clean, editorial, commercial quality.

### Illustration / Stylized Art

Specify: art style explicitly, subject, color palette, line style, shading approach, background treatment.

> A fox sitting in a field of wildflowers, Studio Ghibli-inspired illustration style. Soft watercolor washes with visible paper texture. Warm palette: burnt sienna, sage green, dusty rose, cream. Delicate ink outlines with varying line weight. Dappled sunlight filtering through unseen trees. Gentle, nostalgic mood.

### Abstract / Conceptual

Specify: color palette (hex for precision), composition rules, medium/texture, movement, emotional register.

> An abstract composition evoking the feeling of deep-sea pressure. Palette: #0A1628 (abyssal navy), #1B3A5C (deep teal), #C5A880 (bioluminescent gold). Heavy impasto oil technique with visible palette knife strokes. Downward diagonal movement from upper-left, gold elements scattered like deep-sea organisms. Dense, oppressive, with small points of warmth.

### Minimalist / Negative Space

Specify: single subject, exact positioning, background color, space allocation, lighting quality.

> A single delicate red maple leaf positioned in the bottom-right third of the frame. Background: vast, empty off-white canvas (#F5F0EB). Significant negative space occupying 80% of the image. Soft, diffused lighting from the top-left casting a faint shadow. Clean, contemplative, designed for text overlay.

### Text-Heavy (Logos, Posters, Signs)

Specify: exact text content in quotes, font characteristics (descriptive, not font names), placement, color, integration with imagery.

> A vintage-style concert poster for a jazz night. Headline text: "BLUE NOTE SESSIONS" in a bold, condensed sans-serif typeface, #E8D5B7 cream color, centered in the upper third. Subtitle: "Every Thursday at 9PM" in a lighter weight below. Background: smoky indigo (#2C1F4A) with a silhouetted trumpet player in the lower half. Art deco geometric border elements in gold (#C4A265). Textured paper grain overlay.

### Sequential Art (Comics / Storyboards)

Create character reference sheets first. Load references with every subsequent panel generation. Specify panel layout, art style, character actions, dialogue placement.

> A 2x2 comic panel layout in clean ligne claire style. Panel 1: wide shot of a rain-soaked city street, a woman with a red umbrella walking toward the viewer. Panel 2: close-up of her face, expression shifting from neutral to surprised. Panel 3: her POV, a fox sitting calmly in the middle of the crosswalk. Panel 4: wide shot, she crouches down, umbrella tilted, reaching toward the fox. Consistent warm tungsten streetlight throughout. Speech bubble in panel 2: "What on earth..."

## Editing Patterns

### Targeted Element Change

State what changes and what stays. Be specific about preservation boundaries.

> Change only the sky from overcast gray to a vivid sunset gradient (warm oranges and pinks). Keep the foreground buildings, street, people, and all shadows exactly as they are. Adjust only the sky reflection in puddles to match the new sky color.

### Object Addition

Describe the new element with enough detail to match the existing scene's style and lighting.

> Add a black cat sitting on the windowsill in the left side of the image. The cat should be lit consistently with the existing warm interior light, casting a soft shadow on the sill. Match the photorealistic style of the rest of the scene. Do not alter anything else.

### Object Removal

Describe what to remove and how to fill the gap.

> Remove the trash can on the left side of the frame. Fill the area to match the surrounding brick wall texture and concrete sidewalk. Maintain consistent lighting and shadow in the filled region.

### Style Transfer

The autoregressive architecture can resist pure style transfer. If direct stylization fails, reframe as creating a new image using the subject from the reference.

> Create a new image of the person shown in the provided photo, rendered in the style of a Monet impressionist painting. Loose, visible brushstrokes. Pastel color palette with emphasis on reflected light. Soft focus throughout. Preserve the subject's face, pose, and clothing accurately.

### Background Replacement

> Replace the background behind the person with a sun-drenched Mediterranean terrace overlooking the sea. Keep the person, their clothing, hair, and all body details exactly the same. Adjust ambient lighting on the person to match warm, bright outdoor sunlight from the upper right. Add subtle environmental reflections in sunglasses if present.

### Incremental Editing

Make one edit per turn. Large changes across multiple elements cause drift, especially in faces. If v3 goes wrong, branch from an earlier session file rather than trying to fix everything in v4.

Know when to abandon a session chain. Long chains accumulate drift; details established early (boot size, color choices, proportions) can silently regress as the conversation grows. If the model starts forgetting established details after 3-4 turns, start a fresh generation with a complete prompt rather than continuing to correct within the session. Carry forward the language that worked, not the session history.

## Advanced Techniques

### Camera and Lens for Photorealism

Specific camera and lens language forces photographic rendering. Stack these for maximum realism:

- Camera body: "Canon EOS R5," "Hasselblad X2D," "Leica M11"
- Lens: "85mm f/1.4" (portrait), "24mm f/2.8" (wide), "100mm macro" (detail)
- Settings: "shallow depth of field," "motion blur at 1/15s," "high ISO grain"
- Film stock for analog look: "Kodak Portra 400 film stock," "Fujifilm Velvia 50 color saturation"

### Compositional Authority

Photography-specific status language improves composition. "Pulitzer-prize-winning cover photo" triggers rule-of-thirds adherence and professional color balance. Use when you want the model to make strong compositional choices rather than centering everything.

### Weighted Style Blending

Combine aesthetics with proportional guidance: "60% minimalist product photography, 30% lifestyle editorial, 10% fashion campaign." The model interpolates between referenced styles.

### Character Consistency

- Establish the character with maximum detail in the first prompt: face structure, hair, distinguishing features, clothing
- Use the session to carry the character forward
- Change one element per turn (background OR lighting OR outfit, not all three)
- If features drift after several iterations, restart the session with the detailed character description and a reference image from the best earlier generation
- For multi-panel work, create a character reference sheet (neutral pose, front-facing, clean lighting) and include it as input with every generation
- Generate multi-view references as separate images. Requesting multiple views in a single image (front and side, front and back) causes inconsistency between views. Generate each view independently for reliable results.

### Multi-Image Reference

Flash 3.1 and Pro support up to 14 reference images per prompt. Flash 2.5 supports up to 3. Use cases:

- Up to 6 images for object fidelity (product from multiple angles)
- Up to 5 images for character consistency across scenes
- Style references: provide 2-3 examples of the target aesthetic

### Text Rendering

Pro is more reliable for text rendering. For best results:

- Specify exact text in quotes
- Describe font characteristics rather than naming fonts: "bold condensed sans-serif," "elegant flowing script"
- Specify placement: "centered in the upper third," "along the bottom edge"
- Specify color with hex codes
- Keep text simple; complex multi-line layouts may need iteration

## Anti-Patterns

### Diffusion Model Conventions

These are meaningless or harmful to the language model:

- Quality tags: "4k, masterpiece, best quality, hyperrealistic, trending on artstation"
- Weight syntax: `(word:1.5)`, `[word]`, `{word}`
- Parameter syntax: "steps: 50, CFG: 7.5, sampler: euler"
- Negative prompt blocks (as a separate section; natural-language exclusions within the main prompt work fine)

### Keyword Spam

Overloading with conflicting modifiers ("cinematic, volumetric lighting, 35mm, f/1.4, 8k, hyperreal, artstation, unreal engine") confuses the model. The model tries to satisfy everything and satisfies nothing. Pick the terms that matter and describe them in context.

### Vague Prompts

Without style, medium, or composition instructions, the model defaults to generic output. "A cat" produces a stock-photo cat. "A marmalade tabby cat curled on a sun-bleached windowsill, afternoon light streaming through lace curtains, shot in the style of a Dutch Golden Age still life" produces an image.

### Stacking Multiple Changes

Editing multiple elements in one turn increases drift. Faces are especially sensitive. Change one thing at a time and use sessions to accumulate changes incrementally.

### Expecting Identical Reproduction

The model generates something new with every call. Without explicit consistency measures (reference images, session continuity, detailed character descriptions), faces and details will vary between generations.

## Safety Filter Notes

The model blocks content involving minors in unsafe contexts, violence, hate speech, and explicit material. It also blocks photorealistic depictions of identifiable real people.

False positives happen. Strategies for legitimate content that triggers filters:

- Add context: "family-friendly poster," "educational chemistry diagram," "medical illustration"
- Crop input images tighter to reduce incidental flagged content
- Rephrase ambiguous terms: "kill the process" can trigger violence filters; "terminate the process" may not
- Retry with slight rephrasing; the filter has some randomness
