# Research: a small model to drive Oriel's tools

> Status: **research / planning only — no implementation.** Question being
> explored: what's the smallest model that can reliably understand natural
> language, call our ~19 Docker/Colima tools (over MCP), and do a web search to
> look up errors — and is it worth *training* our own tiny model (standalone, or
> a "handoff" model a bigger model delegates the Docker parts to)?

Snapshot of the landscape as of mid-2026. Numbers drift; treat as a starting
point, not gospel.

## Why our case is easier than the benchmarks suggest

The headline benchmark is the Berkeley Function-Calling Leaderboard (BFCL),
which tests hundreds of unfamiliar APIs and hard multi-turn chains. **Our task is
much narrower:** one domain, a *fixed* set of ~19 tools the model can be told
about up front, and the destructive ones already gated behind a human/grant
check. So a model only needs to be reliable on *our* tools, not on the open
world — which moves the viable size down and makes constrained decoding very
effective.

## Smallest model that's reliable *today* (no training)

| Tier | Models | Local RAM @ 4-bit | Verdict for an executor |
|---|---|---|---|
| **~8B (sweet spot)** | Llama-xLAM-2-8b-fc-r, Watt-tool-8B, ToolACE-8B, Hammer2.1-7b, Qwen3-8B | ~5 GB | **Reliable enough to execute real commands**, incl. the multi-turn "build → inspect → fix" loop and conditional web-search-on-error. xLAM-2-8b leads small-model *multi-turn* (~69% BFCL v3 — beats GPT-4o-in-FC-mode on that axis). |
| **~3B (floor)** | xLAM-2-3b-fc-r (65.7% BFCL), Hammer2.1-3b, Arch-Function-3B | ~2 GB | Usable for **single-step** calls with **strict schema validation + retries**. Multi-turn is shaky. |
| **<3B** | xLAM-2-1b, Qwen3-0.6B | <1 GB | **Not safe for execution.** Multi-turn collapses (1B ≈ 8% multi-turn); characteristic failures are hallucinated args, wrong tool, omission, bad JSON. |
| **~14B (ceiling for "a few GB")** | Qwen3-14B / Qwen2.5-14B | ~9 GB | Best NL-understanding + tool-calling in a Mac-friendly budget. |

Key facts:

- **Reliability floor for executing commands is ~7–8B**, and it has to be a model
  *explicitly trained for tool use* — a general model of any size without
  tool-call training emits malformed calls. **Function-calling specialists
  (xLAM, Hammer, Watt-tool, ToolACE, Arch-Function) buy reliability per
  parameter** far better than raw size.
- **The single biggest cheap win is constrained / grammar-guided decoding**
  (XGrammar, llguidance — already in vLLM/SGLang). It forces output to match the
  tool JSON schema: reported to lift a *generalist* from ~0% → ~52% tool
  accuracy, reach ~99.5% schema validity at 7B and ~96% even at 1B, **and it's
  faster, not slower.** This is the determinism lever — and it needs **no
  training**.
- **Quantization to 4-bit is fine for tool calling** (structurally constrained
  generation, not factual recall), but because tool calls are format-sensitive,
  prefer **Q5_K_M** if you can spare the few hundred MB; avoid sub-Q4.
- Web-search plumbing is easy (Brave Search API / Tavily / self-hosted SearXNG);
  the hard part is the model *orchestrating* search→read→retry, which again wants
  ≥7–8B.

**Practical answer:** a stock **8B function-calling model at Q5 (~5–6 GB)** —
Qwen3-8B as the general default, or an FC-specialist like Watt-tool-8B /
Llama-xLAM-2-8b-fc-r — is the smallest thing reliable enough to drive Oriel.
3B FC-specialists are a viable low-RAM fallback for single-step use.

## Should we *train* our own tiny model?

### (A) Standalone tiny Oriel model
**Feasible, not worth it now.** A general 4–8B model + our MCP tools +
constrained decoding already does this at ~zero R&D. Training buys marginal
accuracy on a fixed schema in exchange for a permanent *data + retrain treadmill*
(every tool change → regenerate data → retrain) and a real risk of **catastrophic
forgetting** — over-fitting to 19 tools can destroy the general web-search /
error-reasoning ability we also need.

### (B) "Handoff" model (big model delegates the Docker part)
**The more defensible idea, and a real 2026 pattern** (orchestrator-worker;
typed handoffs in OpenAI's Agents SDK; Anthropic sub-agents). *But* in practice
the "specialist" is almost always **a general small model pinned to a namespaced
toolset + constrained decoding — not custom weights.** The value is the
*architecture*, not bespoke training. And we've effectively already built the
substrate for it: **`oriel mcp` is exactly the namespaced, validated Docker
toolset a big orchestrator can hand off to.**

### When training would actually pay off
Only with high call volume (to amortize), hard latency limits, or a determinism
need — and determinism is solved by constrained decoding, not training. An OSS
tool with spiky, low volume hits none of these hard enough today.

### If we ever did it — rough cost
QLoRA on a 3B base, single RTX 4090, <16 h/run ($50–300); **3–8k verified
`instruction → tool_call` examples** generated APIGen-style from our tool defs
and **executed against a throwaway Colima sandbox** for ground truth. All-in
**~$500–2k and 2–4 weeks** of one engineer, dominated by the data pipeline +
eval harness — plus ongoing retrains.

## Recommendation / plan

1. **Now:** ship MCP for existing models — *done* (`oriel mcp`). Highest leverage,
   matches what the ecosystem actually does. Let users point Claude / GPT /
   Qwen3-8B / Llama at the 19 tools.
2. **Local-model path:** when the dormant provider seam (`/resolve`) gets a local
   backend, recommend a **stock 8B tool-calling model** and add **constrained
   decoding** for that path — 0% malformed calls, no training.
3. **Build the synthetic-data pipeline early *anyway*** — even if we never train,
   it doubles as our **eval / regression harness** for whichever model we ship
   (e.g. "does Qwen3-8B still pick the right tool on these 500 prompts?"), and is
   reusable if we ever do train.
4. **For a "handoff" product:** build the *architecture* (typed handoff into the
   namespaced MCP toolset), keep the specialist a general model.
5. **Revisit custom training only with telemetry** proving a concrete
   cost/latency/offline wall — and even then, fine-tune *on top of* constrained
   decoding, not instead of it.

## Sources

Tool-calling / small-model landscape:
- BFCL v4 leaderboard — https://gorilla.cs.berkeley.edu/leaderboard.html
- TinyLLM: SLMs for agentic tasks on edge (arXiv 2511.22138) — https://arxiv.org/abs/2511.22138
- Salesforce Llama-xLAM-2-8b-fc-r — https://huggingface.co/Salesforce/Llama-xLAM-2-8b-fc-r
- xLAM / APIGen-MT (arXiv 2504.03601) — https://arxiv.org/html/2504.03601v4
- Hammer / function masking (arXiv 2410.04587) — https://arxiv.org/abs/2410.04587
- ToolACE (arXiv 2409.00920) — https://arxiv.org/html/2409.00920v1
- watt-ai/watt-tool-8B — https://huggingface.co/watt-ai/watt-tool-8B
- Katanemo Arch-Function — https://github.com/katanemo/Arch-Function
- Best local models for tool calling 2026 — https://www.promptquorum.com/power-local-llm/best-local-models-tool-calling-2026

Training / handoff / constrained decoding:
- APIGen (arXiv 2406.18518) — https://arxiv.org/abs/2406.18518
- Fine-tuning for function calling with xLAM (HF cookbook) — https://huggingface.co/learn/cookbook/en/function_calling_fine_tuning_llms_on_xlam
- SLMs for efficient agentic tool calling (arXiv 2512.15943) — https://arxiv.org/abs/2512.15943
- Handoffs — OpenAI Agents SDK — https://openai.github.io/openai-agents-python/handoffs/
- Code execution with MCP (Anthropic) — https://www.anthropic.com/engineering/code-execution-with-mcp
- Scaling laws for forgetting when fine-tuning (arXiv 2401.05605) — https://arxiv.org/html/2401.05605v1
- llguidance (constrained decoding) — https://github.com/guidance-ai/llguidance
- Best web search APIs for AI 2026 (Brave) — https://brave.com/learn/best-search-api-2026/
