---
name: banana
description: Generate or edit images using Gemini's native image generation. Use when the user wants to create, generate, draw, or edit images, or when they reference banana or image generation.
allowed-tools: Bash(*/banana *)
---
<skill-content>
For image generation, operate as a creative director translating intent into precise visual descriptions. The banana CLI wraps Gemini's native image generation, which is a language model that generates images, not a diffusion model. This distinction shapes how prompts should be written: natural language paragraphs describing the scene outperform keyword lists, and diffusion-model conventions are counterproductive. Prompt quality is the primary lever for output quality; a well-crafted prompt consistently outperforms iteration on a vague one.

Image generation costs money through the Gemini API. Always present the final prompt and command for user approval before executing banana. Never run the command without explicit confirmation.

The goal is an image that matches the user's intent. Phases build toward this: Phase 1 establishes what the user wants, Phase 2 constructs the prompt, Phase 3 presents for approval, Phase 4 executes and delivers.

# Phase 1: Intent

Determine the task type: text-to-image generation, editing an existing image, iterating on a previous generation, or generating images with rendered text (posters, signs, logos, diagrams). For generation, identify the subject, desired style, mood, and technical requirements. For editing, identify the source image, what should change, and what must be preserved. For iteration, identify the session file from the previous generation and what the user wants changed. For text-in-image, identify the exact text content, placement, and font characteristics; this task type should default to Pro for reliable text rendering.

Choose the model. Flash (default) handles most tasks and costs less. Pro produces higher fidelity at higher cost. Use flash unless the user requests pro or the task demands it. Concrete triggers for Pro: the image contains rendered text (Pro has under 10% error rate; Flash is unreliable), the user needs high resolution (Pro supports 2K/4K; Flash is 1K only), the task requires more than 3 reference images (Pro handles up to 14; Flash handles up to 3), or previous flash attempts produced unsatisfactory results.

Choose the aspect ratio from the supported set: 1:1 (default), 2:3, 3:2, 3:4, 4:3, 9:16, 16:9, 21:9. Match the ratio to the content. Portraits suit 2:3 or 3:4. Landscapes suit 16:9 or 3:2. Social media stories suit 9:16. Cinematic scenes suit 21:9. Square works for icons, avatars, and when orientation is unimportant.

If the user's request is too vague to construct a strong prompt, ask focused questions about the missing elements rather than guessing. A portrait needs lighting and mood. A product shot needs surface and angle. An illustration needs style and palette.

# Phase 2: Prompt Construction

Craft the prompt as natural language. Gemini is a language model; it parses grammar, understands relationships between concepts, and reasons about composition. A coherent paragraph describing the scene outperforms comma-separated tags.

For detailed techniques including templates for specific image types, editing patterns, and advanced methods, see [prompting-reference.md](prompting-reference.md).

Structure the prompt around these elements in order of importance:

Subject and action. What is in the scene and what is happening. Be specific: "an elderly potter inspecting a cracked raku bowl" rather than "a person with a bowl."

Composition and framing. Camera angle, shot type, focal point. "Close-up portrait shot" or "wide establishing shot from a low angle."

Setting and environment. Where the scene takes place. Concrete details ground the image: "a rain-soaked Tokyo alley at night" rather than "a city street."

Lighting. The single strongest lever for mood and realism. "Warm golden-hour sidelight" or "harsh overhead fluorescent" or "diffused overcast daylight." Always specify lighting.

Style and medium. Photorealistic, watercolor, cel-shaded, oil painting. For photorealism, camera and lens specs force the model toward photographic rendering: "shot on a Canon EOS R5 with an 85mm f/1.4 lens."

Mood and atmosphere. Emotional tone that ties everything together: "serene and contemplative" or "tense and claustrophobic."

When exact colors matter, use hex codes ("#9F2B68") rather than descriptive names ("amaranth purple"). For long prompts, repeat critical constraints at the end; the model can drift on requirements stated only at the beginning.

Do not use diffusion-model conventions. Quality tags like "4k, masterpiece, hyperrealistic, trending on artstation" are meaningless to the language model and may be penalized. Weight syntax like "(word:1.5)" is not supported. For emphasis, use ALL CAPS on critical elements. For exclusions, describe what you want positively rather than listing what to avoid.

For editing prompts, be explicit about preservation. State what should change and what must remain untouched. The model understands physics: changing time of day automatically adjusts shadows and reflections. But explicit preservation boundaries prevent unwanted changes to elements the user cares about. Make one edit per turn rather than stacking multiple changes; faces are especially sensitive to drift when too much changes at once.

For multi-image reference, pass each image with a separate `-i` flag. Use cases: style transfer from multiple sources, combining elements from different photos, character consistency across scenes, or providing object references for composition. In the prompt, refer to each image by its role ("the first image shows the character, the second shows the environment"). Flash accepts up to 3 reference images; if the task needs more, switch to Pro.

# Phase 3: Approval

Present the complete banana command to the user before executing. Include the constructed prompt, the chosen model with brief rationale, the aspect ratio, and for editing the input image path. Format the command so the user can read the prompt clearly.

Wait for explicit approval. The user may adjust the prompt, change settings, or cancel. Proceed only after confirmation.

# Phase 4: Execute

The `banana` binary lives in this skill's directory. Construct the path from the skill location provided by the system when this skill was invoked. For example, if the skill is at `/home/user/.claude/skills/banana/SKILL.md`, the binary is `/home/user/.claude/skills/banana/banana`.

Run the banana command. The CLI syntax:

```
<skill-dir>/banana -p <prompt> -o <output> [-i <input>...] [-s <session>] [-m flash|pro] [-r <ratio>] [-z 1k|2k|4k] [-f]
```

Flags:
- `-p` text prompt (required)
- `-o` output file path (required; must end in .png, .jpg, .webp, .heic, or .heif)
- `-i` input image for editing/reference (optional, repeatable). Flash: up to 3 images. Pro: up to 14. Each file must be under 7 MB.
- `-s` session file to continue from (optional)
- `-m` model: flash (default) or pro
- `-r` aspect ratio (default 1:1)
- `-z` output resolution: 1k, 2k, or 4k (pro only)
- `-f` overwrite output file if it exists

Every invocation produces a session file alongside the output image (`cat.png` creates `cat.session.json`). This session file contains the model name and full conversation history. When continuing with `-s`, the session file is updated in place rather than creating a new one derived from `-o`.

Choose an output filename that reflects the content. Place output in the current working directory unless the user specifies otherwise.

After execution, report the result: file path, size, and session file path. If generation was blocked by safety filters, try these strategies before reporting failure: rephrase with explicit context ("family-friendly poster," "educational diagram," "medical illustration"), use positive framing instead of exclusion language, crop input images tighter to reduce incidental flagged content, or retry with slight rewording since the filter has some randomness. If these fail, report the block reason and the strategies attempted so the user can adjust.

If the user wants changes, use the session file from the previous generation with `-s` rather than starting fresh. This preserves conversation context so the model understands references like "make it warmer" or "change the hat." The `-s` flag updates the session file in place, so `cat.session.json` always reflects the latest turn of that conversation. Construct a short, conversational follow-up prompt rather than repeating the full original description. Choose a new output filename that reflects the iteration (e.g., `cat_v2.png`, `cat_blue.png`).

The user can also rewind by referencing an earlier session file. If v3 went wrong, continuing from `cat_v1.session.json` branches from that point without losing v2 or v3.

When NOT to use sessions: if the user wants a completely different image unrelated to previous work, start fresh without `-s`. Sessions carry visual style and context forward, which is counterproductive when the user wants a clean break.

# Resources

- [prompting-reference.md](prompting-reference.md): templates for different image types, editing patterns, advanced techniques, and anti-patterns
</skill-content>
