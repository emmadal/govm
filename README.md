govm is a tool for managing multiple Go versions on macOS, Linux

## Installation

Follow the instructions for your operating system:

### macOS

```sh
brew install govm
```

### Linux

```sh
sudo apt install govm
```

### Windows

Not yet supported. We welcome contributions!

## Example

```sh
$ govm install 1.24.1
  Switched to go1.24.1. Run 'source ~/.zshrc' or restart your terminal to apply permanently.

$ govm use 1.24.1
  Switched to go1.24.1. Run 'source ~/.zshrc' or restart your terminal to apply permanently.

$ govm list
  go1.24.1

$ govm uninstall 1.24.1
  Uninstalled go1.24.1
```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.
