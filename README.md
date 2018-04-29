# emersyx_i2t [![Build Status][build-img]][build-url] [![Go Report Card][gorep-img]][gorep-url] [![GoDoc][godoc-img]][godoc-url]

This is an emersyx processor which forwards messages between IRC channels and Telegram groups. It acts as a traditional
bridge between the two chat services.

## Usage

Source files in `emi2t` provide the implementation of the go plugin. The plugin can be built by running `make`. The
resulting `emi2t.so` file can then be used by emersyx core.

[build-img]: https://travis-ci.org/emersyx/emersyx_i2t.svg?branch=master
[build-url]: https://travis-ci.org/emersyx/emersyx_i2t
[gorep-img]: https://goreportcard.com/badge/github.com/emersyx/emersyx_i2t
[gorep-url]: https://goreportcard.com/report/github.com/emersyx/emersyx_i2t
[godoc-img]: https://godoc.org/emersyx.net/emersyx_i2t?status.svg
[godoc-url]: https://godoc.org/emersyx.net/emersyx_i2t
