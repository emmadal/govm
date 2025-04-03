# govm - Go Version Manager

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GitHub release](https://img.shields.io/github/release/emmadal/govm.svg)](https://github.com/emmadal/govm/releases)
[![GitHub issues](https://img.shields.io/github/issues/emmadal/govm.svg)](https://github.com/emmadal/govm/issues)
[![GitHub stars](https://img.shields.io/github/stars/emmadal/govm.svg)](https://github.com/emmadal/govm/stargazers)

**govm** is a version manager for Go, allowing you to easily install and switch between multiple versions of Go.

## Installation

You can install `govm` using the following command:

```bash
curl -o- https://raw.githubusercontent.com/emmadal/govm/main/install.sh | bash
```

or

```bash
wget -qO- https://raw.githubusercontent.com/emmadal/govm/main/install.sh | bash
```

## Usage

### Installing a Go version

```bash
govm install <version>
```

### Using a specific Go version

```bash
govm use <version>
```

### Listing installed Go versions

```bash
govm list
```

### Removing a Go version

```bash
govm rm <version>
```

### Updating govm

You can update `govm` to the latest version using the following command:

```bash
govm update
```

## Features

- Easy installation and management of multiple Go versions
- Automatic switching between Go versions
- Support for Linux and macOS (Windows support coming soon)
- Simple command-line interface

## Requirements

- Bash 3.2 or later
- A POSIX-compliant system (Linux, macOS, etc.)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

If you encounter any issues or have questions, please file an issue on the [GitHub repository](https://github.com/emmadal/govm/issues).

