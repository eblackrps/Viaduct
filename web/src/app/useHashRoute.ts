import { startTransition, useEffect, useState } from "react";
import { defaultRoute, getRouteHref, normalizeRoutePath, type AppRoutePath } from "./navigation";

function readCurrentRoute(): AppRoutePath {
  if (typeof window === "undefined") {
    return defaultRoute;
  }

  return normalizeRoutePath(window.location.hash);
}

function syncHash(path: AppRoutePath) {
  const href = getRouteHref(path);
  if (window.location.hash !== href) {
    window.history.replaceState(null, "", href);
  }
}

export function useHashRoute() {
  const [path, setPath] = useState<AppRoutePath>(() => readCurrentRoute());

  useEffect(() => {
    const applyRoute = () => {
      const nextPath = readCurrentRoute();
      syncHash(nextPath);
      startTransition(() => {
        setPath(nextPath);
      });
    };

    applyRoute();
    window.addEventListener("hashchange", applyRoute);
    return () => window.removeEventListener("hashchange", applyRoute);
  }, []);

  return { path };
}
