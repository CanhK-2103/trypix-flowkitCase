import { useEffect, useRef } from "react";
import { EditorView, basicSetup } from "codemirror";
import { json, jsonParseLinter } from "@codemirror/lang-json";
import { linter } from "@codemirror/lint";
export function JsonEditor({
  value,
  onChange,
}: {
  value: string;
  onChange: (value: string) => void;
}) {
  const container = useRef<HTMLDivElement>(null);
  const editor = useRef<EditorView | null>(null);
  const callback = useRef(onChange);
  const applying = useRef(false);
  callback.current = onChange;
  useEffect(() => {
    const view = new EditorView({
      doc: value,
      extensions: [
        basicSetup,
        json(),
        linter(jsonParseLinter()),
        EditorView.lineWrapping,
        EditorView.updateListener.of((update) => {
          if (update.docChanged && !applying.current)
            callback.current(update.state.doc.toString());
        }),
        EditorView.theme({
          "&": { fontSize: "12px", height: "460px" },
          ".cm-scroller": {
            overflow: "auto",
            fontFamily: "ui-monospace, SFMono-Regular, Consolas, monospace",
          },
          ".cm-content": { padding: "14px 0" },
          ".cm-gutters": {
            backgroundColor: "#f8f9fb",
            borderRight: "1px solid #e7eaf0",
          },
        }),
      ],
      parent: container.current!,
    });
    editor.current = view;
    return () => {
      view.destroy();
      editor.current = null;
    };
  }, []);
  useEffect(() => {
    const view = editor.current;
    if (view && view.state.doc.toString() !== value) {
      applying.current = true;
      try {
        view.dispatch({
          changes: { from: 0, to: view.state.doc.length, insert: value },
        });
      } finally {
        applying.current = false;
      }
    }
  }, [value]);
  return <div className="json-editor" ref={container} />;
}
