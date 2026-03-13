"use client";

import { use } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  BarChart3,
  FolderKanban,
  LayoutDashboard,
  ListTodo,
  Users,
} from "lucide-react";

import { cn } from "@/lib/utils";
import { Separator } from "@/components/ui/separator";

const navItems = [
  { label: "Dashboard", href: "", icon: LayoutDashboard },
  { label: "Projects", href: "/projects", icon: FolderKanban },
  { label: "Tasks", href: "/tasks", icon: ListTodo },
  { label: "Members", href: "/members", icon: Users },
  { label: "Reports", href: "/reports", icon: BarChart3 },
];

export default function WorkspaceLayout({
  children,
  params,
}: {
  children: React.ReactNode;
  params: Promise<{ wsId: string }>;
}) {
  const { wsId } = use(params);
  const pathname = usePathname();
  const basePath = `/workspaces/${wsId}`;

  return (
    <div className="flex min-h-screen">
      <aside className="border-r bg-muted/30 w-56 shrink-0">
        <div className="p-4">
          <Link
            href="/"
            className="text-muted-foreground hover:text-foreground text-sm"
          >
            &larr; Workspaces
          </Link>
          <h2 className="mt-2 truncate text-lg font-semibold">Workspace</h2>
          <p className="text-muted-foreground truncate font-mono text-xs">
            {wsId.slice(0, 8)}...
          </p>
        </div>
        <Separator />
        <nav className="flex flex-col gap-1 p-2">
          {navItems.map((item) => {
            const href = `${basePath}${item.href}`;
            const isActive =
              item.href === ""
                ? pathname === basePath
                : pathname.startsWith(href);
            return (
              <Link
                key={item.href}
                href={href}
                className={cn(
                  "flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                  isActive
                    ? "bg-accent text-accent-foreground"
                    : "text-muted-foreground hover:bg-accent/50 hover:text-foreground",
                )}
              >
                <item.icon className="size-4" />
                {item.label}
              </Link>
            );
          })}
        </nav>
      </aside>
      <main className="flex-1 overflow-auto p-6">{children}</main>
    </div>
  );
}
