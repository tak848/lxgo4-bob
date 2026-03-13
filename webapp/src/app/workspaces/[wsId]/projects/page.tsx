"use client";

import { use, useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { ExternalLink, Plus } from "lucide-react";

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

interface Project {
  id: string;
  workspace_id: string;
  name: string;
  description: string;
  status: "active" | "archived";
  total_tasks?: number;
  done_tasks?: number;
  active_tasks?: number;
}

export default function ProjectsPage({
  params,
}: {
  params: Promise<{ wsId: string }>;
}) {
  const { wsId } = use(params);
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [open, setOpen] = useState(false);
  const [editProject, setEditProject] = useState<Project | null>(null);
  const [form, setForm] = useState({
    name: "",
    description: "",
    status: "active" as "active" | "archived",
  });

  const fetchProjects = useCallback(async () => {
    const { data } = await api.GET("/workspaces/{wsId}/projects", {
      params: { path: { wsId } },
    });
    if (data) {
      setProjects(data as unknown as Project[]);
    }
    setLoading(false);
  }, [wsId]);

  useEffect(() => {
    void fetchProjects();
  }, [fetchProjects]);

  const resetForm = () => {
    setForm({ name: "", description: "", status: "active" });
    setEditProject(null);
  };

  const handleOpenCreate = () => {
    resetForm();
    setOpen(true);
  };

  const handleOpenEdit = (project: Project) => {
    setEditProject(project);
    setForm({
      name: project.name,
      description: project.description,
      status: project.status,
    });
    setOpen(true);
  };

  const handleSave = async () => {
    if (editProject) {
      await api.PUT("/workspaces/{wsId}/projects/{id}", {
        params: { path: { wsId, id: editProject.id } },
        body: form,
      });
    } else {
      await api.POST("/workspaces/{wsId}/projects", {
        params: { path: { wsId } },
        body: form,
      });
    }
    setOpen(false);
    resetForm();
    void fetchProjects();
  };

  const handleDelete = async (id: string) => {
    await api.DELETE("/workspaces/{wsId}/projects/{id}", {
      params: { path: { wsId, id } },
    });
    setProjects((prev) => prev.filter((p) => p.id !== id));
  };

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">プロジェクト</h1>
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
              新規作成
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>
                {editProject ? "プロジェクト編集" : "プロジェクト作成"}
              </DialogTitle>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label>名前</Label>
                <Input
                  value={form.name}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, name: e.target.value }))
                  }
                />
              </div>
              <div className="grid gap-2">
                <Label>説明</Label>
                <Textarea
                  value={form.description}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, description: e.target.value }))
                  }
                />
              </div>
              <div className="grid gap-2">
                <Label>ステータス</Label>
                <Select
                  value={form.status}
                  onValueChange={(v: "active" | "archived") =>
                    setForm((f) => ({ ...f, status: v }))
                  }
                >
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="active">Active</SelectItem>
                    <SelectItem value="archived">Archived</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <DialogFooter>
              <Button onClick={() => void handleSave()} disabled={!form.name}>
                {editProject ? "保存" : "作成"}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      {loading ? (
        <p className="text-muted-foreground">Loading...</p>
      ) : projects.length === 0 ? (
        <p className="text-muted-foreground">
          プロジェクトがありません。作成してください。
        </p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>プロジェクト名</TableHead>
              <TableHead>説明</TableHead>
              <TableHead>ステータス</TableHead>
              <TableHead className="text-center">タスク数</TableHead>
              <TableHead className="text-center">進捗</TableHead>
              <TableHead className="w-40">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {projects.map((project) => {
              const total = project.total_tasks ?? 0;
              const done = project.done_tasks ?? 0;
              const active = project.active_tasks ?? 0;
              const pct = total > 0 ? Math.round((done / total) * 100) : 0;
              return (
                <TableRow key={project.id}>
                  <TableCell className="font-medium">
                    {project.name}
                  </TableCell>
                  <TableCell className="max-w-xs truncate">
                    {project.description}
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={
                        project.status === "active" ? "default" : "secondary"
                      }
                    >
                      {project.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-center">
                    <div className="flex flex-col items-center gap-0.5">
                      <span className="text-lg font-semibold">{total}</span>
                      <span className="text-muted-foreground text-xs">
                        完了 {done} / 進行中 {active}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell className="text-center">
                    <div className="mx-auto w-20">
                      <div className="bg-muted h-2 overflow-hidden rounded-full">
                        <div
                          className="h-full rounded-full bg-emerald-500 transition-all"
                          style={{ width: `${pct}%` }}
                        />
                      </div>
                      <span className="text-muted-foreground text-xs">
                        {pct}%
                      </span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex gap-2">
                      <Link
                        href={`/workspaces/${wsId}/tasks?project_id=${project.id}`}
                      >
                        <Button variant="outline" size="sm">
                          <ExternalLink className="mr-1 size-3" />
                          タスク
                        </Button>
                      </Link>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleOpenEdit(project)}
                      >
                        編集
                      </Button>
                      <Button
                        variant="destructive"
                        size="sm"
                        onClick={() => void handleDelete(project.id)}
                      >
                        削除
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      )}
    </div>
  );
}
