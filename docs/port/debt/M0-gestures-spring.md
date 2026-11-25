# M0-gestures-spring debt

- spring.go: `springResolveOf` returns `any` (float32 -> float32, bool -> AnimationPhase); the stage that ports the spring animation element (animation.rs consumer) should type its output when it wires `SpringAnimation[T]`.
- spring.go / gestures.go: f32 maths go through float64 `math` calls cast back to float32; results can differ from Rust's f32 intrinsics in the last ulp.
