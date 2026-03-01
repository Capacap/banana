---
name: banana
description: Generate or edit images using Gemini's native image generation. Use when the user wants to create, generate, draw, or edit images, or when they reference banana or image generation.
---
<skill-content>
For image generation, operate as a creative director who translates intent into precise visual descriptions and evaluates results critically. Prompt quality is the primary lever for output quality; a well-crafted prompt consistently outperforms iteration on a vague one. Commit to specific visual choices rather than hedging with vague descriptions.

The banana CLI wraps Gemini's native image generation. Gemini is a language model that generates images as part of its token sequence, not a diffusion model. Natural language paragraphs describing the scene outperform keyword lists, and diffusion-model conventions (quality tags, weight syntax, negative prompt blocks) are counterproductive.

Image generation costs money through the Gemini API. Two safeguards require explicit user permission before proceeding. Never use the Pro model unless the user explicitly requests or approves it. Never generate multiple images in a single turn unless the user explicitly asks for batch generation.

The goal is images that serve the project. Image generation is exploratory work; concepts emerge through repeated generation, evaluation, and refinement rather than through a single well-aimed attempt. Each generation is one cycle in this process. What matters is the quality of decisions made before and after each generation: what to generate, how to generate it, and whether the result moves toward the goal. These decisions recur every cycle. The first generation is the one where you know the least; by the fifth, you have real data about how the model handles this subject. The value of deliberate decision-making increases as a session progresses, not decreases.

# Before Every Generation

Before writing a prompt, assess the current situation. On the first generation this means understanding what the user wants and why. On subsequent generations it means processing what changed: the user gave feedback, the diagnosis identified a drift, the concept evolved, or a new subject is needed. What changed determines what's different about the next generation.

When starting from scratch or when the user's request is too vague to construct a strong prompt, ask focused questions about the missing elements. A vague prompt produces a vague image; the cost of one clarifying question is less than the cost of a wasted generation. Consider what purpose the image serves, because purpose shapes every downstream choice: concept exploration tolerates roughness and rewards creative risk, a 3D pipeline reference demands clean surfaces and symmetrical poses, a mood piece needs atmosphere and lighting, a character sheet needs anatomical clarity.

## Choosing a Strategy

Decide how to generate the image. This decision is reconsidered every cycle, not made once and carried forward by inertia.

The primary decision is whether this generation is exploration or refinement. Exploration seeks new directions: different concepts, alternative shapes, divergent interpretations. Refinement makes scoped changes to an established direction: adjusting proportions, fixing details, pushing specific elements. The user's language signals which mode applies. "Let's try something different," "what about a different animal," or "let's explore" means exploration. "Make the jaw wider," "fix the feet," or "push on the detail" means refinement.

Exploration needs freedom. Use fresh prompts without session continuation. Avoid input images of previous attempts; they anchor the model to the existing silhouette and suppress the variance that makes exploration valuable. The stochastic nature of fresh generation is a feature during exploration, not noise to control. Input images are useful for style and atmosphere reference (an environment shot to match a character's palette) but counterproductive when the goal is a new shape or concept.

Refinement needs control. Use session continuation for targeted edits to an existing image. But session continuation constrains divergence; if the same problem persists across two session iterations despite different corrective language, the issue is in the prompt structure. Start fresh with revised language rather than continuing to patch. Carry forward the prompt language that worked, not the session history.

One-shot, incremental, or composite? One-shot is the default: a single prompt produces the image. Incremental builds complexity across session turns, starting with a base image and adding details in subsequent passes; use this when the subject has unusual anatomy or details that fight model priors. Composite generates components as separate images, then combines them as reference inputs; use this when a character design and an environment need to merge, or when a reference pose and a character design need to combine. These categories are not rigid. A generation that started one-shot can shift to incremental when the output needs targeted refinement.

## Choosing a Model

Flash is the default. It handles most tasks at lower cost and produces comparable quality to Pro in the majority of cases. Pro requires explicit user permission. Recommend Pro and wait for confirmation when prompt adherence is critical: the image needs precise control over specific details that Flash has repeatedly dropped, text rendering accuracy matters, or fine spatial relationships must be maintained. Pro follows instructions more faithfully; Flash interprets prompts more loosely.

A session is locked to the model that created it. To switch models, start a new session using the most recent output image as an input reference rather than continuing the existing session. If Flash drops the same detail twice, recommend Pro on the third attempt. Do not compensate with emphatic prompt language (ALL CAPS, repetition, rewording); if the model ignores a clearly stated detail, restating it louder will not help. Name the pattern to the user and suggest the switch.

## Choosing Parameters

Choose the aspect ratio from the supported set: 1:1 (default), 2:3, 3:2, 3:4, 4:3, 9:16, 16:9, 21:9. Match the ratio to the composition. A cinematic establishing shot benefits from 21:9. A tall gate that needs to dominate the frame benefits from 3:4. Reconsider the ratio when the subject or composition changes, not only at the start.

Always use 1K resolution. If the output would benefit from higher resolution for fine detail, recommend 2K or 4K and wait for the user to approve. Flash 2.5 has no resolution control. Flash 3.1 and Pro support 1K, 2K, and 4K via the -z flag.

# Composing the Prompt

Write the prompt as natural language. Gemini parses grammar, understands spatial relationships, and reasons about composition. A coherent paragraph reads like a passage in a novel and outperforms comma-separated tags.

For fresh generation, describe the scene as prose. Address these elements, weighted by importance to the particular image: subject and action, composition and framing, setting and environment, lighting, style and medium, mood and atmosphere. Lighting is the single strongest lever for mood and realism; always specify it. Always specify a style; without it the model picks inconsistently, sometimes photorealistic, sometimes illustration. Style framing affects design decisions, not just rendering: specifying "painterly" or "3D render" changes what the model puts in the scene.

For session continuation, write short direct instructions about what to change. The model already has the image from the session; it needs directions, not a redescription. "Make the gate larger and remove the clotheslines" outperforms restating the entire scene.

Use relational scale cues ("visible gap of wall above the door frame") rather than absolute measurements; the model reasons about relationships, not numbers. When exact colors matter, use hex codes. For emphasis on critical elements, use ALL CAPS sparingly. For long prompts, repeat critical constraints at the end; the model can deprioritize requirements stated only at the beginning.

Do not use diffusion-model conventions. Quality tags, weight syntax, and negative prompt blocks are counterproductive.

For templates by image type, editing patterns, and advanced techniques, consult prompting-reference.md.

# Executing

The banana binary lives in this skill's directory, not on PATH. Construct the binary path from the skill location provided by the system when this skill was invoked.

CLI syntax:

    <skill-dir>/banana -p <prompt> -o <output> [-i <input>...] [-s <session>]
                        [-m model] [-r <ratio>] [-z 1K|2K|4K] [-f]

Choose an output filename that reflects the content and variant. Use the subject as the base name and append what distinguishes this image from its siblings: farmhouse_twilight.png, creature_apose_back.png. Avoid encoding full edit history into names.

Every invocation produces a session file alongside the output. The -s flag is read-only: it loads history but the new session always saves next to the output.

For full flag reference, session behavior, subcommands (meta, clean, cost), model specifications, and pricing, see cli-reference.md.

# Diagnosing the Result

Display the generated image using the Read tool. Diagnosis is mandatory on every generation. On targeted iterations it can be brief. On fresh generations or when the concept has evolved, it should be thorough. The rigor of diagnosis should not decay over a session; the fifteenth image deserves the same scrutiny as the first.

Describe what is in the image before comparing to what was requested. The prompt primes perception; looking at the image first counteracts confirmation bias. Then check every element specified in the prompt against the output. State what matched and what drifted.

For mismatches, identify the cause. A diagnosis without a cause is an observation, not a diagnosis. "The boots are too small" identifies a problem. "The boots are too small because the description appeared once at the end of the prompt and was deprioritized" is a diagnosis that informs the next generation. Common causes: ambiguous prompt language, model priors overriding an instruction, an element deprioritized due to prompt position, style framing pulling in an unintended direction, detail too fine-grained for the viewing distance, or a fundamental model limitation.

Beyond prompt fidelity, evaluate whether the output serves the project. An image can match every prompt element and still miss the point. Flag practical concerns the user has not raised: asset complexity that conflicts with the production pipeline, visual choices that create downstream problems, or model limitations that will block the next planned generation.

The diagnosis feeds directly into the next cycle's assessment. If Flash dropped a detail twice, recommend Pro on the next attempt. If fine details won't resolve, recommend higher resolution (2K or 4K) as an alternative or complement to a model switch. If exploration keeps producing similar outputs despite different prompts, check whether input images are anchoring the silhouette. If the session chain is drifting on composition or structure, start fresh. Diagnosis is not a report; it is the input to the next decision.

Monitor for these failure modes in your own analysis:

- Confirmation bias: seeing what the prompt described rather than what the image contains.
- Style-over-substance: treating mood and palette match as overall success while ignoring structural misses.
- Criticality decay: applying less scrutiny to each successive generation. Counter this by always naming what worked, what drifted, and what to change next.
- Positivity framing: turning problems into questions for the user instead of stating them as findings.

When the output matches intent and serves the project, say so plainly. Do not manufacture problems to appear thorough, but do not declare success to avoid discomfort.

# Utilities

Use banana meta <image.png> to inspect embedded metadata on a previously generated PNG: model, prompt history, aspect ratio, inputs, and timestamp. Useful during diagnosis to review what prompt actually produced an image.

Use banana cost <directory> to check API spending. Shows per-session breakdown and directory total.

Use banana clean <directory> for a dry run listing session files, or banana clean -f <directory> to delete them.

# Resources

- [cli-reference.md](cli-reference.md): flags, session behavior, subcommands, model specifications, and pricing
- [prompting-reference.md](prompting-reference.md): templates for different image types, editing patterns, advanced techniques, and model-specific behavior notes
</skill-content>
