import fs from "node:fs/promises";
import path from "node:path";
import { NextResponse } from "next/server";

const DOCS_DIR = path.join(process.cwd(), "..", "docs");

export async function GET() {
  const files = await fs.readdir(DOCS_DIR);
  const entries = [];

  for (const file of files) {
    if (!file.includes("bob-usage") || !file.endsWith(".md")) continue;
    if (file === "bob-usage.md") continue;

    const content = await fs.readFile(path.join(DOCS_DIR, file), "utf-8");
    const firstLine = content.split("\n").find((l) => l.startsWith("# "));
    const title = firstLine?.replace(/^#\s+/, "") ?? file;
    const orderMatch = title.match(/^(\d+)\./);
    const order = orderMatch ? parseInt(orderMatch[1], 10) : 99;
    const slug = file.replace(/\.md$/, "");

    entries.push({ slug, title, order });
  }

  entries.sort((a, b) => a.order - b.order);
  return NextResponse.json(entries);
}
