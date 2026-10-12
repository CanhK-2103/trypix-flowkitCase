import { useState, type ReactNode } from "react";
import type { Workflow } from "./types";

type Field = {
  type: string;
  minLength?: number;
  maxLength?: number;
  minimum?: number;
  maximum?: number;
  enum?: (string | number | boolean)[];
};
type Schema = { properties?: Record<string, Field>; required?: string[] };
type Values = Record<string, unknown>;

function initialValues(schema: Schema): Values {
  const values: Values = {};
  for (const [name, field] of Object.entries(schema.properties ?? {})) {
    values[name] = undefined;
    if (field.enum?.length) {
      if (schema.required?.includes(name)) values[name] = field.enum[0];
    } else if (field.type === "boolean") {
      values[name] = false;
    } else if (field.type === "string") {
      values[name] = "";
    }
  }
  return values;
}

export function WorkflowInput({
  workflow,
  onSubmit,
  children,
}: {
  workflow: Workflow;
  onSubmit: (values: Values) => void;
  children: (values: Values) => ReactNode;
}) {
  const schema = (workflow.definition.nodes.find((n) => n.kind === "input")
    ?.params?.inputSchema ?? {}) as Schema;
  const key = JSON.stringify([workflow.id, workflow.revision, schema]);
  const [draft, setDraft] = useState<{ key: string; values: Values }>({
    key,
    values: initialValues(schema),
  });
  // A changed workflow/revision/schema resets synchronously, before any submission.
  const values = draft.key === key ? draft.values : initialValues(schema);
  const update = (name: string, value: unknown) =>
    setDraft({ key, values: { ...values, [name]: value } });
  const input = Object.fromEntries(
    Object.keys(schema.properties ?? {})
      .filter(
        (name) =>
          values[name] !== undefined &&
          (values[name] !== "" || schema.required?.includes(name)),
      )
      .map((name) => [name, values[name]]),
  );

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        onSubmit(input);
      }}
    >
      {Object.entries(schema.properties ?? {}).map(([name, field]) => (
        <label className="field" key={name}>
          <span>
            {name}
            {schema.required?.includes(name) && <b> *</b>}
          </span>
          {field.enum?.length ? (
            <select
              required={schema.required?.includes(name)}
              value={
                values[name] === undefined
                  ? ""
                  : String(
                      field.enum.indexOf(
                        values[name] as string | number | boolean,
                      ),
                    )
              }
              onChange={(event) =>
                update(
                  name,
                  event.target.value === ""
                    ? undefined
                    : field.enum![Number(event.target.value)],
                )
              }
            >
              <option value="">Seçiniz</option>
              {field.enum.map((value, i) => (
                <option key={i} value={i}>
                  {String(value)}
                </option>
              ))}
            </select>
          ) : field.type === "boolean" ? (
            <input
              type="checkbox"
              checked={!!values[name]}
              onChange={(event) => update(name, event.target.checked)}
            />
          ) : field.type === "string" ? (
            <textarea
              value={String(values[name] ?? "")}
              required={schema.required?.includes(name)}
              minLength={field.minLength}
              maxLength={field.maxLength}
              onInput={(event) => update(name, event.currentTarget.value)}
            />
          ) : (
            <input
              type="number"
              step={field.type === "integer" ? 1 : "any"}
              min={field.minimum}
              max={field.maximum}
              required={schema.required?.includes(name)}
              value={String(values[name] ?? "")}
              onInput={(event) =>
                update(
                  name,
                  event.currentTarget.value === ""
                    ? undefined
                    : Number(event.currentTarget.value),
                )
              }
            />
          )}
        </label>
      ))}
      {children(input)}
    </form>
  );
}
