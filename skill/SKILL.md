# Banana Image Generation CLI Tool

You have access to the banana CLI tool for sending image creation and editing requests to the Google Gemini Image models through the Gemini API. Follow these guidelines for effective application of the tool in creative tasks.

## Banana CLI Reference

The banana binary lives in this skill's directory, not on PATH. Construct the binary path from the skill location provided by the system when this skill was invoked.

### Primary Command

```
banana -p <prompt> -o <output> [-i <input>...] [-s <session>] [-m model] [-r <ratio>] [-z 1K|2K|4K] [-t min|high] [-f]
```

Send instructions to Gemini Image models, outputs a PNG and a JSON session file. Output is always PNG. The Gemini API returns PNG data; other output formats are not supported.

| Flag | Required | Description |
|------|----------|-------------|
| `-p` | yes | Text prompt |
| `-o` | yes | Output PNG file path (must end in `.png`) |
| `-i` | no | Input image for editing/reference (repeatable; supports png, jpg/jpeg, webp, heic, heif) |
| `-s` | no | Session file to continue from |
| `-m` | no | Model: `flash` (default), `pro`, `flash-2.5`, `flash-3.1`, `pro-3.0` |
| `-r` | no | Aspect ratio (default `1:1`): `1:1`, `2:3`, `3:2`, `3:4`, `4:3`, `9:16`, `16:9`, `21:9` |
| `-z` | no | Output size: `1K`, `2K`, or `4K` (`flash-3.1`, `pro-3.0` only) |
| `-t` | no | Thinking level: `min` (default), `high` (`flash-3.1` only) |
| `-f` | no | Overwrite output and session files if they already exist |

Pass `-i` multiple times for multiple reference images. Each file must be under 7 MB. Model-specific limits:
- Flash 2.5: up to 3 input images
- Flash 3.1: up to 14
- Pro 3.0: up to 14

The CLI validates file existence, MIME type, and size before making the API call. If the input count exceeds the model limit, the error message suggests switching to a model with a higher cap.

The `-r` flag sets the aspect ratio for the generated image. The default is `1:1`. All supported ratios: `1:1`, `2:3`, `3:2`, `3:4`, `4:3`, `9:16`, `16:9`, `21:9`.

The `-z` flag controls output resolution. Only available on models with resolution support (`flash-3.1`, `pro-3.0`). Options: `1K`, `2K`, `4K`. Defaults to `1K` when omitted.

The `-t` flag controls how much the model reasons before generating. `min` (the default) is faster and cheaper. `high` makes the model think more deeply about the prompt, which can improve adherence to complex instructions at the cost of higher latency and token usage. Only available on `flash-3.1`.

### Session files

IMPORTANT: A session cannot switch model. The chosen model must match the one recorded in the file. Bare aliases are resolved before comparison, so `-m flash` and `-m flash-3.1` are equivalent when `flash` aliases to `flash-3.1`. If you must pivot to a different model, start a new session with the new model and use the last image of the previous session as an input.

Every generation produces a session file alongside the image output (e.g., `cat.png` produces `cat.session.json`). It records the session history for multi-turn interactions.

The `-s` flag is read-only: it loads history from the specified session file but never writes back to it. The new continued session file always saves alongside `-o`. This preserves the source session for rewind and branching. To branch from an earlier point, pass that point's session file with `-s` and a new output path with `-o`. The earlier session file remains intact, and the new generation starts a new branch from that history.

When continuing a session, the tool will automatically reuse the session's last used CLI flags. If other arguments are provided, they will override the session defaults.

## Subcommands

### meta

```
banana meta <image.png>
```

Show metadata embedded in a generated PNG. Non-PNG files and PNGs without banana metadata produce distinct error messages.

### clean

```
banana clean <directory>        # dry run: list files with details
banana clean -f <directory>     # delete validated session files
```

Find and remove session files from a directory. Scans non-recursively. Files that fail validation (corrupt JSON, unknown structure) are skipped with a warning and never deleted.

### cost

```
banana cost <session-file>      # single session breakdown
banana cost <directory>         # summarize all sessions in a directory
```

Estimate API cost from session files using a hard-coded snapshot of published Gemini API pricing. Exact prices may be outdated.

### transform

```
banana transform -i <input> -o <output> [-f] <operation> [args]
```

Flip, rotate, or resize an image locally without calling the Gemini API. Input and output must both be `.png`. Existing banana metadata is preserved through the transform.

Operations:
- `flip-h` — horizontal flip (mirror)
- `flip-v` — vertical flip
- `rotate 90|180|270` — clockwise rotation
- `resize WxH|Wx|xH` — resize to exact dimensions, or proportionally by specifying only width (`800x`) or height (`x600`)

Resize uses CatmullRom interpolation. Use `-f` to overwrite an existing output file.

## Gemini Image Models

More well-known under the alias 'Nano Banana', the Gemini Image models are a family of language models that generate images as part of their token sequence. They are not diffusion models. They parse grammar, understand spatial relationships, and reason about composition.

Within the family there are two model categories: Flash and Pro. Flash is a smaller, quicker model while Pro is slow but accurate. Both generally produce images of equal visual fidelity, though Pro is better at parsing complicated instructions and getting the details right.

The models are biased toward natural and cinematic-looking images. Technical views (orthographic projections, turnaround sheets, precise camera angles) are less reliable, especially for unconventional subjects. Conventional humanoid front and side views work reasonably well; unusual body plans and strict orthographic precision require more iteration.

## Prompting Fundamentals

Write prompts as natural language instructions. The model parses grammar and reasons about composition. An instructive prompt with clear actionable specifications will outperform overloaded descriptions and keyword lists.

Design instructions around model behavior:
- Relational cues rather than absolute measurements. The model can reason about spatial relationships but has no way of performing precise measurements. For precise camera angles, reinforce terminology with geometric constraints the model can check: "the far-side limbs are completely hidden behind the near-side limbs," "the top surface of the back is not visible." These give concrete pass/fail criteria that terms like "orthographic" alone do not.
- Positive framing over negation. Instead of "no cars," write "an empty, deserted street with no signs of traffic." When exclusions are necessary, natural language works: "Do not include any text, watermarks, or line overlays."
- Hex colors when exact colors matter. "#9F2B68" outperforms "amaranth purple" for precision.
- Markdown lists for multiple constraints. The model's text encoder was trained on Markdown, so dashed lists improve instruction clarity.
- ALL CAPS for critical requirements: "MUST include exactly three figures," "NEVER include any text or watermarks." Use sparingly; capitalizing everything dilutes the signal.
- Repeat critical requirements at the end of long prompts. The model weighs earlier content more heavily in lengthy prompts.

Prompt anti-patterns:
- Diffusion-model conventions such as quality tags, weight syntax, and negative prompt blocks.
- Overloading with conflicting modifiers ("cinematic, volumetric lighting, 35mm, f/1.4, 8k, hyperreal, artstation, unreal engine"). Pick the terms that matter and describe them in context.

## Prompting for Image Creation

Creation prompts describe a scene that does not yet exist. The prompt is the model's only input, so it must be comprehensive. Write it as a coherent paragraph that reads like a passage in a novel, not a comma-separated tag list.

Always specify a style. Without one the model picks inconsistently according to innate bias, leading to generic-looking output. Style affects design language, not just rendering: specifying "painterly" or "3D render" changes what the model puts in the scene.

Order prompt elements by importance:
1. Subject and action: who or what, doing what. Concrete beats abstract: "an elderly potter inspecting a cracked raku bowl" over "a person with a bowl."
2. Composition and framing: shot type, camera angle, focal point. Photography language works: "close-up portrait," "wide establishing shot from a low angle," "over-the-shoulder."
3. Setting and environment: where, with grounding details. "A rain-soaked Tokyo alley at night, neon signs reflecting in puddles" over "a city street."
4. Lighting: the strongest lever for mood and realism. Always specify: "warm golden-hour sidelight," "harsh overhead fluorescent," "diffused overcast daylight," "dramatic Rembrandt lighting."
5. Style and medium: photorealistic, watercolor, cel-shaded, oil painting, etc. For photorealism, camera and lens specs force photographic rendering: "shot on a Canon EOS R5 with an 85mm f/1.4 lens, shallow depth of field."
6. Mood and atmosphere: the emotional thread tying everything together: "serene and contemplative," "tense and claustrophobic."

Creation prompting anti-patterns:
- Vague prompts without style, medium, or composition instructions. The model defaults to generic output when instructions are ambiguous.
- Omitting lighting. Lighting is the single strongest lever for mood and realism. An unlit prompt leaves the model to guess, and it guesses blandly.

## Prompting for Image Transformation

Transformation prompts direct changes to an existing image, whether through image inputs or session continuation. The image already exists; the prompt specifies the delta, not the whole scene.

Write short, directive instructions focused on what to change. The model has the image (from the session or as an input). It needs directions, not a redescription. "Make the gate taller so it reaches the top of the wall" outperforms restating the entire scene with a taller gate.

Specify preservation boundaries. State what must stay the same alongside what must change. "Change only the sky to a warm sunset gradient. Keep the foreground buildings, street, people, and all shadows exactly as they are." Without explicit preservation instructions, the model may reinterpret unchanged areas.

When adding elements, describe the new element with enough detail to match the existing scene's style and lighting. "Add a black cat on the windowsill, lit consistently with the warm interior light, casting a soft shadow on the sill."

When removing elements, describe how to fill the gap. "Remove the trash can on the left. Fill the area to match the surrounding brick wall texture and concrete sidewalk."

Transformation prompting anti-patterns:
- Redescribing the entire scene. A full scene description signals creation, not transformation. The model may regenerate everything from scratch, losing the existing image's details.
- Omitting preservation boundaries. Without explicit instructions to keep elements unchanged, the model treats everything as open to reinterpretation.
- Stacking multiple changes in a single prompt. Each change adds cognitive load and increases drift. Decompose into one change per prompt and chain with session continuations.
- Relying on camera terminology alone for technical views. "Orthographic side view" sets intent but is not sufficient. Always reinforce with geometric constraints: what should be occluded, what surfaces should be visible or invisible, what should be symmetrical.

## Image Inputs

IMPORTANT: If images are provided, the model will try to use them, so they must always be paired with prompt instructions for how they are to be used. Input images not mentioned by the prompt become noise at best, and misdirection at worst.

Input images work together with the prompt instructions. Like the prompt, they are a multi-purpose tool for communicating visual concepts to the model. They are typically used for style reference or character consistency, but are by no means limited to those roles.

Image input patterns:
- Character consistency by providing images of the character
- Style consistency by providing reference images
- Style transfer by providing a target image and a style reference image
- Placing a character in an environment using both as reference inputs
- Combining features of two or more designs to create something new, often by extracting specific features of each
- Object removal by providing focused instructions for what to remove alongside specifications for how to fill the gap
- Object addition by providing instructions for how to add the new element with enough detail to match the existing scene's style and lighting
- Targeted element change by providing focused instructions for what to change and what to keep. Be specific about preservation boundaries.

Image input anti-patterns:
- Not providing usage instructions in the prompt. Always describe what the input images should be used for.
- Redescribing the entire image rather than giving instructions for how to transform the inputs. The prompt should specify what to change, not what the final image looks like.
- Not accounting for camera angle anchoring. Reference images anchor camera angle as well as subject. When the reference is a three-quarter view and the target is a side view, the model gravitates toward the reference's angle. The prompt must explicitly counteract the reference's camera with geometric constraints describing what should and should not be visible.

## Session Continuations

Continuing a session allows you to continue work on an image with the full context of all previous turns in the model's context window. This enables lossless iteration by describing what further changes to make.

Session continuations have the advantage of accumulated context. The model has seen every prior prompt and output in the chain, so it can make targeted changes without redescription. This makes sessions the preferred tool for iterative refinement where multiple targeted changes are spread across multiple turns.

Session patterns:
- Iterative refinement of a decent initial output.
- Multiple edits spread across multiple turns to maximize prompt adherence.
- Creating alternative camera angles of the same motif.

Session anti-patterns:
- Attempting to use a different model than the one that started the session. Use an image input approach instead.
- Continuing a session when a fresh image was desired. Session continuations are for iteration, not exploration.

## Workflow

Image generation costs money through the Gemini API. Two safeguards require explicit user permission before proceeding: never use the Pro model unless the user explicitly requests or approves it, and never generate multiple images in a single turn unless the user explicitly asks for batch generation.

Default to using Flash at 1K resolution as it is the cheapest option and still capable of most tasks. Recommend escalating to Pro when Flash repeatedly fails to produce adequate output even after adjusting the prompt. Pro is especially good for images that need to contain text elements, such as advertisements and memes. Similarly, only increase the resolution if the user wants to. It is best to explore at lower resolution and only scale up once you are confident the prompt works.

Choose an output filename that reflects the content and variant. Use the subject as the base name and append what distinguishes this image from its siblings: `farmhouse_twilight.png`, `creature_apose_back.png`. Avoid encoding full edit history into names.

### Creative collaboration

Users often do not know exactly what they want. Function as their creative collaborator: understand the purpose of the task, the audience the image should serve, and the story it should tell. Generating images to help visualize ideas, and asking focused questions about what the user likes and dislikes, are the primary tools for narrowing down direction.

Image generation is exploratory. The stochastic variance of model outputs means it will almost always take multiple attempts before the image is right. Each rejected image informs direction by revealing what works and what does not. This is an iterative process of navigating ideas through creative exploration of prompts, not a single well-aimed attempt.

When starting from scratch and the request is too vague to construct a strong prompt, ask focused questions about the missing elements. A vague prompt produces a vague image; the cost of one clarifying question is less than the cost of a wasted generation.

It is often easier to settle the broader concepts first, then gradually distill them into concrete details. This means working through the prompt elements in reverse:
1. Settle on mood and atmosphere.
2. Pick a style that fits the project.
3. Choose lighting that fits the style and atmosphere.
4. Determine a setting and environment that grounds the scene.
5. Find a composition and framing that reinforces the story.
6. Pin down the specific details of the subject and action.

### Generation loop

Every generation follows the same cycle: assess, compose, execute, diagnose. The value of each step increases as a session progresses, not decreases. The first generation is the one where you know the least; by the fifth, you have data about how the model handles this subject.

1. **Assess.** Determine what this generation needs to accomplish and how to approach it. On the first generation, this means understanding the user's goal. On subsequent generations, it means identifying what changed: the user gave feedback, the diagnosis found drift, the concept evolved. Then choose the approach: fresh generation, image inputs, session continuation, or a combination. The Workflow Patterns section has heuristics for when each applies.
2. **Compose.** Write the prompt, and optionally pick out reference materials. For fresh generation, write a full scene description following the element ordering described by the Prompting for Image Creation section. For session continuation, write a short directive about what to change. For image inputs, write instructions for how the inputs should be used or transformed.
3. **Verify.** Critique the prompt. Compare the prompt to this skill's guidance and the user's preferences to maximize the chance of success.
4. **Execute.** Run the banana command. Match the aspect ratio to the subject's proportions in the target view, not to convention or habit. A horizontal creature needs a landscape ratio even for a front view. Reconsider the ratio when the subject or composition changes, not only at the start of a project.
5. **Diagnose.** Load the generated image using the Read tool so you can see it. The user views images independently using their own image viewer. Diagnosis is mandatory on every generation.

LLM image analysis is unreliable for evaluating generation quality. Confirmation bias (seeing what the prompt described rather than what the image contains) is a persistent failure mode that self-awareness does not fix. The user's eyes are the authoritative evaluation tool. Hand diagnosis to the user through targeted questions.

Note any obvious observations in a brief sentence or two. Do not evaluate overall quality or declare success. Then ask the user multiple specific questions using AskUserQuestion, each targeting one concrete element. Never ask a single general question like "does this look good?"

For creation prompts, derive questions from the prompt structure hierarchy. Ask about each element that the prompt specified, prioritized by importance and risk:
- Subject and action: did the subject match? Is the pose or action correct?
- Composition and framing: is the camera angle and framing what was intended?
- Setting and environment: are the grounding details present?
- Lighting: does the lighting match what was specified?
- Style and medium: did the model render in the intended style?
- Novel or complex elements: ask about each individually. These carry the highest risk of being dropped or distorted.

For transformation prompts, ask about both sides of the change:
- Did the requested change land? In the right direction, by the right amount?
- Did preserved elements hold? Ask about specific elements that should not have changed.
- If continuing a session, ask whether elements that were correct in the previous version have drifted.

Always ask about known weak areas when relevant: faces, hands, fine spatial relationships, thin overlapping geometry.

Use the user's answers to plan the next cycle. Their feedback is the diagnosis. Translate their observations into prompt-level causes: if a detail was dropped, check prompt position and novelty load; if the change went the wrong direction, the instruction may have been ambiguous; if elements regressed, the session chain may be too long.

Workflow patterns:
- Decompose multi-edit requests into one change per generation. When the user asks for multiple changes, list them, then execute each as a separate session continuation. One change per turn is the default. The only exception is when two changes are physically entangled (e.g., removing an object and filling the gap it leaves).
- Use session continuation for targeted edits to an existing image that is close to what the user wants.
- Use image inputs when you need to bring in visual references: style sheets, character references, environment references, or a previous output that needs to be recontextualized rather than iterated on.
- Use fresh generation when exploring new concepts or when session continuation has failed to fix the same problem twice. Carry forward the prompt language that worked; drop the session history.
- If Flash fails to capture one or more details across multiple attempts, recommend escalating to Pro. "Fails to capture" includes missing, distorted, or reinterpreted details, not just absent ones. Count across all attempts including fresh starts, not just within a single session chain. Do not compensate with emphatic prompt language; if the model misrenders a clearly stated detail, restating it louder will not help.
- If fine details will not resolve at 1K, recommend higher resolution as an alternative or complement to a model switch.
- If exploration keeps producing similar outputs despite different prompts, check whether input images are anchoring the model.
- If a session chain is drifting after 3-4 turns, start fresh with a revised prompt rather than continuing to patch.
- For technical views (orthographic, turnaround sheets), reinforce camera terminology with geometric occlusion constraints. Describe what should be hidden and what should be visible. Start with Pro if precise camera control matters.

Workflow anti-patterns:
- Monolithic edit prompts. Combining multiple changes into a single prompt forces the model to track multiple objectives simultaneously, increasing drift and reducing adherence to each individual change.
- Providing image inputs when a fresh, unbiased image was requested. The model anchors to provided images, reducing creative variance.
- Stacking multiple changes in a single session continuation instead of spreading them across turns.
- Using session continuation when the image needs to change substantially. Sessions constrain what the model can produce; that constraint is harmful when the goal is divergence.
- Skipping diagnosis. Every generation must be evaluated before deciding what to do next.
- Evaluating image quality yourself instead of asking the user. LLM image analysis is unreliable; the user's eyes are authoritative.
- Asking vague questions ("does this look good?") instead of targeting specific prompt elements.
- Criticality decay: asking fewer or less specific questions on successive generations.
- Fixing unconfirmed problems. When the user reports an issue, do not assume you know the cause and immediately generate a fix. Ask the user what specifically is wrong before iterating. LLM image analysis is unreliable for diagnosing spatial and compositional problems; a wrong self-diagnosis wastes generations fixing the wrong thing.