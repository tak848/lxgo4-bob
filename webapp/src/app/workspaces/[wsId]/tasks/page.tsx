"use client";

import { use, useCallback, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { Plus } from "lucide-react";

import { api } from "@/lib/api/client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";

type TaskStatus = "todo" | "in_progress" | "done";
type TaskPriority = "low" | "medium" | "high" | "urgent";

interface Task {
  id: string;
  workspace_id: string;
  project_id: string;
  assignee_id?: string;
  title: string;
  description: string;
  status: TaskStatus;
  priority: TaskPriority;
  due_date?: string;
}

interface Project {
  id: string;
  name: string;
}

interface Member {
  id: string;
  name: string;
}

const statusColors: Record<TaskStatus, "default" | "secondary" | "outline"> = {
  todo: "outline",
  in_progress: "secondary",
  done: "default",
};

const priorityColors: Record<
  TaskPriority,
  "default" | "secondary" | "destructive" | "outline"
> = {
  low: "outline",
  medium: "secondary",
  high: "default",
  urgent: "destructive",
};

const FILTER_ALL = "__all__";

export default function TasksPage({
  params,
}: {
  params: Promise<{ wsId: string }>;
}) {
  const { wsId } = use(params);
  const searchParams = useSearchParams();
  const [tasks, setTasks] = useState<Task[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [members, setMembers] = useState<Member[]>([]);
  const [loading, setLoading] = useState(true);
  const [open, setOpen] = useState(false);
  const [editTask, setEditTask] = useState<Task | null>(null);

  const [filterStatus, setFilterStatus] = useState<string>(FILTER_ALL);
  const [filterPriority, setFilterPriority] = useState<string>(FILTER_ALL);
  const [filterProject, setFilterProject] = useState<string>(searchParams.get("project_id") ?? FILTER_ALL);
  const [filterAssignee, setFilterAssignee] = useState<string>(FILTER_ALL);

  const [form, setForm] = useState({
    project_id: "",
    assignee_id: "",
    title: "",
    description: "",
    status: "todo" as TaskStatus,
    priority: "medium" as TaskPriority,
    due_date: "",
  });

  const fetchData = useCallback(async () => {
    const [tasksRes, projectsRes, membersRes] = await Promise.all([
      api.GET("/workspaces/{wsId}/tasks", {
        params: {
          path: { wsId },
          query: {
            ...(filterStatus !== FILTER_ALL && {
              status: filterStatus as TaskStatus,
            }),
            ...(filterPriority !== FILTER_ALL && {
              priority: filterPriority as TaskPriority,
            }),
            ...(filterProject !== FILTER_ALL && {
              project_id: filterProject,
            }),
            ...(filterAssignee !== FILTER_ALL && {
              assignee_id: filterAssignee,
            }),
            limit: 100,
          },
        },
      }),
      api.GET("/workspaces/{wsId}/projects", {
        params: { path: { wsId } },
      }),
      api.GET("/workspaces/{wsId}/members", {
        params: { path: { wsId } },
      }),
    ]);
    if (tasksRes.data) setTasks(tasksRes.data);
    if (projectsRes.data) setProjects(projectsRes.data);
    if (membersRes.data) setMembers(membersRes.data);
    setLoading(false);
  }, [wsId, filterStatus, filterPriority, filterProject, filterAssignee]);

  useEffect(() => {
    void fetchData();
  }, [fetchData]);

  const resetForm = () => {
    setForm({
      project_id: "",
      assignee_id: "",
      title: "",
      description: "",
      status: "todo",
      priority: "medium",
      due_date: "",
    });
    setEditTask(null);
  };

  const handleOpenCreate = () => {
    resetForm();
    setOpen(true);
  };

  const handleOpenEdit = (task: Task) => {
    setEditTask(task);
    setForm({
      project_id: task.project_id,
      assignee_id: task.assignee_id ?? "",
      title: task.title,
      description: task.description,
      status: task.status,
      priority: task.priority,
      due_date: task.due_date ?? "",
    });
    setOpen(true);
  };

  const handleSave = async () => {
    const body = {
      project_id: form.project_id,
      title: form.title,
      description: form.description,
      status: form.status,
      priority: form.priority,
      ...(form.assignee_id && { assignee_id: form.assignee_id }),
      ...(form.due_date && { due_date: form.due_date }),
    };
    if (editTask) {
      const { data } = await api.PUT("/workspaces/{wsId}/tasks/{id}", {
        params: { path: { wsId, id: editTask.id } },
        body,
      });
      if (data) {
        setTasks((prev) => prev.map((t) => (t.id === data.id ? data : t)));
      }
    } else {
      const { data } = await api.POST("/workspaces/{wsId}/tasks", {
        params: { path: { wsId } },
        body,
      });
      if (data) {
        setTasks((prev) => [...prev, data]);
      }
    }
    setOpen(false);
    resetForm();
  };

  const handleDelete = async (id: string) => {
    await api.DELETE("/workspaces/{wsId}/tasks/{id}", {
      params: { path: { wsId, id } },
    });
    setTasks((prev) => prev.filter((t) => t.id !== id));
  };

  const projectMap = new Map(projects.map((p) => [p.id, p.name]));
  const memberMap = new Map(members.map((m) => [m.id, m.name]));

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">Tasks</h1>
        <Dialog
          open={open}
          onOpenChange={(v) => {
            setOpen(v);
            if (!v) resetForm();
          }}
        >
          <DialogTrigger asChild>
            <Button onClick={handleOpenCreate}>
              <Plus className="size-4" />
              New Task
            </Button>
          </DialogTrigger>
          <DialogContent className="max-w-lg">
            <DialogHeader>
              <DialogTitle>
                {editTask ? "Edit Task" : "Create Task"}
              </DialogTitle>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label>Title</Label>
                <Input
                  value={form.title}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, title: e.target.value }))
                  }
                />
              </div>
              <div className="grid gap-2">
                <Label>Description</Label>
                <Textarea
                  value={form.description}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, description: e.target.value }))
                  }
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="grid gap-2">
                  <Label>Project</Label>
                  <Select
                    value={form.project_id}
                    onValueChange={(v) =>
                      setForm((f) => ({ ...f, project_id: v }))
                    }
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue placeholder="Select project" />
                    </SelectTrigger>
                    <SelectContent>
                      {projects.map((p) => (
                        <SelectItem key={p.id} value={p.id}>
                          {p.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="grid gap-2">
                  <Label>Assignee</Label>
                  <Select
                    value={form.assignee_id || FILTER_ALL}
                    onValueChange={(v) =>
                      setForm((f) => ({
                        ...f,
                        assignee_id: v === FILTER_ALL ? "" : v,
                      }))
                    }
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue placeholder="Unassigned" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value={FILTER_ALL}>Unassigned</SelectItem>
                      {members.map((m) => (
                        <SelectItem key={m.id} value={m.id}>
                          {m.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className="grid grid-cols-3 gap-4">
                <div className="grid gap-2">
                  <Label>Status</Label>
                  <Select
                    value={form.status}
                    onValueChange={(v: TaskStatus) =>
                      setForm((f) => ({ ...f, status: v }))
                    }
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="todo">Todo</SelectItem>
                      <SelectItem value="in_progress">In Progress</SelectItem>
                      <SelectItem value="done">Done</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="grid gap-2">
                  <Label>Priority</Label>
                  <Select
                    value={form.priority}
                    onValueChange={(v: TaskPriority) =>
                      setForm((f) => ({ ...f, priority: v }))
                    }
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="low">Low</SelectItem>
                      <SelectItem value="medium">Medium</SelectItem>
                      <SelectItem value="high">High</SelectItem>
                      <SelectItem value="urgent">Urgent</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="grid gap-2">
                  <Label>Due Date</Label>
                  <Input
                    type="date"
                    value={form.due_date}
                    onChange={(e) =>
                      setForm((f) => ({ ...f, due_date: e.target.value }))
                    }
                  />
                </div>
              </div>
            </div>
            <DialogFooter>
              <Button
                onClick={() => void handleSave()}
                disabled={!form.title || !form.project_id}
              >
                {editTask ? "Save" : "Create"}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      <div className="mb-4 flex flex-wrap gap-3">
        <Select value={filterStatus} onValueChange={setFilterStatus}>
          <SelectTrigger className="w-36">
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={FILTER_ALL}>All Status</SelectItem>
            <SelectItem value="todo">Todo</SelectItem>
            <SelectItem value="in_progress">In Progress</SelectItem>
            <SelectItem value="done">Done</SelectItem>
          </SelectContent>
        </Select>
        <Select value={filterPriority} onValueChange={setFilterPriority}>
          <SelectTrigger className="w-36">
            <SelectValue placeholder="Priority" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={FILTER_ALL}>All Priority</SelectItem>
            <SelectItem value="low">Low</SelectItem>
            <SelectItem value="medium">Medium</SelectItem>
            <SelectItem value="high">High</SelectItem>
            <SelectItem value="urgent">Urgent</SelectItem>
          </SelectContent>
        </Select>
        <Select value={filterProject} onValueChange={setFilterProject}>
          <SelectTrigger className="w-40">
            <SelectValue placeholder="Project" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={FILTER_ALL}>All Projects</SelectItem>
            {projects.map((p) => (
              <SelectItem key={p.id} value={p.id}>
                {p.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={filterAssignee} onValueChange={setFilterAssignee}>
          <SelectTrigger className="w-40">
            <SelectValue placeholder="Assignee" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={FILTER_ALL}>All Members</SelectItem>
            {members.map((m) => (
              <SelectItem key={m.id} value={m.id}>
                {m.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {loading ? (
        <p className="text-muted-foreground">Loading...</p>
      ) : tasks.length === 0 ? (
        <p className="text-muted-foreground">No tasks found.</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Title</TableHead>
              <TableHead>Project</TableHead>
              <TableHead>Assignee</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Priority</TableHead>
              <TableHead>Due Date</TableHead>
              <TableHead className="w-32">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {tasks.map((task) => (
              <TableRow key={task.id}>
                <TableCell className="font-medium">{task.title}</TableCell>
                <TableCell>
                  {projectMap.get(task.project_id) ?? "-"}
                </TableCell>
                <TableCell>
                  {task.assignee_id
                    ? (memberMap.get(task.assignee_id) ?? "-")
                    : "-"}
                </TableCell>
                <TableCell>
                  <Badge variant={statusColors[task.status]}>
                    {task.status.replace("_", " ")}
                  </Badge>
                </TableCell>
                <TableCell>
                  <Badge variant={priorityColors[task.priority]}>
                    {task.priority}
                  </Badge>
                </TableCell>
                <TableCell>{task.due_date ?? "-"}</TableCell>
                <TableCell>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleOpenEdit(task)}
                    >
                      Edit
                    </Button>
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => void handleDelete(task.id)}
                    >
                      Delete
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  );
}
