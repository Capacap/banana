# banana

An image generation and editing skill for AI coding agents. Ships as a single static binary with no runtime dependencies, so any agent harness that can execute a shell command can use it.

The CLI wraps Google's Gemini native image generation API. It was built in Go specifically for portability: cross-compiled static binaries run on any supported platform without requiring the end user to install Go, Python, Node, or anything else. The binary produces no config files, caches, or hidden state. The only files it writes are the output image and accompanying session JSON, both to the same target location, zero behind-the-scenes clutter.

## What's in the box

A release zip extracts to a single `banana/` directory containing three files:

- **`banana`** (or `banana.exe`) - CLI binary that handles Gemini API calls, session persistence, and image I/O.
- **`SKILL.md`** - Skill file that teaches the agent how to use the binary: prompt construction, model selection, iterative refinement, and safety filter strategies.
- **`prompting-reference.md`** - Extended reference on prompt techniques, image types, and model behavior.

Claude Code is the first-class target, but nothing here is Claude-specific. Any agent framework that supports tool/skill definitions and shell execution (Codex, etc.) can use this with appropriate skill file adaptation.

## Setup

The agent's host machine needs a `GOOGLE_API_KEY` environment variable. Get one from [Google AI Studio](https://aistudio.google.com) using the "Get API key" button.

```
export GOOGLE_API_KEY=your-key-here
```

### Install from a release zip

1. Download the zip for your platform from [releases](https://github.com/Capacap/banana/releases).
2. Extract it. You'll get a `banana/` directory containing the binary and skill files.
3. Move that directory into your Claude Code skills folder:

```
mv banana/ ~/.claude/skills/banana/
```

No PATH modification is needed. The agent resolves the binary from the skill directory path provided at invocation.

### Build from source

```
make build      # builds for the current platform
make release    # cross-compiles six platform zips into dist/
```

`make release` produces the same zip structure described above for six platforms: linux/darwin/windows, amd64/arm64.

### Uninstall

Delete `~/.claude/skills/banana/` (or wherever you placed it). Optionally remove `GOOGLE_API_KEY` from your environment. That's it.

## CLI reference

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
| `-r` | no | Aspect ratio (default `1:1`). Options: `1:1`, `2:3`, `3:2`, `3:4`, `4:3`, `9:16`, `16:9`, `21:9` |
| `-z` | no | Output resolution: `1K`, `2K`, or `4K` (requires `-m pro`) |
| `-f` | no | Overwrite output and session files if they already exist |

Pass `-i` multiple times to provide several reference images. Flash 2.5 supports up to 3 input images; Flash 3.1 and Pro support up to 14. Each input file must be under 7 MB. The CLI checks for `GOOGLE_API_KEY` at startup and exits with a clear error if it is missing. Run `banana help` to see usage information.

### Sessions

Every generation produces a session file by replacing the output file's extension with `.session.json` (e.g., `out.png` produces `out.session.json`). The session file records the resolved model name (e.g., `flash-3.1`, not the bare alias `flash`) and conversation history. Passing it with `-s` continues the conversation. The session always saves alongside `-o`, preserving the source session for rewind and branching. The CLI validates that the session's model matches the current `-m` flag to prevent accidental cross-model continuation.

Without `-f`, the CLI refuses to write if the output or session file already exists. This includes the case where `-s` points to the same session file that `-o` would produce (e.g., `-o cat.png -s cat.session.json`). With `-f`, both the output and the session file are overwritten, including the source session if it collides.

### Metadata

Generated PNGs carry embedded metadata in a `tEXt` chunk recording the schema version, model name and ID, aspect ratio, output size (pro only), input file names, session source, timestamp, and prompt history. The `meta` subcommand reads and displays this data.

```
banana meta <image.png>
```

Example output:

```
version:   1
model:     flash-3.1 (gemini-3.1-flash-image-preview)
ratio:     1:1
timestamp: 2026-02-26T15:04:05Z

prompts:
  [1] user: a cat wearing a red hat
```

Fields like `size`, `inputs`, and `session` appear when applicable (e.g., when using `-z`, `-i`, or `-s`). Output is always PNG since the Gemini API returns PNG data.

### Cleanup

Session files accumulate during iterative work. The `clean` subcommand scans a directory (non-recursively) for session files, validates them, and reports what it finds.

```
banana clean <directory>        # dry run: list files and sizes
banana clean -f <directory>     # delete validated session files
```

Without `-f`, nothing is deleted. The listing shows file path, model, turn count, and size for each file. Files that fail validation (corrupt JSON, unknown structure) are skipped with a warning and never deleted.

### Cost estimation

The `cost` subcommand estimates API cost from session files using published Gemini pricing.

```
banana cost <session-file>      # single session breakdown
banana cost <directory>         # summarize all sessions in a directory
```

Single-file output shows model, turn count, token usage with costs, image count, and total. Directory output lists each session with a per-session cost and a grand total. Sessions created before usage tracking (or with unrecognized models) show partial data.

Pricing is based on published rates as of 2026-02-26. Image output costs assume 1K resolution. The estimate covers input tokens, output tokens, and generated images.

### Models

| Model | Flag | Max inputs | Resolution control |
|-------|------|------------|--------------------|
| Gemini 3.1 Flash Image Preview | `-m flash` (default), `-m flash-3.1` | 14 | No |
| Gemini 2.5 Flash Image | `-m flash-2.5` | 3 | No |
| Gemini 3 Pro Image Preview | `-m pro`, `-m pro-3.0` | 14 | Yes (`-z 1K\|2K\|4K`) |

`flash` and `pro` are aliases that always point to the latest version in their family. Use a pinned name (`flash-2.5`, `flash-3.1`, `pro-3.0`) to lock to a specific model version. This matters when a model's particular rendering style is desirable, since different versions have different artistic tendencies.

Flash is the default. Pro is selected when the task requires text rendering or high resolution output (`-z`).

## Repository structure

```
src/         CLI source (Go)
skill/       skill files (copied into release zips)
dist/        release zips (generated by make release)
Makefile     build and release targets
```
