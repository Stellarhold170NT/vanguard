# stub-repo

Minimal fake project exercising the discovery pipeline end-to-end
(`internal/discovery`, task w2-03; corpus rules in `docs/test-strategy.md`).
`Scan("../../testdata/stub-repo")` from the discovery package must: detect
the stub adapter, parse the `.stub.json` files into a valid `ir.ApiSurface`,
report exactly one diagnostic (the deliberately broken file below), and end
clean — this fixture is the acceptance input of w2-03.

Layout:

| Path | Purpose |
|---|---|
| `src/orders.stub.json` | Valid stub service: 3 methods (GET+path param, POST+payload/body param, GET+page/size params), 2 types |
| `src/broken.stub.json` | Truncated JSON — must become exactly one diagnostic, never an abort (best-effort contract, charter §5.3) |
| `scratch/` | Ignored via `.vanguardignore` — must never be walked |
| `node_modules/` | Built-in excluded directory — must never be walked |

The `.stub.json` format mirrors the IR vocabulary (service, methods, types);
the stub adapter (`internal/discovery/stub.go`) documents every field.
