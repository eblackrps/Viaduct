import { useEffect, useRef, useState } from "react";
import { getAbout, getCurrentTenant } from "../../api";
import type { AboutResponse, CurrentTenant } from "../../types";

interface SettingsDataErrors {
  about?: string;
  currentTenant?: string;
}

export interface SettingsDataState {
  about: AboutResponse | null;
  currentTenant: CurrentTenant | null;
  loading: boolean;
  errors: SettingsDataErrors;
}

export function useSettingsData(): SettingsDataState {
  const [about, setAbout] = useState<AboutResponse | null>(null);
  const [currentTenant, setCurrentTenant] = useState<CurrentTenant | null>(null);
  const [loading, setLoading] = useState(true);
  const [errors, setErrors] = useState<SettingsDataErrors>({});
  const requestSequenceRef = useRef(0);

  useEffect(() => {
    const requestSequence = requestSequenceRef.current + 1;
    requestSequenceRef.current = requestSequence;

    Promise.allSettled([getAbout(), getCurrentTenant()])
      .then(([aboutResult, currentTenantResult]) => {
        if (requestSequence !== requestSequenceRef.current) {
          return;
        }

        const nextErrors: SettingsDataErrors = {
          about: aboutResult.status === "rejected" ? toErrorMessage("build metadata", aboutResult.reason) : undefined,
          currentTenant:
            currentTenantResult.status === "rejected" ? toErrorMessage("tenant context", currentTenantResult.reason) : undefined,
        };

        if (aboutResult.status === "fulfilled") {
          setAbout(aboutResult.value);
        }
        if (currentTenantResult.status === "fulfilled") {
          setCurrentTenant(currentTenantResult.value);
        }

        setErrors(nextErrors);
      })
      .finally(() => {
        if (requestSequence === requestSequenceRef.current) {
          setLoading(false);
        }
      });
  }, []);

  return {
    about,
    currentTenant,
    loading,
    errors,
  };
}

function toErrorMessage(scope: string, reason: unknown): string {
  if (reason instanceof Error && reason.message.trim() !== "") {
    return `Unable to load ${scope}: ${reason.message}`;
  }
  return `Unable to load ${scope}.`;
}
