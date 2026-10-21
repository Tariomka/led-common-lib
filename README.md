# Summary

Library with common packages for [Desktop Led Controller](https://github.com/Tariomka/desktop-led-controller), [RPI Led Communication](https://github.com/Tariomka/rpi-led-communication) and the STM32 LED cube firmware (`stm32-led-cube`).

# Packages

| Package | Content |
|---|---|
| `pkg/lightshow` | iCubeSmart show engine: player, key functions, script VM |
| `pkg/lightshow/blob` | Show data parser and writer |
| `pkg/lightshow/render` | Voxel canvas and drawing primitives |
| `pkg/lightshow/keyframe` | Shows described in Go code |
| `pkg/communication/uart` | Cube UART protocol: console commands, log line parser, stream reader, show upload client |
| `pkg/communication/tcp` | Desktop ↔ Pico messages `[version][type][length u16 BE][payload]` |
| `pkg/logger` | Colored `slog` handler (host and Pico only) |
| `pkg/common` | Shared errors and helpers |

The `lightshow` and `communication` packages build with TinyGo: they avoid `fmt`, `reflect`, `encoding/json`, goroutines and Unicode tables.

# Requirements

1. Go v1.27.2 or above: [download link](https://go.dev/doc/install).