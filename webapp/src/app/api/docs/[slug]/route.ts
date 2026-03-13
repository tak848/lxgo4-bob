import fs from "node:fs/promises";
import path from "node:path";
import { type NextRequest, NextResponse } from "next/server";

const DOCS_DIR = path.join(process.cwd(), "..", "docs");

export async function GET(
  _req: NextRequest,
  { params }: { params: Promise<{ slug: string }> },
) {
  const { slug } = await params;
  const filePath = path.join(DOCS_DIR, `${slug}.md`);

  try {
    const content = await fs.readFile(filePath, "utf-8");
    return NextResponse.json({ content });
  } catch {
    return NextResponse.json({ error: "Not found" }, { status: 404 });
  }
}
