module github.com/lewtec/eletrocromo/examples/astro

go 1.25.4

require (
	github.com/lewtec/eletrocromo v0.0.0
	orvalho v0.0.0
)

require (
	github.com/dlclark/regexp2/v2 v2.2.1 // indirect
	github.com/dop251/goja v0.0.0-20260701091749-b07b74453ea9 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/google/pprof v0.0.0-20230207041349-798e818bf904 // indirect
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/text v0.32.0 // indirect
)

replace github.com/lewtec/eletrocromo => ../..

// Local checkout of https://github.com/lucasew/orvalho (module path is "orvalho").
// Adjust if your layout differs from WORKSPACE/{LEWTEC/eletrocromo,OPENSOURCE-own/orvalho}.
replace orvalho => ../../../../OPENSOURCE-own/orvalho
