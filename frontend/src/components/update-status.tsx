import { useEffect, useState } from "react";
import { IconDownload } from "@tabler/icons-react";
import { Button } from "@/components/ui/button";
import { useSettingsStore } from "@/stores/settings-store";
import { ApplyUpdate, GetPendingUpdate, RefreshPendingUpdate } from "../../bindings/github.com/focusd-so/focusd/internal/updater/service";
import type { UpdateInfo } from "../../bindings/github.com/focusd-so/focusd/internal/updater/models";

export function UpdateStatus() {
  const autoUpdate = useSettingsStore((state) => state.autoUpdate);
  const [pendingUpdate, setPendingUpdate] = useState<UpdateInfo | null>(null);
  const [isApplying, setIsApplying] = useState(false);

  useEffect(() => {
    if (autoUpdate) {
      setPendingUpdate(null);
      return;
    }

    let active = true;
    const handlePendingUpdate = (event: Event) => {
      if (!active) {
        return;
      }

      setPendingUpdate((event as CustomEvent<UpdateInfo | null>).detail ?? null);
    };

    window.addEventListener("update:available", handlePendingUpdate);

    const loadPendingUpdate = async () => {
      try {
        const currentPending = await GetPendingUpdate();
        if (active) {
          setPendingUpdate(currentPending);
        }

        const latestPending = await RefreshPendingUpdate();
        if (active) {
          setPendingUpdate(latestPending);
        }
      } catch (error) {
        console.error("Failed to fetch pending update:", error);
      }
    };

    void loadPendingUpdate();

    return () => {
      active = false;
      window.removeEventListener("update:available", handlePendingUpdate);
    };
  }, [autoUpdate]);

  if (autoUpdate || pendingUpdate == null) {
    return null;
  }

  const handleApplyUpdate = async () => {
    try {
      setIsApplying(true);
      await ApplyUpdate();
    } catch (error) {
      console.error("Failed to apply update:", error);
      setIsApplying(false);
    }
  };

  return (
    <Button
      variant="outline"
      size="sm"
      className="h-7 border-emerald-500/40 bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20 hover:text-emerald-300 text-xs gap-1.5"
      onClick={handleApplyUpdate}
      disabled={isApplying}
    >
      <IconDownload className="w-3.5 h-3.5" />
      <span>{isApplying ? "Updating..." : `Update ${pendingUpdate.version}`}</span>
    </Button>
  );
}
