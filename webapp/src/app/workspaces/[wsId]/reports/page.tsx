"use client";

import { use, useCallback, useEffect, useState } from "react";
import Link from "next/link";
import {
  CheckCircle2,
  Clock,
  ExternalLink,
  FolderKanban,
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

interface ProjectStats {
  id: string;
  name: string;
  total_tasks: number;
  done_tasks: number;
  active_tasks: number;
}

interface MemberSummary {
  id: string;
  name: string;
  assigned_tasks: number;
  completed_tasks: number;
  overdue_tasks: number;
}

interface DashboardData {
  project_count: number;
  member_count: number;
  total_tasks: number;
  done_tasks: number;
  overdue_tasks: number;
}

export default function ReportsPage({
  params,
}: {
  params: Promise<{ wsId: string }>;
}) {
  const { wsId } = use(params);
  const [dashboard, setDashboard] = useState<DashboardData | null>(null);
  const [projectStats, setProjectStats] = useState<ProjectStats[]>([]);
  const [memberSummary, setMemberSummary] = useState<MemberSummary[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchReports = useCallback(async () => {
    const [dashRes, psRes, msRes] = await Promise.all([
      api.GET("/workspaces/{wsId}/reports/dashboard", {
        params: { path: { wsId } },
      }),
      api.GET("/workspaces/{wsId}/reports/project-stats", {
        params: { path: { wsId } },
      }),
      api.GET("/workspaces/{wsId}/reports/member-summary", {
        params: { path: { wsId } },
      }),
    ]);
    if (dashRes.data) setDashboard(dashRes.data);
    if (psRes.data) setProjectStats(psRes.data);
    if (msRes.data) setMemberSummary(msRes.data);
    setLoading(false);
  }, [wsId]);

  useEffect(() => {
    void fetchReports();
  }, [fetchReports]);

  if (loading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-48" />
        <div className="grid gap-4 sm:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-20 rounded-xl" />
          ))}
        </div>
        <Skeleton className="h-96 rounded-xl" />
      </div>
    );
  }

  const totalTasks = dashboard?.total_tasks ?? 0;
  const doneTasks = dashboard?.done_tasks ?? 0;
  const overdueTasks = dashboard?.overdue_tasks ?? 0;
  const inProgress = totalTasks - doneTasks;
  const completionRate =
    totalTasks > 0 ? Math.round((doneTasks / totalTasks) * 100) : 0;

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">レポート</h1>

      {/* サマリーカード */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardContent className="flex items-center gap-3 pt-5">
            <div className="rounded-lg bg-purple-100 p-2">
              <FolderKanban className="size-5 text-purple-600" />
            </div>
            <div>
              <p className="text-muted-foreground text-xs">タスク合計</p>
              <p className="text-2xl font-bold">{totalTasks}</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-3 pt-5">
            <div className="rounded-lg bg-emerald-100 p-2">
              <CheckCircle2 className="size-5 text-emerald-600" />
            </div>
            <div>
              <p className="text-muted-foreground text-xs">完了率</p>
              <p className="text-2xl font-bold">{completionRate}%</p>
              <p className="text-muted-foreground text-xs">
                {doneTasks} / {totalTasks} 完了
              </p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-3 pt-5">
            <div className="rounded-lg bg-blue-100 p-2">
              <Clock className="size-5 text-blue-600" />
            </div>
            <div>
              <p className="text-muted-foreground text-xs">進行中</p>
              <p className="text-2xl font-bold">{inProgress}</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-3 pt-5">
            <div
              className={`rounded-lg p-2 ${overdueTasks > 0 ? "bg-red-100" : "bg-neutral-100"}`}
            >
              <TriangleAlert
                className={`size-5 ${overdueTasks > 0 ? "text-red-600" : "text-muted-foreground"}`}
              />
            </div>
            <div>
              <p className="text-muted-foreground text-xs">期限超過</p>
              <p
                className={`text-2xl font-bold ${overdueTasks > 0 ? "text-red-600" : ""}`}
              >
                {overdueTasks}
              </p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 全体進捗バー */}
      <Card>
        <CardContent className="pt-5">
          <div className="mb-2 flex items-center justify-between text-sm">
            <span className="font-medium">全体進捗</span>
            <span className="text-muted-foreground">{completionRate}%</span>
          </div>
          <div className="bg-muted h-3 overflow-hidden rounded-full">
            <div
              className="h-full rounded-full bg-emerald-500 transition-all"
              style={{ width: `${completionRate}%` }}
            />
          </div>
        </CardContent>
      </Card>

      {/* タブ: プロジェクト統計 / メンバーサマリ */}
      <Tabs defaultValue="projects">
        <TabsList>
          <TabsTrigger value="projects">
            <FolderKanban className="mr-1.5 size-4" />
            プロジェクト別
          </TabsTrigger>
          <TabsTrigger value="members">
            <Users className="mr-1.5 size-4" />
            メンバー別
          </TabsTrigger>
        </TabsList>

        <TabsContent value="projects" className="mt-4">
          {projectStats.length === 0 ? (
            <Card>
              <CardContent className="py-12 text-center">
                <p className="text-muted-foreground">データがありません</p>
              </CardContent>
            </Card>
          ) : (
            <div className="grid gap-4 sm:grid-cols-2">
              {projectStats.map((ps) => {
                const pct =
                  ps.total_tasks > 0
                    ? Math.round((ps.done_tasks / ps.total_tasks) * 100)
                    : 0;
                const todo =
                  ps.total_tasks - ps.done_tasks - ps.active_tasks;
                return (
                  <Card key={ps.id}>
                    <CardHeader className="pb-3">
                      <div className="flex items-start justify-between">
                        <div>
                          <CardTitle className="text-base">
                            {ps.name}
                          </CardTitle>
                          <CardDescription>
                            {ps.total_tasks} タスク
                          </CardDescription>
                        </div>
                        <Link
                          href={`/workspaces/${wsId}/tasks?project_id=${ps.id}`}
                        >
                          <Button variant="ghost" size="sm">
                            <ExternalLink className="size-3" />
                          </Button>
                        </Link>
                      </div>
                    </CardHeader>
                    <CardContent className="space-y-3">
                      {/* 進捗バー */}
                      <div>
                        <div className="mb-1 flex justify-between text-xs">
                          <span className="text-muted-foreground">進捗</span>
                          <span className="font-medium">{pct}%</span>
                        </div>
                        <div className="bg-muted h-2 overflow-hidden rounded-full">
                          <div
                            className="h-full rounded-full bg-emerald-500 transition-all"
                            style={{ width: `${pct}%` }}
                          />
                        </div>
                      </div>
                      <Separator />
                      {/* 内訳 */}
                      <div className="flex justify-between text-sm">
                        <div className="flex items-center gap-1.5">
                          <div className="size-2.5 rounded-full bg-neutral-300" />
                          <span className="text-muted-foreground">未着手</span>
                          <span className="font-medium">{todo > 0 ? todo : 0}</span>
                        </div>
                        <div className="flex items-center gap-1.5">
                          <div className="size-2.5 rounded-full bg-blue-500" />
                          <span className="text-muted-foreground">進行中</span>
                          <span className="font-medium">
                            {ps.active_tasks}
                          </span>
                        </div>
                        <div className="flex items-center gap-1.5">
                          <div className="size-2.5 rounded-full bg-emerald-500" />
                          <span className="text-muted-foreground">完了</span>
                          <span className="font-medium">{ps.done_tasks}</span>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                );
              })}
            </div>
          )}
        </TabsContent>

        <TabsContent value="members" className="mt-4">
          {memberSummary.length === 0 ? (
            <Card>
              <CardContent className="py-12 text-center">
                <p className="text-muted-foreground">データがありません</p>
              </CardContent>
            </Card>
          ) : (
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {memberSummary.map((ms) => {
                const completionPct =
                  ms.assigned_tasks > 0
                    ? Math.round(
                        (ms.completed_tasks / ms.assigned_tasks) * 100,
                      )
                    : 0;
                return (
                  <Card key={ms.id}>
                    <CardHeader className="pb-3">
                      <CardTitle className="text-base">{ms.name}</CardTitle>
                      <CardDescription>
                        担当 {ms.assigned_tasks} タスク
                      </CardDescription>
                    </CardHeader>
                    <CardContent className="space-y-3">
                      {/* 進捗バー */}
                      <div>
                        <div className="mb-1 flex justify-between text-xs">
                          <span className="text-muted-foreground">完了率</span>
                          <span className="font-medium">
                            {completionPct}%
                          </span>
                        </div>
                        <div className="bg-muted h-2 overflow-hidden rounded-full">
                          <div
                            className="h-full rounded-full bg-emerald-500 transition-all"
                            style={{ width: `${completionPct}%` }}
                          />
                        </div>
                      </div>
                      <Separator />
                      {/* バッジ */}
                      <div className="flex flex-wrap gap-2">
                        <Badge
                          variant="secondary"
                          className="bg-emerald-100 text-emerald-700"
                        >
                          <CheckCircle2 className="mr-1 size-3" />
                          完了 {ms.completed_tasks}
                        </Badge>
                        <Badge
                          variant="secondary"
                          className="bg-blue-100 text-blue-700"
                        >
                          <Clock className="mr-1 size-3" />
                          進行中{" "}
                          {ms.assigned_tasks - ms.completed_tasks}
                        </Badge>
                        {ms.overdue_tasks > 0 && (
                          <Badge
                            variant="secondary"
                            className="bg-red-100 text-red-700"
                          >
                            <TriangleAlert className="mr-1 size-3" />
                            期限超過 {ms.overdue_tasks}
                          </Badge>
                        )}
                      </div>
                    </CardContent>
                  </Card>
                );
              })}
            </div>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}
