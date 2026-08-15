// Tasks API: CRUD for Shelley work-item tasks plus the directory dropdown data.
const baseUrl = "/api";

export interface Task {
  task_id: string;
  title: string;
  cwd?: string | null;
  tags: string[];
  created_at: string;
  updated_at: string;
  handled: boolean;
  handled_at?: string | null;
  thread_slug?: string | null;
  missing?: boolean;
}

export interface TaskDirectories {
  git_roots: string[];
  cwds: string[];
}

async function parseResponse<T>(response: Response, prefix: string): Promise<T> {
  if (!response.ok) {
    let msg = response.statusText;
    try {
      const data = await response.json();
      if (data && data.message) msg = data.message;
    } catch {
      /* ignore */
    }
    throw new Error(`${prefix}: ${msg}`);
  }
  return response.json();
}

export const tasksApi = {
  async list(): Promise<Task[]> {
    const res = await fetch(`${baseUrl}/tasks`);
    return parseResponse<Task[]>(res, "Failed to list tasks");
  },

  async create(input: { title: string; cwd?: string | null; tags?: string[] }): Promise<Task> {
    const res = await fetch(`${baseUrl}/tasks`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(input),
    });
    return parseResponse<Task>(res, "Failed to create task");
  },

  async update(
    taskId: string,
    input: { title?: string; cwd?: string | null; tags?: string[] },
  ): Promise<Task> {
    const res = await fetch(`${baseUrl}/tasks/${encodeURIComponent(taskId)}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(input),
    });
    return parseResponse<Task>(res, "Failed to update task");
  },

  async delete(taskId: string): Promise<void> {
    const res = await fetch(`${baseUrl}/tasks/${encodeURIComponent(taskId)}`, {
      method: "DELETE",
    });
    if (!res.ok) {
      await parseResponse(res, "Failed to delete task");
    }
  },

  async directories(): Promise<TaskDirectories> {
    const res = await fetch(`${baseUrl}/tasks/directories`);
    return parseResponse<TaskDirectories>(res, "Failed to load directories");
  },
};
