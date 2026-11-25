import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const wasmPath = process.argv[2] ?? process.env.GPUI_WEB_WASM;
const workers = Number(process.argv[3] ?? 8);
assert.ok(wasmPath, "usage: node test_queue.mjs <gpui-web-examples.wasm> [worker-count]");
assert.ok(Number.isInteger(workers) && workers > 0);

const goRoot = execFileSync("go", ["env", "GOROOT"], { encoding: "utf8" }).trim();
const require = createRequire(import.meta.url);
globalThis.require = require;
globalThis.fs = fs;
globalThis.path = path;
(0, eval)(fs.readFileSync(path.join(goRoot, "lib/wasm/wasm_exec.js"), "utf8"));

const go = new globalThis.Go();
go.argv = ["js", "queue-test"];
go.env = {};
const { instance } = await WebAssembly.instantiate(fs.readFileSync(wasmPath), go.importObject);
go.run(instance);
const delay = (milliseconds) => new Promise((resolve) => setTimeout(resolve, milliseconds));
for (let attempt = 0; attempt < 100 && !globalThis.gpuiWebTests; attempt++) await delay(10);
const fixture = globalThis.gpuiWebTests;
assert.ok(fixture?.new_queue_stress, "queue-test exports are missing");

const rounds = 128;
const batch = 4096;
const stress = fixture.new_queue_stress(workers, rounds * batch);
assert.equal(typeof stress, "object", String(stress));
while (!stress.ready()) await delay(1);
for (let round = 0; round < rounds; round++) {
    assert.equal(stress.send_batch(batch), undefined);
    for (let drained = stress.drain(); drained !== (round + 1) * batch; drained = stress.drain()) {
        assert.equal(typeof drained, "number", drained);
        await delay(1);
    }
    await delay(1);
}
assert.equal(stress.stop(), undefined);
while (!stress.stopped()) await delay(1);
console.log("PASS: queue contention", { workers, items: rounds * batch, rounds });
process.exit(0);
