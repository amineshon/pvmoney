"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";

const mem = new Map<string, string>();

export function DebtLogo({
  debtId,
  hasLogo,
  name,
  color,
  size = "md",
  rev,
}: {
  debtId: string;
  hasLogo?: boolean;
  name: string;
  color?: string;
  size?: "sm" | "md" | "lg";
  rev?: string;
}) {
  const dim = size === "lg" ? "h-20 w-20 text-2xl" : size === "sm" ? "h-10 w-10 text-sm" : "h-14 w-14 text-lg";
  const key = hasLogo ? `${debtId}:${rev || ""}` : "";
  const [src, setSrc] = useState<string | null>(key && mem.has(key) ? mem.get(key)! : null);

  useEffect(() => {
    if (!hasLogo || !debtId) {
      setSrc(null);
      return;
    }
    const cacheKey = `${debtId}:${rev || ""}`;
    const hit = mem.get(cacheKey);
    if (hit) {
      setSrc(hit);
      return;
    }
    let alive = true;
    let created = "";
    api
      .debtLogo(debtId)
      .then((blob) => {
        if (!blob || !alive) return;
        created = URL.createObjectURL(blob);
        mem.set(cacheKey, created);
        setSrc(created);
      })
      .catch(() => {
        if (alive) setSrc(null);
      });
    return () => {
      alive = false;
    };
  }, [debtId, hasLogo, rev]);

  const letter = (name || "?").trim().charAt(0) || "د";

  if (src) {
    return (
      <img
        src={src}
        alt=""
        className={`${dim} shrink-0 rounded-2xl object-cover ring-1 ring-white/10`}
      />
    );
  }

  return (
    <div
      className={`${dim} flex shrink-0 items-center justify-center rounded-2xl font-black text-white shadow-inner ring-1 ring-white/10`}
      style={{ background: color || "#fb7185" }}
    >
      {letter}
    </div>
  );
}
