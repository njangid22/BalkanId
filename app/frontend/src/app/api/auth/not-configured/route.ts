import { NextResponse } from "next/server";

export function GET() {
  return NextResponse.json(
    {
      error: "OAuth not configured",
      message: "Set NEXT_PUBLIC_API_URL to your backend before using this route."
    },
    { status: 500 }
  );
}
