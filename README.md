# mimic - testing terminal interactions

[![Apache 2.0 License](https://img.shields.io/badge/License-Apache%202.0-blue)](./LICENSE)
![Go Version](https://img.shields.io/github/go-mod/go-version/jimschubert/mimic)
[![Go Build](https://github.com/jimschubert/mimic/actions/workflows/build.yml/badge.svg)](https://github.com/jimschubert/mimic/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jimschubert/mimic)](https://goreportcard.com/report/github.com/jimschubert/mimic)

Mimic aims to provide a simple and clean contract for interacting with terminal contents while unit testing.

The idea arose from use of [go-survey/survey](https://github.com/go-survey/survey) and interest in using something like [charmbracelet/vhs](https://github.com/charmbracelet/vhs) for integration testing of a CLI.
A well-crafted applications can use mimic for integration testing of CLIs with full capability to mock dependencies. This differs from integration testing with tools like vhc because mimic runs in-process.

A goal of this project is to simplify testing CLIs using `survey` by providing a test suite which reduces test boilerplate and adheres to testify's suite interface(s).

Mimic also provides commands for:

* checking existence of strings in the output buffer
* checking existence of patterns in the output buffer
* checking formatted output according to pseudoterminal configuration

## Usage

For example usage, see [suite_test.go](./suite/suite_test.go) in this repository.

## License

This project is [licensed](./LICENSE) under Apache 2.0.
