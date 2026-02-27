# CLI Reference

## Generation Command

```
banana -p <prompt> -o <output> [-i <input>...] [-s <session>] [-m model] [-r <ratio>] [-z 1K|2K|4K] [-f]
```

| Flag | Required | Description |
|------|----------|-------------|
| `-p` | yes | Text prompt |
| `-o` | yes | Output PNG file path (must end in `.png`) |
| `-i` | no | Input image for editing/reference (repeatable; supports png, jpg/jpeg, webp, heic, heif) |
| `-s` | no | Session file to continue from |
| `-m` | no | Model: `flash` (default), `pro`, `flash-2.5`, `flash-3.1`, `pro-3.0` |
| `-r` | no | Aspect ratio (default `1:1`): `1:1`, `2:3`, `3:2`, `3:4`, `4:3`, `9:16`, `16:9`, `21:9` |
| `-z` | no | Output size: `1K`, `2K`, or `4K` (`flash-3.1`, `pro-3.0` only) |
| `-f` | no | Overwrite output and session files if they already exist |

Output is always PNG. The Gemini API returns PNG data; other output formats are not supported.

### Input images

Pass `-i` multiple times for multiple reference images. Each file must be under 7 MB. Model-specific limits:

- Flash 2.5: up to 3 input images
- Flash 3.1: up to 14
- Pro: up to 14

The CLI validates file existence, MIME type, and size before making the API call. If the input count exceeds the model limit, the error message suggests switching to a model with a higher cap.

### Aspect ratio

The `-r` flag sets the aspect ratio for the generated image. The default is `1:1`. All supported ratios: `1:1`, `2:3`, `3:2`, `3:4`, `4:3`, `9:16`, `16:9`, `21:9`.

### Output size

The `-z` flag controls output resolution. Only available on models with resolution support (`flash-3.1`, `pro-3.0`). Options: `1K`, `2K`, `4K`. When omitted, the API chooses the default (1K).

## Sessions

Every generation produces a session file by replacing the output file's extension with `.session.json` (e.g., `cat.png` produces `cat.session.json`). The session records the resolved model name (e.g., `flash-3.1`, not the bare alias `flash`), output size, conversation history, and token usage.

### Loading and saving

The `-s` flag is read-only: it loads history from the specified session file but never writes back to it. The new session always saves alongside `-o`. This preserves the source session for rewind and branching. Without `-f`, the CLI refuses to write if the derived session path collides with the `-s` source. With `-f`, the source session is overwritten.

### Model cross-check

The CLI validates that the session's model matches the current `-m` flag to prevent accidental cross-model continuation. Legacy sessions (created before model tracking was added) skip this check. Bare aliases are resolved before comparison, so `-m flash` and `-m flash-3.1` are equivalent when `flash` aliases to `flash-3.1`.

### Branching and rewind

Every generation writes its own session file next to its output, and the source session is never modified. To branch from an earlier point, pass that point's session file with `-s` and a new output path with `-o`. The earlier session file remains intact, and the new generation starts a new branch from that history.

## Subcommands

### meta

Show metadata embedded in a generated PNG.

```
banana meta <image.png>
```

Output:

```
version:   1
model:     flash-3.1 (gemini-3.1-flash-image-preview)
ratio:     1:1
size:      2K
timestamp: 2026-02-26T15:04:05Z
inputs:    reference.png
session:   cat.session.json

prompts:
  [1] user: a cat wearing a red hat
  [2] user: make the hat blue
```

Fields like `size`, `inputs`, and `session` appear only when applicable (when `-z`, `-i`, or `-s` were used). Non-PNG files and PNGs without banana metadata produce distinct error messages.

### clean

Find and remove session files from a directory. Scans non-recursively.

```
banana clean <directory>        # dry run: list files with details
banana clean -f <directory>     # delete validated session files
```

Without `-f`, nothing is deleted. Each file is listed with its path, model, turn count, and size. Files that fail validation (corrupt JSON, unknown structure) are skipped with a warning and never deleted.

### cost

Estimate API cost from session files using published Gemini pricing.

```
banana cost <session-file>      # single session breakdown
banana cost <directory>         # summarize all sessions in a directory
```

Single-file output shows model, turn count, token usage with per-category costs, image count with per-image cost at the recorded size, and total. Directory output lists each session on one line with model, size, turn count, image count, and cost, followed by a grand total.

Sessions without usage data (created before tracking) or with unrecognized models show partial information. Image costs use the session's recorded output size; legacy sessions without size data are priced at 1K.

### help

```
banana help
```

Prints usage information showing all flags and subcommands.

## Models

| Model | Flag | API ID | Max inputs | Resolution control |
|-------|------|--------|------------|--------------------|
| Gemini 3.1 Flash Image Preview | `-m flash` (default), `-m flash-3.1` | `gemini-3.1-flash-image-preview` | 14 | Yes (`-z 1K\|2K\|4K`) |
| Gemini 2.5 Flash Image | `-m flash-2.5` | `gemini-2.5-flash-image` | 3 | No |
| Gemini 3 Pro Image Preview | `-m pro`, `-m pro-3.0` | `gemini-3-pro-image-preview` | 14 | Yes (`-z 1K\|2K\|4K`) |

### Aliases vs pinned names

`flash` and `pro` are bare aliases that resolve to the latest version in their family (`flash` → `flash-3.1`, `pro` → `pro-3.0`). Use a pinned name (`flash-2.5`, `flash-3.1`, `pro-3.0`) to lock to a specific model version. This matters when a model's particular rendering style is desirable, since different versions have different artistic tendencies. Sessions store the resolved pinned name, not the alias.

### Pricing

Approximate per-generation costs (as of 2026-02-26):

| Model | Image cost (1K) | Image cost (2K) | Image cost (4K) |
|-------|-----------------|-----------------|-----------------|
| Flash 2.5 | $0.039 | — | — |
| Flash 3.1 | $0.067 | $0.101 | $0.151 |
| Pro 3.0 | $0.134 | $0.134 | $0.240 |

These are per-image output costs. Input and output token costs are additional but typically small relative to the image cost. Use `banana cost` to get exact estimates from session files.

## Environment

The CLI requires `GOOGLE_API_KEY` to be set. It checks at startup and exits with a clear error if missing.

The binary is self-contained: no config files, caches, or hidden state. The only files it writes are the output PNG and the accompanying session JSON, both to the path specified by `-o`.
