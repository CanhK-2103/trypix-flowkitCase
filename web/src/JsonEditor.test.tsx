// @vitest-environment jsdom
import { act } from "react";
import { createRoot } from "react-dom/client";
import { EditorView } from "@codemirror/view";
import { afterEach, expect, it, vi } from "vitest";
import { JsonEditor } from "./JsonEditor";
Object.assign(globalThis, {
  IS_REACT_ACT_ENVIRONMENT: true,
  requestAnimationFrame: (callback: FrameRequestCallback) =>
    setTimeout(() => callback(0), 0),
  cancelAnimationFrame: clearTimeout,
});
const roots: ReturnType<typeof createRoot>[] = [];
afterEach(() => {
  for (const root of roots) {
    act(() => root.unmount());
  }
  roots.length = 0;
  document.body.innerHTML = "";
});
it("server save response updates the editor without marking the draft edited; user edits still emit", async () => {
  const el = document.createElement("div");
  document.body.append(el);
  const root = createRoot(el);
  roots.push(root);
  const change = vi.fn();
  await act(async () => {
    root.render(<JsonEditor value={'{"revision":1}'} onChange={change} />);
  });
  await act(async () => {
    root.render(<JsonEditor value={'{"revision":2}'} onChange={change} />);
  });
  expect(change).not.toHaveBeenCalled();
  const editorElement = el.querySelector<HTMLElement>(".cm-editor")!;
  const editor = EditorView.findFromDOM(editorElement)!;
  await act(async () => {
    editor.dispatch({
      changes: {
        from: 0,
        to: editor.state.doc.length,
        insert: '{"revision":2,"name":"unsaved"}',
      },
    });
  });
  expect(change).toHaveBeenLastCalledWith('{"revision":2,"name":"unsaved"}');
  expect(editor.state.doc.toString()).toContain("unsaved");
});
