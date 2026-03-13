"use client";

import { use, useCallback, useEffect, useState } from "react";
import Link from "next/link";
import {
  ArrowRight,
  CheckCircle2,
  Clock,
  FolderKanban,
  ListTodo,
  TriangleAlert,
  Users,
} from "lucide-react";

import { api } from "@/lib/api/client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";

interface DashboardData {
  project_count: number;
  member_count: number;
  total_tasks: number;
  done_tasks: number;
  overdue_tasks: number;
}

interface ProjectStat {
  id: string;
  name: string;
  total_tasks: number;
  done_tasks: number;
  active_tasks: number;
}

interface TaskItem {
  id: string;
  title: string;
  status: string;
  priority: string;
  project_name?: string;
  assignee_name?: string;
  due_date?: string;
}

const statusColor: Record<string, string> = {
  todo: "bg-neutral-200 text-neutral-700",
  in_progress: "bg-blue-100 text-blue-700",
  done: "bg-emerald-100 text-emerald-700",
};

const priorityColor: Record<string, string> = {
  low: "bg-neutral-100 text-neutral-600",
  medium: "bg-yellow-100 text-yellow-700",
  high: "bg-orange-100 text-orange-700",
  urgent: "bg-red-100 text-red-700",
};

export default function DashboardPage({
  params,
}: {
  params: Promise<{ wsId: string }>;
}) {
  const { wsId } = use(params);
  const [dashboard, setDashboard] = useState<DashboardData | null>(null);
  const [projects, setProjects] = useState<ProjectStat[]>([]);
  const [recentTasks, setRecentTasks] = useState<TaskItem[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchAll = useCallback(async () => {
    const [dashRes, projRes, taskRes] = await Promise.all([
      api.GET("/workspaces/{wsId}/reports/dashboard", {
        params: { path: { wsId } },
      }),
      api.GET("/workspaces/{wsId}/reports/project-stats", {
        params: { path: { wsId } },
      }),
      api.GET("/workspaces/{wsId}/tasks", {
        params: { path: { wsId }, query: { limit: 5 } },
      }),
    ]);
    if (dashRes.data) setDashboard(dashRes.data);
    if (projRes.data) setProjects(projRes.data);
    if (taskRes.data) setRecentTasks(taskRes.data as unknown as TaskItem[]);
    setLoading(false);
  }, [wsId]);

  useEffect(() => {
    void fetchAll();
  }, [fetchAll]);

  if (loading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-48" />
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-24 rounded-xl" />
          ))}
        </div>
        <div className="grid gap-6 lg:grid-cols-2">
          <Skeleton className="h-64 rounded-xl" />
          <Skeleton className="h-64 rounded-xl" />
        </div>
      </div>
    );
  }

  if (!dashboard) {
    return <p className="text-muted-foreground">Failed to load dashboard.</p>;
  }

  const completionRate =
    dashboard.total_tasks > 0
      ? Math.round((dashboard.done_tasks / dashboard.total_tasks) * 100)
      : 0;

  const stats = [
    {
      label: "プロジェクト",
      value: dashboard.project_count,
      icon: FolderKanban,
      color: "text-blue-600",
      href: `/workspaces/${wsId}/projects`,
    },
    {
      label: "メンバー",
      value: dashboard.member_count,
      icon: Users,
      color: "text-green-600",
      href: `/workspaces/${wsId}/members`,
    },
    {
      label: "タスク合計",
      value: dashboard.total_tasks,
      icon: ListTodo,
      color: "text-purple-600",
      href: `/workspaces/${wsId}/tasks`,
    },
    {
      label: "完了済み",
      value: dashboard.done_tasks,
      icon: CheckCircle2,
      color: "text-emerald-600",
      sub: `${completionRate}%`,
    },
    {
      label: "期限超過",
      value: dashboard.overdue_tasks,
      icon: dashboard.overdue_tasks > 0 ? TriangleAlert : Clock,
      color:
        dashboard.overdue_tasks > 0
          ? "text-red-600"
          : "text-muted-foreground",
    },
  ];

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">ダッシュボード</h1>

      {/* KPI カード */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
        {stats.map((stat) => {
          const inner = (
            <Card
              className={
                stat.href
                  ? "hover:bg-accent/50 cursor-pointer transition-colors"
                  : ""
              }
            >
              <CardHeader className="flex flex-row items-center justify-between pb-1">
                <CardTitle className="text-muted-foreground text-xs font-medium">
                  {stat.label}
                </CardTitle>
                <stat.icon className={`size-4 ${stat.color}`} />
              </CardHeader>
              <CardContent className="pb-3">
                <div className="flex items-baseline gap-2">
                  <span className="text-2xl font-bold">{stat.value}</span>
                  {stat.sub && (
                    <span className="text-muted-foreground text-sm">
                      {stat.sub}
                    </span>
                  )}
                </div>
              </CardContent>
            </Card>
          );
          return stat.href ? (
            <Link key={stat.label} href={stat.href}>
              {inner}
            </Link>
          ) : (
            <div key={stat.label}>{inner}</div>
          );
        })}
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* プロジェクト進捗 */}
        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <div>
                <CardTitle className="text-base">プロジェクト進捗</CardTitle>
                <CardDescription>各プロジェクトのタスク消化率</CardDescription>
              </div>
              <Link href={`/workspaces/${wsId}/reports`}>
                <Button variant="ghost" size="sm">
                  レポート
                  <ArrowRight className="ml-1 size-3" />
                </Button>
              </Link>
            </div>
          </CardHeader>
          <CardContent>
            {projects.length === 0 ? (
              <p className="text-muted-foreground py-8 text-center text-sm">
                プロジェクトがありません
              </p>
            ) : (
              <div className="space-y-4">
                {projects.map((proj) => {
                  const pct =
                    proj.total_tasks > 0
                      ? Math.round(
                          (proj.done_tasks / proj.total_tasks) * 100,
                        )
                      : 0;
                  return (
                    <div key={proj.id} className="space-y-1.5">
                      <div className="flex items-center justify-between text-sm">
                        <span className="font-medium">{proj.name}</span>
                        <span className="text-muted-foreground">
                          {proj.done_tasks}/{proj.total_tasks} ({pct}%)
                        </span>
                      </div>
                      <div className="bg-muted h-2 overflow-hidden rounded-full">
                        <div
                          className="h-full rounded-full bg-emerald-500 transition-all"
                          style={{ width: `${pct}%` }}
                        />
                      </div>
                      <div className="text-muted-foreground flex gap-3 text-xs">
                        <span>進行中: {proj.active_tasks}</span>
                        <span>完了: {proj.done_tasks}</span>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </CardContent>
        </Card>

        {/* 最近のタスク */}
        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <div>
                <CardTitle className="text-base">最近のタスク</CardTitle>
                <CardDescription>直近追加されたタスク</CardDescription>
              </div>
              <Link href={`/workspaces/${wsId}/tasks`}>
                <Button variant="ghost" size="sm">
                  すべて表示
                  <ArrowRight className="ml-1 size-3" />
                </Button>
              </Link>
            </div>
          </CardHeader>
          <CardContent>
            {recentTasks.length === 0 ? (
              <p className="text-muted-foreground py-8 text-center text-sm">
                タスクがありません
              </p>
            ) : (
              <div className="space-y-1">
                {recentTasks.map((task, i) => (
                  <div key={task.id}>
                    {i > 0 && <Separator className="my-2" />}
                    <div className="flex items-start justify-between gap-2 py-1">
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium">
                          {task.title}
                        </p>
                        <div className="text-muted-foreground mt-0.5 flex items-center gap-2 text-xs">
                          {task.project_name && (
                            <span>{task.project_name}</span>
                          )}
                          {task.assignee_name && (
                            <>
                              <span>·</span>
                              <span>{task.assignee_name}</span>
                            </>
                          )}
                        </div>
                      </div>
                      <div className="flex shrink-0 items-center gap-1.5">
                        <Badge
                          variant="secondary"
                          className={`text-[10px] ${priorityColor[task.priority] ?? ""}`}
                        >
                          {task.priority}
                        </Badge>
                        <Badge
                          variant="secondary"
                          className={`text-[10px] ${statusColor[task.status] ?? ""}`}
                        >
                          {task.status}
                        </Badge>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
