const route = window.location.pathname.replace(/\/+$/, "") || "/";
const gallery = document.getElementById("gallery");
const back = document.getElementById("back");
const status = document.getElementById("status");
const example = [...gallery.querySelectorAll("a[data-module]")].find(
    (link) => new URL(link.href).pathname === route,
);
const sharedBinary = new Set(["hello-web", "dynamic-fonts"]);

function showError(error) {
    console.error(error);
    status.hidden = false;
    status.textContent = `Unable to start example: ${error.message ?? error}`;
    status.setAttribute("role", "alert");
}

function loadWasmExec() {
    return new Promise((resolve, reject) => {
        if (globalThis.Go) {
            resolve();
            return;
        }
        const script = document.createElement("script");
        script.src = "/wasm_exec.js";
        script.onload = () => resolve();
        script.onerror = () => reject(new Error("wasm_exec.js is missing; copy it from $(go env GOROOT)/lib/wasm/ next to index.html"));
        document.head.append(script);
    });
}

async function startExample(module) {
    await loadWasmExec();
    const go = new globalThis.Go();
    go.argv = ["js", module];
    const url = sharedBinary.has(module) ? "/gpui-web-examples.wasm" : `/${module}.wasm`;
    const { instance } = await WebAssembly.instantiateStreaming(fetch(url), go.importObject);
    go.run(instance);
}

if (route !== "/") {
    gallery.hidden = true;
    back.hidden = false;
    status.hidden = false;

    if (!example) {
        showError(new Error(`Unknown example path "${route}". Choose an example from the gallery.`));
    } else {
        const title = example.querySelector("strong").textContent;
        document.title = `${title} — GPUI Web Examples`;
        status.textContent = `Loading ${title}…`;

        try {
            await startExample(example.dataset.module);
            status.hidden = true;
        } catch (error) {
            showError(error);
        }
    }
}
