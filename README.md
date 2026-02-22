# banana

An image generation and editing skill for AI coding agents. Ships as a single static binary with no runtime dependencies, so any agent harness that can execute a shell command can use it.

The CLI wraps Google's Gemini native image generation API. It was built in Go specifically for portability: cross-compiled static binaries run on any supported platform without requiring the end user to install Go, Python, Node, or anything else. The binary produces no config files, caches, or hidden state. The only files it writes are the output image and accompanying session JSON, both to the same target location, zero behind-the-scenes clutter.

## What's in the box

The skill has two parts:

- **`banana`** - A CLI binary that handles Gemini API calls, session persistence, and image I/O.
- **`skill/`** - Skill files (SKILL.md + prompting reference) that teach the agent how to use the binary effectively: prompt construction, model selection, iterative refinement, and safety filter strategies.

Claude Code is the first-class target, but nothing here is Claude-specific. Any agent framework that supports tool/skill definitions and shell execution (Codex, etc.) can use this with appropriate skill file adaptation.

## Setup

The agent's host machine needs a `GOOGLE_API_KEY` environment variable. Get one from [Google AI Studio](https://aistudio.google.com) using the "Get API key" button.

```
export GOOGLE_API_KEY=your-key-here
```

### From a release zip

Download the zip for your platform from releases. Each zip contains the `banana` binary and the skill files. The agent resolves the binary from the skill directory path provided at invocation, so no PATH modification is needed.

### From source

```
make build      # builds for the current platform
make release    # cross-compiles six platform zips into dist/
```

Platforms: linux/darwin/windows, amd64/arm64.

### Uninstall

Delete the skill directory. Optionally remove `GOOGLE_API_KEY` from your environment. That's it.

## CLI reference

```
banana -p <prompt> -o <output> [-i <input>...] [-s <session>] [-m flash|pro] [-r <ratio>] [-z 1k|2k|4k] [-f]
```

| Flag | Required | Description |
|------|----------|-------------|
| `-p` | yes | Text prompt |
| `-o` | yes | Output file path (png, jpg, webp) |
| `-i` | no | Input image for editing/reference (repeatable) |
| `-s` | no | Session file to continue from |
| `-m` | no | Model: `flash` (default) or `pro` |
| `-r` | no | Aspect ratio (default `1:1`). Options: `1:1`, `2:3`, `3:2`, `3:4`, `4:3`, `9:16`, `16:9`, `21:9` |
| `-z` | no | Output resolution: `1k`, `2k`, or `4k` (requires `-m pro`) |
| `-f` | no | Overwrite output file if it already exists |

Pass `-i` multiple times to provide several reference images. Flash supports up to 3, Pro up to 14. Each input file must be under 7 MB.

### Sessions

Every generation produces a session file alongside the output (e.g., `out.png` creates `out.session.json`). The session file records the model name and conversation history. Passing it with `-s` continues the conversation and updates the session file in place. The CLI validates that the session's model matches the current `-m` flag to prevent accidental cross-model continuation.

### Models

| Model | Flag | Speed | Resolution control |
|-------|------|-------|--------------------|
| Gemini 2.5 Flash Image | `-m flash` | ~4s | No (1K only) |
| Gemini 3 Pro Image Preview | `-m pro` | ~8-12s | Yes (`-z 1k\|2k\|4k`) |

Flash is the default. Pro is selected when the task requires text rendering, high resolution, or more than 3 reference images.

## Project structure

```
src/         CLI source (Go)
skill/       agent skill files
dist/        release zips (generated)
Makefile     build and release targets
```
