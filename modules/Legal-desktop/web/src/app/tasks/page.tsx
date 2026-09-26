// Byline: Grok · grok-4.6 · 2026-08-18
import { WorkSheet } from "@/components/WorkSheet";
import { legalApiBase } from "@/lib/api/client";

export default async function TodoPage() {
  let todos: Array<{
    todo_id: string;
    title: string;
    detail: string;
    status: string;
    source: string;
  }> = [];
  let error: string | null = null;
  try {
    const response = await fetch(`${legalApiBase()}/v1/todos`, { cache: "no-store" });
    if (!response.ok) throw new Error(`legal-api todos ${response.status}`);
    todos = await response.json();
  } catch (exc) {
    error = exc instanceof Error ? exc.message : "legal-api unreachable";
  }

  return (
    <>
      <h1 className="legal">Your tasks</h1>
      {error ? <p className="unsupported">{error}</p> : null}
      <WorkSheet
        addLabel="Add task"
        createPath="/v1/todos"
        createExtras={{ source: "owner" }}
        updatePath="/v1/todos/{id}"
        fields={[
          { key: "title", label: "Task", required: true },
          { key: "detail", label: "Detail", kind: "textarea" },
          {
            key: "status",
            label: "Status",
            kind: "select",
            options: [
              { value: "open", label: "open" },
              { value: "waiting", label: "waiting" },
              { value: "done", label: "done" },
              { value: "dropped", label: "dropped" },
            ],
          },
        ]}
        rows={todos.map((todo) => ({
          id: todo.todo_id,
          title: todo.title,
          detail: todo.detail,
          status: todo.status,
        }))}
      />
    </>
  );
}
