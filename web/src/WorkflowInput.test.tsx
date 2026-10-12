// @vitest-environment jsdom
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, expect, it, vi } from "vitest";
import { WorkflowInput } from "./WorkflowInput";
import type { Workflow } from "./types";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
const roots: ReturnType<typeof createRoot>[] = [];
afterEach(() => {
  for (const root of roots) act(() => root.unmount());
  roots.length = 0;
  document.body.innerHTML = "";
});
function workflow(
  properties: Record<string, unknown>,
  required: string[],
  revision = 1,
): Workflow {
  return {
    id: "workflow",
    revision,
    name: "Input regression",
    definition: {
      schemaVersion: 1,
      nodes: [
        {
          id: "source",
          kind: "input",
          params: {
            inputSchema: {
              type: "object",
              properties,
              required,
              additionalProperties: false,
            },
          },
        },
      ],
      edges: [],
    },
  };
}
async function mount(wf: Workflow) {
  const el = document.createElement("div");
  document.body.append(el);
  const root = createRoot(el);
  roots.push(root);
  const submitted = vi.fn();
  const render = async (next: Workflow) =>
    act(async () =>
      root.render(
        <WorkflowInput workflow={next} onSubmit={submitted}>
          {() => <button>Run</button>}
        </WorkflowInput>,
      ),
    );
  await render(wf);
  return { el, submitted, render };
}
async function enter(
  element: HTMLInputElement | HTMLTextAreaElement,
  value: string,
) {
  await act(async () => {
    element.value = value;
    element.dispatchEvent(new Event("input", { bubbles: true }));
  });
}
async function submit(el: HTMLElement) {
  await act(async () =>
    el
      .querySelector("form")!
      .dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })),
  );
}
it("submits a different field name and untouched required boolean without a hidden description", async () => {
  const { el, submitted } = await mount(
    workflow({ message: { type: "string" }, approved: { type: "boolean" } }, [
      "message",
      "approved",
    ]),
  );
  await enter(el.querySelector("textarea")!, "A message");
  await submit(el);
  expect(submitted).toHaveBeenLastCalledWith({
    message: "A message",
    approved: false,
  });
});
it("clears old values when the workflow or its revision changes", async () => {
  const { el, submitted, render } = await mount(
    workflow({ description: { type: "string" } }, ["description"]),
  );
  await enter(el.querySelector("textarea")!, "Old value");
  await render(workflow({ message: { type: "string" } }, ["message"], 2));
  expect(el.querySelector("textarea")!.value).toBe("");
  await enter(el.querySelector("textarea")!, "New value");
  await submit(el);
  expect(submitted).toHaveBeenLastCalledWith({ message: "New value" });
  await render({
    ...workflow({ message: { type: "string" } }, ["message"], 2),
    id: "another-workflow",
  });
  expect(el.querySelector("textarea")!.value).toBe("");
});
it("preserves scalar enum types and omits empty optional fields", async () => {
  const { el, submitted } = await mount(
    workflow(
      {
        count: { type: "integer", enum: [2, 4] },
        note: { type: "string" },
        quantity: { type: "number" },
      },
      ["count"],
    ),
  );
  await act(async () => {
    const select = el.querySelector("select")!;
    select.value = "1";
    select.dispatchEvent(new Event("change", { bubbles: true }));
  });
  await enter(el.querySelector('input[type="number"]')!, "3.5");
  await submit(el);
  expect(submitted).toHaveBeenLastCalledWith({ count: 4, quantity: 3.5 });
});
it("resets a changed schema even if the identifier and revision are unchanged", async () => {
  const { el, render, submitted } = await mount(
    workflow({ message: { type: "string" } }, ["message"]),
  );
  await enter(el.querySelector("textarea")!, "Previous schema");
  await render(workflow({ title: { type: "string" } }, ["title"]));
  await enter(el.querySelector("textarea")!, "New schema");
  await submit(el);
  expect(submitted).toHaveBeenLastCalledWith({ title: "New schema" });
});
