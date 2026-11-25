# M3-platform-benches debt

- bench allocation stats use runtime/metrics (`/gc/heap/allocs:objects`, `/memory/classes/heap/objects:bytes`) instead of bench_metrics::allocation_stats; always reported, never None (D-BENCHMETRICS).
- atlas_upload / scene_ordering / scene_finish drop bench_metrics secondary-metric printing (MetricReport.iter is unexported in root).
- BenchmarkInspectorRender_* and BenchmarkAtlasUpload_* skip unless built with -tags gpui_release (upstream asserts release); atlas also skips off darwin (upstream requires Metal; macOS platform not ported yet, so it currently always skips).
