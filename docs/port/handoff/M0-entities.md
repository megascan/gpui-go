# M0-entities handoff

- stubs.go: the `App` stub now has the field `entities *entityMap` (upstream `App::entities`); `Entity.Read` uses it. M0-app-core keeps the field and builds it with `newEntityMap()`.
- App primitives should use the erased entity-map API (docs/api-contracts.md 4.1): `newEntity` = `reserveErased` + build + `EntityCreated` + `insertErased`; `updateEntity` = `leaseErased` / `endLeaseErased`; `readEntity` = `readErased`; the flush loop calls `takeDropped()` and runs `Drop(cx)` on states implementing it.
- Test harness (M0-app-contexts / M0-test-contexts): end each run with `cx.entities.report()` after quit, task drain and GC (DESIGN 6.11).
- M0-test-contexts (or M0-exit, which has fix authority, since subscription_test.go is not in M0-test-contexts' owns): replace the four `t.Skip` bodies in subscription_test.go in place, keeping the names; never add same-named tests elsewhere, which would break the root package build (`TestSubscription_UnsubscribeDuringCallbackWithInsert`, `CallbackDroppedByEarlierCallbackDoesNotFire`, `SelfDropDuringCallback`, `SubscriptionDrop`) with the TestApp ports and flip their rows in core-app.md.
- `paritycheck -lint` currently fails only outside this stage: internal/linux/wl/display.go:285 (LockOSThread) and internal/linux/xcbshim/setup.go:5 (comment).
- `(Entity[T]).IntoAnyElement` stays in stubs.go for M0-element.
