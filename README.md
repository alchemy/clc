# clc

A command-line tool for converting between configuration file formats.

## Supported Formats

| Format | Extensions         |
|--------|--------------------|
| JSON   | `.json`            |
| JSON5  | `.json5`           |
| TOML   | `.toml`            |
| YAML   | `.yaml`, `.yml`    |
| INI    | `.ini`, `.cfg`, `.conf` |

## Installation

```
go install github.com/alchemy/clc@latest
```

## Usage

```
clc [flags] [file]
```

### Flags

| Flag              | Description                                              |
|-------------------|----------------------------------------------------------|
| `-f`, `--from`    | Source format. Auto-detected from file extension if omitted. Required when reading from stdin. |
| `-t`, `--to`      | Target format. Required.                                 |
| `-o`, `--output`  | Output file path. Defaults to stdout.                    |

### Examples

Convert a TOML file to YAML:

```
clc -t yaml config.toml
```

Convert JSON from stdin to TOML:

```
cat config.json | clc -f json -t toml
```

Write output to a file:

```
clc -t yaml -o config.yaml config.toml
```

Pipe between formats:

```
echo '{"port": 8080}' | clc -f json -t toml
```

## Limitations

INI is a flat format. When encoding to INI:

- Top-level scalar values are written to the default section.
- One level of nested maps is supported as INI sections.
- Deeper nesting or arrays will produce an error.
