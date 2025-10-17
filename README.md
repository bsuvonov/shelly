# Shelly

Simple LLM-powered tool for shell command generation and debugging via natural language

## Dependencies

### Required
- **Go 1.24.0+** - To build the project
- **OpenRouter API key** - Get one at https://openrouter.ai/
- **Clipboard utility** (one of):
  - `xclip` (recommended)
  - `xsel`
  - `wl-clipboard` (for Wayland)

## Usage

### 1. Debug Mode (`-d` / `--debug`)

Analyze broken commands and get fixes.

**Example 1: Fix case-insensitive search**
```bash
$ echo 'grep pattern file.txt' | shelly -d "need case-insensitive search"

You're missing the -i flag for case-insensitive matching.

1. grep -i pattern file.txt
2. grep --ignore-case pattern file.txt
3. grep -i "pattern" file.txt

Select a command (1-3): 1
Command copied to clipboard: grep -i pattern file.txt
```

**Example 2: Debug find command**
```bash
$ echo 'find . -name *.txt' | shelly -d "not finding all text files"

The wildcard *.txt needs quotes to prevent shell expansion.

1. find . -name "*.txt"
2. find . -name '*.txt'
3. find . -type f -name "*.txt"

Select a command (1-3): 3
Command copied to clipboard: find . -type f -name "*.txt"
```

### 2. Command Mode (`-c` / `--command`)

Generate command suggestions from natural language.

**Example 1: Search files recursively**
```bash
$ shelly -c "search for 'TODO' in all Python files"

1. grep -r "TODO" --include="*.py"
2. find . -name "*.py" -exec grep "TODO" {} +
3. grep -rn "TODO" --include="*.py" .

Select a command (1-3): 1
Command copied to clipboard: grep -r "TODO" --include="*.py"
```

**Example 2: Find large files**
```bash
$ shelly -c "find files larger than 100MB in current directory"

1. find . -maxdepth 1 -type f -size +100M
2. find . -type f -size +100M
3. du -h . | grep -E '^[0-9]+M'

Select a command (1-3): 1
Command copied to clipboard: find . -maxdepth 1 -type f -size +100M
```

### 3. Question Mode (`-q` / `--question`)

Get quick answers with examples.

**Example 1: Understand command options**
```bash
$ shelly -q "what does find -exec do?"

The -exec option executes a command on each file found.

Syntax: find [path] -exec command {} \;

Examples:
find . -name "*.log" -exec rm {} \;           # Delete files
find . -type f -exec chmod 644 {} \;          # Change permissions
find . -name "*.txt" -exec grep "TODO" {} +   # Search in files

The {} is replaced with the filename. Use \; to end or + to pass multiple files at once.
```

**Example 2: Ask with context (piping)**
```bash
$ tail -n 10 error.log | shelly -q "what might be causing this error?"

Based on the context, the error appears to be related to...

[Provides analysis based on the piped content]
```

## API Usage

Shelly uses DeepSeek via OpenRouter API with the free tier model: `deepseek/deepseek-chat-v3.1:free`.
