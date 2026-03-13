"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { BookOpen } from "lucide-react";

import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

interface DocEntry {
  slug: string;
  title: string;
  order: number;
}

export default function DocsIndexPage() {
  const [entries, setEntries] = useState<DocEntry[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchEntries = useCallback(async () => {
    const res = await fetch("/api/docs");
    const data = (await res.json()) as DocEntry[];
    setEntries(data);
    setLoading(false);
  }, []);

  useEffect(() => {
    void fetchEntries();
  }, [fetchEntries]);

  return (
    <div className="mx-auto max-w-3xl p-8">
      <div className="mb-8">
        <Link
          href="/"
          className="text-muted-foreground hover:text-foreground text-sm"
        >
          &larr; Home
        </Link>
        <h1 className="mt-2 text-3xl font-bold">bob ORM Usage Docs</h1>
        <p className="text-muted-foreground mt-1">
          How bob is used in this project
        </p>
      </div>

      {loading ? (
        <div className="flex flex-col gap-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-16 rounded-xl" />
          ))}
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          {entries.map((entry) => (
            <Link key={entry.slug} href={`/docs/${entry.slug}`}>
              <Card className="hover:bg-accent/50 transition-colors">
                <CardHeader className="flex flex-row items-center gap-3 py-4">
                  <BookOpen className="text-muted-foreground size-5 shrink-0" />
                  <CardTitle className="text-base font-medium">
                    {entry.title}
                  </CardTitle>
                </CardHeader>
              </Card>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
