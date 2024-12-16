module github.com/solarlune/tetra3d

go 1.22.0

toolchain go1.23.0

require (
	github.com/hajimehoshi/ebiten/v2 v2.8.0-alpha.3.0.20240826172230-42209606b1cf
	github.com/qmuntal/gltf v0.27.0
	github.com/tanema/gween v0.0.0-20221212145351-621cc8a459d1
	golang.org/x/image v0.20.0
)

require (
	github.com/ebitengine/gomobile v0.0.0-20240911145611-4856209ac325 // indirect
	github.com/ebitengine/hideconsole v1.0.0 // indirect
	github.com/ebitengine/purego v0.8.0-alpha.5 // indirect
	github.com/jezek/xgb v1.1.1 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
	golang.org/x/text v0.18.0 // indirect
)

replace github.com/hajimehoshi/ebiten/v2 => ../../libraries/ebiten
