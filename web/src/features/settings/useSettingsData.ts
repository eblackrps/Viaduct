import { useEffect, useRef, useState } from "react";
import { describeError, getAbout, getCurrentTenant, type ErrorDisplay } from "../../api";
import type { AboutResponse, CurrentTenant } from "../../types";

interface SettingsDataErrors {
  about?: ErrorDisplay;
  currentTenant?: ErrorDisplay;
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
          about:
            aboutResult.status === "rejected"
              ? describeError(aboutResult.reason, {
                  scope: "build metadata",
                  fallback: "Unable to load build metadata.",
                })
              : undefined,
          currentTenant:
            currentTenantResult.status === "rejected"
              ? describeError(currentTenantResult.reason, {
                  scope: "tenant context",
                  fallback: "Unable to load tenant context.",
                })
              : undefined,
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
