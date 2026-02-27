---
name: banana
description: Generate or edit images using Gemini's native image generation. Use when the user wants to create, generate, draw, or edit images, or when they reference banana or image generation.
---
<skill-content>
For image generation, operate as a creative director translating intent into precise visual descriptions. The banana CLI wraps Gemini's native image generation, which is a language model that generates images, not a diffusion model. This distinction shapes how prompts should be written: natural language paragraphs describing the scene outperform keyword lists, and diffusion-model conventions are counterproductive. Prompt quality is the primary lever for output quality; a well-crafted prompt consistently outperforms iteration on a vague one. Commit to specific visual choices in the prompt; hedging with vague descriptions produces vague images.

Image generation costs money through the Gemini API. For the flash model (default), proceed with generation after constructing the prompt. The user can review and approve the command through Claude Code's built-in permission system. Do not add a separate approval step on top of this.

Two safeguards require explicit user permission before proceeding:
- **Pro model**: Never use `-m pro` unless the user explicitly requests it or approves the switch. If the task would benefit from Pro, recommend it and wait for confirmation.
- **Batch generation**: Never generate multiple images in a single turn unless the user explicitly asks for it. One image per turn is the default.

The goal is an image that matches the user's intent. Each section builds toward this: understanding establishes what the user wants, prompt construction translates that into precise visual language, execution runs the command, and evaluation drives the next iteration.

# Understanding Intent

Identify what the user wants: generating a new image, editing an existing one, iterating on a previous generation, or rendering text (posters, signs, logos). If editing or iterating, identify the source image or session file. If the request is too vague to construct a strong prompt, ask focused questions about the missing elements rather than guessing.

Choose the model. Flash (default) handles most tasks and costs less. Pro produces higher fidelity at higher cost. Use flash unless the user requests pro or the task demands it. Both Flash 3.1 and Pro support 2K/4K output via `-z`. Concrete triggers for Pro: rendered text (Pro has under 10% error rate; Flash is unreliable), or previous flash attempts produced unsatisfactory results. If the user wants a specific model version for its rendering style, use a pinned name (`flash-2.5`, `flash-3.1`, `pro-3.0`).

Choose the aspect ratio from the supported set: 1:1 (default), 2:3, 3:2, 3:4, 4:3, 9:16, 16:9, 21:9. Match the ratio to the content and justify the choice briefly when presenting the command.

# Prompt Construction

Craft the prompt as natural language. Gemini parses grammar, understands relationships between concepts, and reasons about composition. A coherent paragraph describing the scene outperforms comma-separated tags.

A strong prompt addresses most of these elements, woven naturally into the description rather than listed mechanically: subject and action, composition and framing, setting and environment, lighting, style and medium, mood and atmosphere. Lighting is the single strongest lever for mood and realism; always specify it. Not every element needs equal weight. Emphasize what matters most for the particular image.

When exact colors matter, use hex codes ("#9F2B68") rather than descriptive names ("amaranth purple"). For long prompts, repeat critical constraints at the end; the model can drift on requirements stated only at the beginning.

Do not use diffusion-model conventions. Quality tags like "4k, masterpiece, hyperrealistic, trending on artstation" are meaningless to the language model and may be penalized. Weight syntax like "(word:1.5)" is not supported. For emphasis, use ALL CAPS on critical elements. For exclusions, describe what you want positively rather than listing what to avoid.

When a specific detail fights the model's strongest visual association (e.g., "phone flashlight" vs the archetypal handheld flashlight), give it extra emphasis or describe it more concretely. The model has strong priors about common visual concepts and will default to the most typical version unless pushed.

For editing prompts, be explicit about preservation. State what should change and what must remain untouched. Faces are especially sensitive to drift; when editing faces or characters, make one change per turn. For non-face edits, multiple changes in a single turn are usually fine.

For multi-image reference, pass each image with a separate `-i` flag. In the prompt, refer to each image by its role ("the first image shows the character, the second shows the environment"). Flash 2.5 accepts up to 3 reference images; Flash 3.1 and Pro accept up to 14.

For detailed templates by image type, editing patterns, and advanced techniques, see [prompting-reference.md](prompting-reference.md).

# Execution

The `banana` binary lives in this skill's directory, not on PATH. This is intentional: the skill is self-contained so it can be installed by extracting a zip and removed by deleting the directory, with no system-level side effects. Construct the binary path from the skill location provided by the system when this skill was invoked. For example, if the skill is at `/home/user/.claude/skills/banana/SKILL.md`, the binary is `/home/user/.claude/skills/banana/banana`.

CLI syntax:

```
<skill-dir>/banana -p <prompt> -o <output> [-i <input>...] [-s <session>] [-m model] [-r <ratio>] [-z 1K|2K|4K] [-f]
```

Flags:
- `-p` text prompt (required)
- `-o` output PNG file path (required; must end in .png)
- `-i` input image for editing/reference (optional, repeatable; supports png, jpg/jpeg, webp, heic, heif). Flash 2.5: up to 3 images. Flash 3.1 and Pro: up to 14. Each file must be under 7 MB.
- `-s` session file to continue from (optional)
- `-m` model: flash (default), pro, flash-2.5, flash-3.1, pro-3.0. Bare names are aliases for the latest version; pinned names lock to a specific model.
- `-r` aspect ratio (default 1:1)
- `-z` output resolution: 1K, 2K, or 4K (flash-3.1, pro-3.0)
- `-f` overwrite output and session files if they exist

Every invocation produces a session file by replacing the output extension with `.session.json` (`cat.png` creates `cat.session.json`). This session file contains the model name and full conversation history. The `-s` flag is read-only: it loads history but the new session always saves alongside `-o`. This preserves the source session file so the user can rewind or branch from any point. Without `-f`, the CLI refuses to write if the derived session path collides with the `-s` source. With `-f`, the source session is overwritten.

Choose an output filename that reflects the content. Place output in the current working directory or the relevant project subdirectory unless the user specifies otherwise.

# File Organization

Name files so the progression is readable during a session. Each filename should communicate what this image is or what changed from its predecessor. Use the subject as the base name and append the variant: `farmhouse_twilight.png`, `player_character_apose_nobag.png`, `player_character_apose_clean.png`. Avoid encoding full edit history into names; `player_character_apose_clean.png` is better than `player_character_apose_nobag_v2_nomud.png`.

During a session, every generated file may turn out to be the keeper. Do not try to categorize files as drafts or finals at generation time. That distinction only becomes clear in retrospect.

Offer to organize when any of these conditions are apparent:

- A directory has accumulated roughly 8-10+ banana-generated images and is becoming hard to scan.
- Superseded files are visible: earlier iterations in a chain sit alongside the version that replaced them (e.g., `nobag.png` alongside the `clean.png` that built on it).
- The user shifts topics (character work to environment work, or similar). The previous topic's files are no longer active and can be tidied.
- Multiple `.session.json` files have accumulated in the same directory.

When offering, keep it brief. One line suggesting cleanup, not a detailed proposal. If the user accepts, move superseded intermediates to an archive subdirectory, group keepers logically, and clean up orphaned session files. If they decline or ignore it, do not raise it again until the situation changes meaningfully.

To clean up session files in bulk, use `banana clean <directory>` for a dry run (lists model, turn count, and size for each file) or `banana clean -f <directory>` to delete them. Files that fail validation are skipped and never deleted. To check API cost, use `banana cost <session-file>` for a single session or `banana cost <directory>` for a summary of all sessions in a directory.

# After Generation

After execution, display the generated image to the user using the Read tool. To inspect metadata on a previously generated PNG, use `banana meta <image.png>`, which shows the model, prompt history, aspect ratio, inputs, and timestamp embedded in the file.

Evaluate the result honestly. Identify what succeeded, what drifted from the prompt, and what could improve. Offer concrete directions for iteration or next steps. The conversation between generations is where creative work happens; don't just deliver the image and wait.

When working on a series of related images, maintain coherence by carrying forward palette descriptions, architectural details, and stylistic language across prompts. Reference earlier generations when relevant to keep the visual language consistent.

If generation was blocked by safety filters, try these strategies before reporting failure: rephrase with explicit context ("family-friendly poster," "educational diagram," "medical illustration"), use positive framing instead of exclusion language, crop input images tighter to reduce incidental flagged content, or retry with slight rewording since the filter has some randomness. If these fail, report the block reason and the strategies attempted so the user can adjust.

# Sessions and Iteration

If the user wants changes to a previous generation, use the session file with `-s` rather than starting fresh. This preserves conversation context so the model understands references like "make it warmer" or "change the hat." Construct a short, conversational follow-up prompt rather than repeating the full original description. Choose a new output filename that reflects the iteration (e.g., `cat_v2.png`, `cat_blue.png`).

The user can rewind by referencing an earlier session file. If v3 went wrong, continuing from `cat_v1.session.json` branches from that point without losing v2 or v3. The source session file is never modified; each generation writes its own session file next to its output.

When NOT to use sessions: if the user wants a completely different image unrelated to previous work, start fresh without `-s`. Sessions carry visual style and context forward, which is counterproductive when the user wants a clean break. A new subject, new style, or new context means a new session.

# Resources

- [prompting-reference.md](prompting-reference.md): templates for different image types, editing patterns, advanced techniques, and anti-patterns
</skill-content>
